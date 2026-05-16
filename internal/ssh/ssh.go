package sshclient

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

const defaultTimeout = 30 * time.Second

type Config struct {
	User     string
	Host     string
	Port     int
	Key      string
	Password string
}

func RunCommand(cfg Config, command string) (string, error) {
	auth := buildAuth(cfg)

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	type result struct {
		output string
		err    error
	}

	ch := make(chan result, 1)

	go func() {
		out, err := session.CombinedOutput(command)
		ch <- result{string(out), err}
	}()

	select {
	case r := <-ch:
		return r.output, r.err
	case <-time.After(defaultTimeout):
		return "", fmt.Errorf("timeout nakon %s", defaultTimeout)
	}
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
