package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/ops"
	"github.com/mkopcic/vpsctl/internal/output"
)

var execCmd = &cobra.Command{
	Use:   "exec [server|all] [command]",
	Short: "Execute command on one or more servers",
	Args:  cobra.ExactArgs(2),

	// SilenceUsage sprjecava Cobru da prikaze Usage kad SSH komanda faila
	// SSH execution failure nije usage greska — Usage bi zbunjivao korisnika
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		command := args[1]
		timeout, _ := cmd.Flags().GetDuration("timeout")

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		servers, err := ops.Targets(cfg, target)
		if err != nil {
			return err
		}

		// Jedan timeout za cijelu operaciju — i za "all"
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()

		var wg sync.WaitGroup
		for _, server := range servers {
			wg.Add(1)
			go func(server config.Server) {
				defer wg.Done()
				runOnServer(ctx, server, command)
			}(server)
		}
		wg.Wait()
		return nil
	},
}

// cliRunner zapisuje jobove sa source="cli"
var cliRunner = ops.Runner{Source: "cli"}

// runOnServer izvrsava SSH komandu na jednom serveru i ispisuje OK/FAIL.
// Ne vraca error Cobri — SSH execution failure nije razlog za prikaz Usage.
// History i log pise ops.Runner.
func runOnServer(ctx context.Context, server config.Server, command string) {
	res := cliRunner.Run(ctx, server, command)

	if !res.OK() {
		// Prepoznaj poznate greske i daj jasnu poruku
		output.Fail("%s — %s", server.Name, friendlyError(res.Output, res.Err.Error()))
		if res.Output != "" {
			fmt.Print(res.Output)
		}
		return
	}

	output.OK("%s\n%s", server.Name, res.Output)
}

// runSequential vrti komandu server po server, svaki sa svojim timeoutom.
// Za operacije koje mijenjaju server (update, restart) — ne zelimo ih
// pokrenuti na svima odjednom. announce ispisuje sto se upravo radi.
func runSequential(ctx context.Context, servers []config.Server, timeout time.Duration, command string, announce func(config.Server)) error {
	for _, server := range servers {
		if ctx.Err() != nil {
			return ctx.Err() // Ctrl+C — ne kreci na sljedeci server
		}
		announce(server)
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		runOnServer(runCtx, server, command)
		cancel()
	}
	return nil
}

// friendlyError prepoznaje poznate sistemske greske i vraca jasnu poruku.
// serverOutput je stdout/stderr sa servera, sshErr je Go SSH error string.
func friendlyError(serverOutput, sshErr string) string {
	o := strings.ToLower(serverOutput)

	switch {
	case strings.Contains(o, "interactive authentication required"):
		return "nema sudo permisije — dodaj NOPASSWD u sudoers za ovog usera"
	case strings.Contains(o, "unit") && strings.Contains(o, "not found"):
		return "servis ne postoji na ovom serveru"
	case strings.Contains(o, "not found") || strings.Contains(o, "not recognized"):
		return "komanda nije pronadjena na serveru"
	case strings.Contains(o, "permission denied"):
		return "permission denied"
	case strings.Contains(o, "failed to connect") || strings.Contains(o, "connection refused"):
		return "konekcija odbijena"
	default:
		return sshErr
	}
}

func init() {
	execCmd.Flags().Duration("timeout", ops.ExecTimeout, "Max trajanje komande (npr. 30s, 5m)")
	rootCmd.AddCommand(execCmd)
}
