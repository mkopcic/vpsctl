package sshclient

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

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
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

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

	output, err := session.CombinedOutput(command)
	return string(output), err
}

func buildAuth(cfg Config) []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// pokušaj key auth ako je key definiran i postoji
	if cfg.Key != "" {
		if signer, err := loadKey(cfg.Key); err == nil {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}

	// password iz YAML-a — proba obje metode jer serveri koriste različite
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

	// interaktivni prompt kao fallback
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
