package cmd

import (
	"fmt"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/output"
	sshclient "github.com/mkopcic/vpsctl/internal/ssh"
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
	Use:   "update [server|all]",
	Short: "Run system update on one or more servers",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		target := args[0]

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
				return runOnServer(server, updateCmd_script)
			}
			return fmt.Errorf("server not found: %s", target)
		}

		for _, server := range cfg.Servers {
			output.Info("updating %s...", server.Name)
			out, err := sshclient.RunCommand(
				sshclient.Config{
					User:     server.User,
					Host:     server.Host,
					Port:     server.Port,
					Key:      server.Key,
					Password: server.Password,
				},
				updateCmd_script,
			)
			if err != nil {
				output.Fail("%s — %v", server.Name, err)
				continue
			}
			output.OK("%s\n%s", server.Name, out)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
