package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/logger"
	"github.com/mkopcic/vpsctl/internal/output"
	sshclient "github.com/mkopcic/vpsctl/internal/ssh"
	"github.com/mkopcic/vpsctl/internal/store"
)

var execCmd = &cobra.Command{
	Use:   "exec [server|all] [command]",
	Short: "Execute command on one or more servers",
	Args:  cobra.ExactArgs(2),

	// SilenceUsage sprjecava Cobru da prikaze Usage kad SSH komanda faila
	// SSH execution failure nije usage greska — Usage bi zbunjivao korisnika
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		command := args[1]
		timeout, _ := cmd.Flags().GetDuration("timeout")

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Jedan timeout za cijelu operaciju — i za "all"
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()

		if target != "all" {
			for _, server := range cfg.Servers {
				if server.Name != target {
					continue
				}
				runOnServer(ctx, server, command)
				return nil
			}
			return fmt.Errorf("server not found: %s", target)
		}

		var wg sync.WaitGroup
		for _, server := range cfg.Servers {
			wg.Add(1)
			go func(server config.Server) {
				defer wg.Done()
				runOnServer(ctx, server, command)
			}(server)
		}
		wg.Wait()
		return nil
	},
}

// sshConfig prevodi server iz YAML-a u SSH config
func sshConfig(server config.Server) sshclient.Config {
	return sshclient.Config{
		User:     server.User,
		Host:     server.Host,
		Port:     server.Port,
		Key:      server.Key,
		Password: server.Password,
	}
}

// runOnServer izvrsava SSH komandu na jednom serveru.
// Ne vraca error Cobri — sama handla prikaz OK/FAIL + log + DB.
// SSH execution failure nije razlog za prikaz Usage.
func runOnServer(ctx context.Context, server config.Server, command string) {
	start := time.Now()

	res, err := sshclient.RunCommand(ctx, sshConfig(server), command)
	out := res.Output()

	// Spremi u SQLite bez obzira na uspjeh/neuspjeh
	job := store.NewJob(server.Name, command, out, "", "cli", err == nil, start)
	if err != nil {
		job.Error = err.Error()
	}
	store.Save(job)

	if err != nil {
		// Prepoznaj poznate greske i daj jasnu poruku
		msg := friendlyError(out, err.Error())
		output.Fail("%s — %s", server.Name, msg)
		if out != "" {
			fmt.Print(out)
		}
		logger.Log(server.Name, command, "FAIL: "+msg+" | "+out)
		return
	}

	output.OK("%s\n%s", server.Name, out)
	logger.Log(server.Name, command, out)
}

// friendlyError prepoznaje poznate sistemske greske i vraca jasnu poruku.
// serverOutput je stdout/stderr sa servera, sshErr je Go SSH error string.
func friendlyError(serverOutput, sshErr string) string {
	o := strings.ToLower(serverOutput)

	switch {
	case strings.Contains(o, "interactive authentication required"):
		return "nema sudo permisije — dodaj NOPASSWD u sudoers za ovog usera"
	case strings.Contains(o, "unit") && strings.Contains(o, "not found"):
		return "servis ne postoji na ovom serveru"
	case strings.Contains(o, "not found") || strings.Contains(o, "not recognized"):
		return "komanda nije pronadjena na serveru"
	case strings.Contains(o, "permission denied"):
		return "permission denied"
	case strings.Contains(o, "failed to connect") || strings.Contains(o, "connection refused"):
		return "konekcija odbijena"
	default:
		return sshErr
	}
}

func init() {
	execCmd.Flags().Duration("timeout", 30*time.Second, "Max trajanje komande (npr. 30s, 5m)")
	rootCmd.AddCommand(execCmd)
}
