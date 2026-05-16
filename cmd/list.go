package cmd

import (
	"fmt"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List servers",

	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		for _, server := range cfg.Servers {

			fmt.Printf(
				"%s (%s)\n",
				server.Name,
				server.Host,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
