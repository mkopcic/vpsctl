// Package ops su operacije nad serverima koje dijele CLI i API:
// SSH izvrsavanje + job history + log na jednom mjestu.
// cmd/ i internal/api/ samo biraju servere i prikazuju rezultat.
package ops

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/logger"
	sshclient "github.com/mkopcic/vpsctl/internal/ssh"
	"github.com/mkopcic/vpsctl/internal/store"
)

// Defaultni timeouti po vrsti operacije — CLI ih nudi kao default flaga,
// API ih koristi izravno.
const (
	ExecTimeout    = 30 * time.Second
	UpdateTimeout  = 15 * time.Minute // apt upgrade zna trajati minutama
	RestartTimeout = time.Minute
)

// UpdateScript detektira package manager automatski
const UpdateScript = `
if command -v apt-get >/dev/null 2>&1; then
    apt-get update && apt-get upgrade -y
elif command -v yum >/dev/null 2>&1; then
    yum update -y
elif command -v apk >/dev/null 2>&1; then
    apk update && apk upgrade
else
    echo "unknown package manager"
    exit 1
fi
`

// serviceName pokriva znakove koje systemd dopusta u imenu unita.
// Ime ide u shell komandu, pa sve ostalo (razmak, ;, $, |...) odbijamo.
var serviceName = regexp.MustCompile(`^[A-Za-z0-9@._:-]+$`)

// RestartCommand slaze komandu za restart systemd servisa.
// is-active nakon restarta potvrdi je li servis stvarno gore.
func RestartCommand(service string) (string, error) {
	if !serviceName.MatchString(service) {
		return "", fmt.Errorf("neispravno ime servisa: %q", service)
	}
	return fmt.Sprintf("systemctl restart %s && systemctl is-active %s", service, service), nil
}

// Targets vraca servere za zadani target: "all" ili ime jednog servera.
func Targets(cfg *config.Config, target string) ([]config.Server, error) {
	if target == "all" {
		return cfg.Servers, nil
	}
	s, ok := cfg.Find(target)
	if !ok {
		return nil, fmt.Errorf("server not found: %s", target)
	}
	return []config.Server{s}, nil
}

// SSHConfig prevodi server iz YAML-a u SSH config
func SSHConfig(server config.Server) sshclient.Config {
	return sshclient.Config{
		User:     server.User,
		Host:     server.Host,
		Port:     server.Port,
		Key:      server.Key,
		Password: server.Password,
	}
}

// Result je ishod jedne komande na jednom serveru.
type Result struct {
	Server string
	Output string // stdout + stderr
	// ExitCode je -1 ako komanda nije ni pokrenuta (konekcija, auth, timeout)
	ExitCode int
	Err      error
}

func (r Result) OK() bool {
	return r.Err == nil
}

// Runner izvrsava komande i svaku zapisuje u history i log.
type Runner struct {
	// Source ide u job history: "cli" ili "api"
	Source string
	// NoPrompt: bez interaktivnog upisa passworda (serve mod nema terminal)
	NoPrompt bool
}

// Run izvrsava komandu na jednom serveru.
// Trajanje ogranicava ctx — pozivatelj bira timeout.
func (r Runner) Run(ctx context.Context, server config.Server, command string) Result {
	start := time.Now()

	sshCfg := SSHConfig(server)
	sshCfg.NoPrompt = r.NoPrompt

	res, err := sshclient.RunCommand(ctx, sshCfg, command)
	out := res.Output()

	result := Result{Server: server.Name, Output: out, Err: err}
	if err != nil {
		// Exit code postoji samo ako je komanda stvarno pokrenuta
		result.ExitCode = -1
		if exitErr, ok := errors.AsType[*sshclient.ExitError](err); ok {
			result.ExitCode = exitErr.Code
		}
	}

	// History i log se pisu bez obzira na uspjeh. Greska pri zapisu ne smije
	// srusiti operaciju koja je na serveru vec izvrsena.
	job := store.NewJob(server.Name, command, out, "", r.Source, err == nil, start)
	logEntry := out
	if err != nil {
		job.Error = err.Error()
		logEntry = "FAIL: " + err.Error() + " | " + out
	}
	store.Save(job)
	logger.Log(server.Name, command, logEntry)

	return result
}
