package sshclient

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// --- lazni SSH server za testove ---

type testServer struct {
	cfg         Config
	hostKey     ssh.Signer
	connections atomic.Int32 // koliko TCP konekcija je server primio
}

// startServer pokrece SSH server na 127.0.0.1 s random portom.
// Prihvaca user "test" / password "tajna" i par laznih komandi (vidi fakeShell).
func startServer(t *testing.T) *testServer {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	serverCfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "test" && string(pass) == "tajna" {
				return nil, nil
			}
			return nil, errors.New("krivi password")
		},
	}
	serverCfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	srv := &testServer{
		cfg:     Config{User: "test", Host: host, Port: port, Password: "tajna"},
		hostKey: signer,
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			srv.connections.Add(1)
			go serveConn(conn, serverCfg)
		}
	}()

	return srv
}

func serveConn(conn net.Conn, cfg *ssh.ServerConfig) {
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		conn.Close()
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	for nc := range chans {
		if nc.ChannelType() != "session" {
			nc.Reject(ssh.UnknownChannelType, "")
			continue
		}
		ch, chReqs, err := nc.Accept()
		if err != nil {
			continue
		}
		go serveSession(ch, chReqs)
	}
}

func serveSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()

	for req := range reqs {
		if req.Type != "exec" {
			req.Reply(false, nil)
			continue
		}

		var payload struct{ Command string }
		ssh.Unmarshal(req.Payload, &payload)
		req.Reply(true, nil)

		code, ok := fakeShell(ch, reqs, payload.Command)
		if !ok {
			return // klijent je prekinuo — nema exit statusa
		}
		ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(code)}))
		return
	}
}

// fakeShell glumi par komandi s predvidljivim outputom
func fakeShell(ch ssh.Channel, reqs <-chan *ssh.Request, command string) (exitCode int, ok bool) {
	switch command {
	case "hello":
		io.WriteString(ch, "out\n")
		io.WriteString(ch.Stderr(), "err\n")
		return 0, true
	case "fail":
		io.WriteString(ch.Stderr(), "boom\n")
		return 3, true
	case "hang":
		// Visi dok klijent ne zatvori kanal (timeout / Ctrl+C na klijentu).
		// Ne citamo ch — klijent bez stdina odmah posalje EOF. Kad se kanal
		// zatvori, zatvori se i reqs (usput pojedemo i "signal" request).
		for range reqs {
		}
		return 0, false
	}
	io.WriteString(ch.Stderr(), "command not found\n")
	return 127, true
}

// --- helperi ---

// useKnownHosts preusmjerava known_hosts u temp fajl, da test ne dira ~/.ssh
func useKnownHosts(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "known_hosts")
	t.Setenv("VPSCTL_KNOWN_HOSTS", path)
	return path
}

// trusted vraca server ciji je kljuc vec u known_hosts
func trusted(t *testing.T) *testServer {
	t.Helper()
	useKnownHosts(t)
	srv := startServer(t)
	if err := AddKnownHost(srv.cfg.addr(), srv.hostKey.PublicKey()); err != nil {
		t.Fatal(err)
	}
	return srv
}

// --- testovi ---

func TestRun_StdoutStderrExitCode(t *testing.T) {
	srv := trusted(t)

	tests := []struct {
		name       string
		command    string
		wantStdout string
		wantStderr string
		wantCode   int
	}{
		{"uspjeh", "hello", "out\n", "err\n", 0},
		{"exit code", "fail", "", "boom\n", 3},
		{"nepoznata komanda", "nema-me", "", "command not found\n", 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := RunCommand(t.Context(), srv.cfg, tt.command)

			if res.Stdout != tt.wantStdout || res.Stderr != tt.wantStderr {
				t.Errorf("stdout=%q stderr=%q, want %q / %q", res.Stdout, res.Stderr, tt.wantStdout, tt.wantStderr)
			}
			if res.ExitCode != tt.wantCode {
				t.Errorf("ExitCode = %d, want %d", res.ExitCode, tt.wantCode)
			}

			exitErr, isExit := errors.AsType[*ExitError](err)
			switch {
			case tt.wantCode == 0 && err != nil:
				t.Errorf("neocekivana greska: %v", err)
			case tt.wantCode != 0 && (!isExit || exitErr.Code != tt.wantCode):
				t.Errorf("err = %v, want *ExitError{%d}", err, tt.wantCode)
			}
		})
	}
}

func TestRun_Timeout(t *testing.T) {
	srv := trusted(t)

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := RunCommand(ctx, srv.cfg, "hang")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("prekid je trajao %s — context nije prekinuo komandu", elapsed)
	}
}

func TestClient_ReusesConnection(t *testing.T) {
	srv := trusted(t)

	client, err := Dial(t.Context(), srv.cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	for range 5 {
		if _, err := client.Run(t.Context(), "hello"); err != nil {
			t.Fatal(err)
		}
	}

	if n := srv.connections.Load(); n != 1 {
		t.Errorf("server je primio %d konekcija, want 1", n)
	}
}

// TestDial_PromptPausesDeadline: korisnik tipka password dulje od dialTimeout.
// Bez pauseDeadline handshake pada s "i/o timeout" (bug s mellon-dev).
func TestDial_PromptPausesDeadline(t *testing.T) {
	srv := trusted(t)

	old := dialTimeout
	dialTimeout = 100 * time.Millisecond
	t.Cleanup(func() { dialTimeout = old })

	kh, err := loadKnownHosts()
	if err != nil {
		t.Fatal(err)
	}

	hs := &handshake{}
	clientConfig := &ssh.ClientConfig{
		User: srv.cfg.User,
		Auth: []ssh.AuthMethod{ssh.PasswordCallback(func() (string, error) {
			hs.pauseDeadline()
			time.Sleep(3 * dialTimeout) // "korisnik" sporo tipka
			return srv.cfg.Password, nil
		})},
		HostKeyCallback: kh.check,
	}

	conn, err := dial(t.Context(), srv.cfg.addr(), clientConfig, kh, hs)
	if err != nil {
		t.Fatalf("handshake pao dok je korisnik tipkao: %v", err)
	}
	conn.Close()
}

func TestDial_UnknownHost(t *testing.T) {
	useKnownHosts(t)
	srv := startServer(t)

	_, err := Dial(t.Context(), srv.cfg)

	unknown, ok := errors.AsType[*UnknownHostError](err)
	if !ok {
		t.Fatalf("err = %v, want *UnknownHostError", err)
	}
	if want := ssh.FingerprintSHA256(srv.hostKey.PublicKey()); unknown.Fingerprint != want {
		t.Errorf("Fingerprint = %s, want %s", unknown.Fingerprint, want)
	}
}

func TestDial_HostKeyChanged(t *testing.T) {
	useKnownHosts(t)
	srv := startServer(t)

	// U known_hosts upisemo neki drugi kljuc za isti host — simulacija MITM-a
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	other, _ := ssh.NewPublicKey(otherPub)
	if err := AddKnownHost(srv.cfg.addr(), other); err != nil {
		t.Fatal(err)
	}

	_, err := Dial(t.Context(), srv.cfg)

	if _, ok := errors.AsType[*HostKeyChangedError](err); !ok {
		t.Fatalf("err = %v, want *HostKeyChangedError", err)
	}
}

func TestFetchHostKey_ThenTrust(t *testing.T) {
	useKnownHosts(t)
	srv := startServer(t)

	hk, err := FetchHostKey(t.Context(), srv.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := errors.AsType[*UnknownHostError](hk.Status); !ok {
		t.Fatalf("Status = %v, want *UnknownHostError", hk.Status)
	}

	if err := AddKnownHost(hk.Addr, hk.Key); err != nil {
		t.Fatal(err)
	}

	// Nakon trusta: FetchHostKey kaze "poznat", a Dial + Run rade
	hk, err = FetchHostKey(t.Context(), srv.cfg)
	if err != nil || hk.Status != nil {
		t.Fatalf("nakon trusta: err=%v status=%v", err, hk.Status)
	}
	if _, err := RunCommand(t.Context(), srv.cfg, "hello"); err != nil {
		t.Fatalf("RunCommand nakon trusta: %v", err)
	}
}

func TestAddKnownHost_MissingTrailingNewline(t *testing.T) {
	path := useKnownHosts(t)

	// Postojeci fajl bez \n na kraju — nova linija se ne smije zalijepiti na nju
	existing := "example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, _ := ssh.NewPublicKey(pub)
	if err := AddKnownHost("[mellon.hr]:2222", key); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("ocekivane 2 linije, dobio %d:\n%s", len(lines), data)
	}

	// Fajl mora i dalje biti citljiv knownhosts paketu
	if _, err := knownhosts.New(path); err != nil {
		t.Errorf("known_hosts vise nije validan: %v", err)
	}
}
