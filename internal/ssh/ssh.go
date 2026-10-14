package sshclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// dialTimeout vrijedi za TCP connect + SSH handshake (kex + auth).
// Trajanje same komande kontrolira context koji salje pozivatelj.
const dialTimeout = 10 * time.Second

type Config struct {
	User     string
	Host     string
	Port     int
	Key      string
	Password string
}

func (c Config) addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Result je ishod jedne komande.
// Stdout i stderr su odvojeni jer ce Detect logika parsirati samo stdout.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Output spaja stdout i stderr za prikaz korisniku i spremanje u history
func (r Result) Output() string {
	return r.Stdout + r.Stderr
}

// ExitError znaci da se komanda izvrsila, ali je vratila exit code != 0.
// Isti pattern kao *exec.ExitError u os/exec — konekcija je OK, komanda nije.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// Client drzi jednu SSH konekciju otvorenu.
// Svaki Run otvara novu session na istoj konekciji — handshake i auth
// rade se samo jednom, bez obzira koliko komandi pokrenemo.
type Client struct {
	conn *ssh.Client
}

// Dial otvara konekciju i provjerava host kljuc protiv known_hosts.
func Dial(ctx context.Context, cfg Config) (*Client, error) {
	kh, err := loadKnownHosts()
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            buildAuth(cfg),
		HostKeyCallback: kh.check,
	}

	conn, err := dial(ctx, cfg.addr(), clientConfig, kh)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// dial je zajednicki dio za Dial i FetchHostKey.
// Ako je kh zadan, HostKeyAlgorithms se suzi na tipove kljuceva koje vec znamo.
func dial(ctx context.Context, addr string, clientConfig *ssh.ClientConfig, kh *knownHosts) (*ssh.Client, error) {
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	if kh != nil {
		clientConfig.HostKeyAlgorithms = kh.algorithms(addr, conn.RemoteAddr())
	}

	// ssh.NewClientConn ne prima context. Ako se context prekine tijekom
	// handshakea (Ctrl+C), zatvaramo socket i handshake odmah pada.
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	// Server koji primi TCP ali ne odgovara na SSH bi inace visio zauvijek
	conn.SetDeadline(time.Now().Add(dialTimeout))

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		conn.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}

	// Ukloni deadline — konekcija ostaje otvorena za vise komandi
	conn.SetDeadline(time.Time{})

	return ssh.NewClient(c, chans, reqs), nil
}

// Run izvrsava jednu komandu u novoj sessionu.
// Vraca *ExitError ako komanda zavrsi s exit code != 0 — Result je i tada popunjen.
func (c *Client) Run(ctx context.Context, command string) (Result, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return Result{}, err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Timeout ili Ctrl+C: ubij proces i zatvori session — to odblokira session.Run
	stop := context.AfterFunc(ctx, func() {
		session.Signal(ssh.SIGKILL)
		session.Close()
	})
	defer stop()

	err = session.Run(command)
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}

	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return res, fmt.Errorf("timeout: %w", ctx.Err())
		}
		return res, fmt.Errorf("prekinuto: %w", ctx.Err())
	}

	// errors.AsType (Go 1.26) je genericka verzija errors.As — bez pomocne varijable
	if exitErr, ok := errors.AsType[*ssh.ExitError](err); ok {
		res.ExitCode = exitErr.ExitStatus()
		return res, &ExitError{Code: res.ExitCode}
	}

	return res, err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// RunCommand je precac za jednu komandu: Dial + Run + Close.
// Za vise komandi na istom serveru koristi Dial pa Run vise puta.
func RunCommand(ctx context.Context, cfg Config, command string) (Result, error) {
	client, err := Dial(ctx, cfg)
	if err != nil {
		return Result{}, err
	}
	defer client.Close()

	return client.Run(ctx, command)
}

func buildAuth(cfg Config) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if cfg.Key != "" {
		if signer, err := loadKey(cfg.Key); err == nil {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}

	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
		methods = append(methods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range questions {
				answers[i] = cfg.Password
			}
			return answers, nil
		}))
	}

	if cfg.Password == "" {
		methods = append(methods, ssh.PasswordCallback(func() (string, error) {
			fmt.Printf("Password for %s@%s: ", cfg.User, cfg.Host)
			raw, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			return string(raw), err
		}))
		methods = append(methods, ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
			fmt.Printf("Password for %s@%s: ", cfg.User, cfg.Host)
			raw, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			answers := make([]string, len(questions))
			for i := range questions {
				answers[i] = string(raw)
			}
			return answers, err
		}))
	}

	return methods
}

func loadKey(path string) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}
