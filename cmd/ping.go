package cmd

import (
	"fmt"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping all configured servers",

	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		for _, server := range cfg.Servers {

			// za sada fake ping
			fmt.Printf(
				"[OK] %s (%s)\n",
				server.Name,
				server.Host,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
