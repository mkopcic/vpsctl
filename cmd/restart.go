package cmd

import (
	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/ops"
	"github.com/mkopcic/vpsctl/internal/output"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:          "restart [server|all] [service]",
	Short:        "Restart a systemd service on one or more servers",
	Args:         cobra.ExactArgs(2),
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[1]
		timeout, _ := cmd.Flags().GetDuration("timeout")

		command, err := ops.RestartCommand(service)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		servers, err := ops.Targets(cfg, args[0])
		if err != nil {
			return err
		}

		return runSequential(cmd.Context(), servers, timeout, command, func(s config.Server) {
			output.Info("restarting %s on %s...", service, s.Name)
		})
	},
}

func init() {
	restartCmd.Flags().Duration("timeout", ops.RestartTimeout, "Max trajanje restarta po serveru")
	rootCmd.AddCommand(restartCmd)
}
