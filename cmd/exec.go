package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/mkopcic/vpsctl/internal/config"
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

		// SINGLE SERVER
		if target != "all" {

			for _, server := range cfg.Servers {

				if server.Name != target {
					continue
				}

				output, err := sshclient.RunCommand(
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
					return err
				}

				fmt.Printf(
					"[OK] %s\n%s\n",
					server.Name,
					output,
				)

				return nil
			}

			return fmt.Errorf(
				"server not found: %s",
				target,
			)
		}

		// ALL SERVERS — OVDJE DOLAZI GO
		var wg sync.WaitGroup

		for _, server := range cfg.Servers {

			wg.Add(1)

			go func(server config.Server) {

				defer wg.Done()

				output, err := sshclient.RunCommand(
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

					fmt.Printf(
						"[FAIL] %s -> %v\n",
						server.Name,
						err,
					)

					return
				}

				fmt.Printf(
					"[OK] %s\n%s\n",
					server.Name,
					output,
				)

			}(server)
		}

		wg.Wait()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
