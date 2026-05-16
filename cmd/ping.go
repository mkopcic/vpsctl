package cmd

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/output"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "TCP ping all configured servers",

	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		var wg sync.WaitGroup

		for _, server := range cfg.Servers {

			wg.Add(1)

			go func(server config.Server) {
				defer wg.Done()

				addr := fmt.Sprintf("%s:%d", server.Host, server.Port)

				start := time.Now()
				conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
				latency := time.Since(start)

				if err != nil {
					output.Fail("%-20s %s", server.Name, server.Host)
					return
				}

				conn.Close()

				output.OK("%-20s %s — %dms", server.Name, server.Host, latency.Milliseconds())
			}(server)
		}

		wg.Wait()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
