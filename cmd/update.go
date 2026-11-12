package cmd

import (
	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/ops"
	"github.com/mkopcic/vpsctl/internal/output"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:          "update [server|all]",
	Short:        "Run system update on one or more servers",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, _ := cmd.Flags().GetDuration("timeout")

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		servers, err := ops.Targets(cfg, args[0])
		if err != nil {
			return err
		}

		return runSequential(cmd.Context(), servers, timeout, ops.UpdateScript, func(s config.Server) {
			output.Info("updating %s...", s.Name)
		})
	},
}

func init() {
	updateCmd.Flags().Duration("timeout", ops.UpdateTimeout, "Max trajanje updatea po serveru")
	rootCmd.AddCommand(updateCmd)
}
