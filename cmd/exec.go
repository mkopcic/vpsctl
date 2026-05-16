package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/logger"
	"github.com/mkopcic/vpsctl/internal/output"
	sshclient "github.com/mkopcic/vpsctl/internal/ssh"
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

func runOnServer(server config.Server, command string) error {
	out, err := sshclient.RunCommand(
		sshclient.Config{
			User:     server.User,
			Host:     server.Host,
			Port:     server.Port,
			Key:      server.Key,
			Password: server.Password,
		},
		command,
	)

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
