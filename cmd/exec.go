package cmd

import (
	"fmt"
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

	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		command := args[1]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if target != "all" {
			for _, server := range cfg.Servers {
				if server.Name != target {
					continue
				}
				return runOnServer(server, command)
			}
			return fmt.Errorf("server not found: %s", target)
		}

		// Svi serveri paralelno
		var wg sync.WaitGroup
		for _, server := range cfg.Servers {
			wg.Add(1)
			go func(server config.Server) {
				defer wg.Done()
				runOnServer(server, command)
			}(server)
		}
		wg.Wait()
		return nil
	},
}

// runOnServer izvrsava SSH komandu na jednom serveru.
// Koriste ga exec, update i restart komande — centralno mjesto za SSH + log + DB.
func runOnServer(server config.Server, command string) error {
	start := time.Now() // za mjerenje trajanja

	out, err := sshclient.RunCommand(sshclient.Config{
		User:     server.User,
		Host:     server.Host,
		Port:     server.Port,
		Key:      server.Key,
		Password: server.Password,
	}, command)

	// Spremi u SQLite bez obzira na uspjeh/neuspjeh
	job := store.NewJob(server.Name, command, out, "", "cli", err == nil, start)
	if err != nil {
		job.Error = err.Error()
	}
	store.Save(job) // greska pri loggingu ne prekida main flow

	if err != nil {
		output.Fail("%s — %v", server.Name, err)
		logger.Log(server.Name, command, "FAIL: "+err.Error())
		return err
	}

	output.OK("%s\n%s", server.Name, out)
	logger.Log(server.Name, command, out)
	return nil
}

func init() {
	rootCmd.AddCommand(execCmd)
}
