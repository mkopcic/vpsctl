package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/output"
	"github.com/spf13/cobra"
)

const updateCmd_script = `
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

var updateCmd = &cobra.Command{
	Use:          "update [server|all]",
	Short:        "Run system update on one or more servers",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		timeout, _ := cmd.Flags().GetDuration("timeout")

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if target != "all" {
			for _, server := range cfg.Servers {
				if server.Name != target {
					continue
				}
				output.Info("updating %s...", server.Name)
				ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
				defer cancel()
				runOnServer(ctx, server, updateCmd_script)
				return nil
			}
			return fmt.Errorf("server not found: %s", target)
		}

		// Serveri idu jedan po jedan, svaki ima svoj timeout
		for _, server := range cfg.Servers {
			if cmd.Context().Err() != nil {
				return cmd.Context().Err() // Ctrl+C — ne kreci na sljedeci server
			}
			output.Info("updating %s...", server.Name)
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			runOnServer(ctx, server, updateCmd_script)
			cancel()
		}

		return nil
	},
}

func init() {
	// apt upgrade zna trajati minutama — 30s kao kod exec bi ga ubio na pola
	updateCmd.Flags().Duration("timeout", 15*time.Minute, "Max trajanje updatea po serveru")
	rootCmd.AddCommand(updateCmd)
}
