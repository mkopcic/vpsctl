package cmd

import (
	"fmt"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/output"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:          "restart [server|all] [service]",
	Short:        "Restart a systemd service on one or more servers",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		service := args[1]
		// is-active nakon restarta potvrdi je li servis gore
		command := fmt.Sprintf("systemctl restart %s && systemctl is-active %s", service, service)

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if target != "all" {
			for _, server := range cfg.Servers {
				if server.Name != target {
					continue
				}
				output.Info("restarting %s on %s...", service, server.Name)
				runOnServer(server, command)
				return nil
			}
			return fmt.Errorf("server not found: %s", target)
		}

		for _, server := range cfg.Servers {
			output.Info("restarting %s on %s...", service, server.Name)
			runOnServer(server, command)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
