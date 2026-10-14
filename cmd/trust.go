package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/output"
	sshclient "github.com/mkopcic/vpsctl/internal/ssh"
	"github.com/spf13/cobra"
)

var trustCmd = &cobra.Command{
	Use:   "trust [server|all]",
	Short: "Add server host key to known_hosts after confirming its fingerprint",
	Long: `Dohvati host kljuc servera, pokazi fingerprint i nakon potvrde ga dodaj u known_hosts.

Usporedi fingerprint s onim na serveru prije nego potvrdis:
  ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		servers := cfg.Servers
		if args[0] != "all" {
			s, ok := cfg.Find(args[0])
			if !ok {
				return fmt.Errorf("server not found: %s", args[0])
			}
			servers = []config.Server{s}
		}

		in := bufio.NewReader(os.Stdin)
		failed := 0
		for _, s := range servers {
			if err := trustServer(cmd.Context(), s, in, yes); err != nil {
				output.Fail("%s — %s", s.Name, err)
				failed++
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d od %d servera nije dodano", failed, len(servers))
		}
		return nil
	},
}

func trustServer(ctx context.Context, s config.Server, in *bufio.Reader, yes bool) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	hk, err := sshclient.FetchHostKey(ctx, sshConfig(s))
	if err != nil {
		return err
	}

	if hk.Status == nil {
		output.OK("%s — vec u known_hosts (%s)", s.Name, hk.Fingerprint)
		return nil
	}

	// Promijenjen kljuc nikad ne zamjenjujemo automatski — to mora biti svjesna odluka
	if changed, ok := errors.AsType[*sshclient.HostKeyChangedError](hk.Status); ok {
		return changed
	}

	output.Info("%s  %s  %s  %s", s.Name, hk.Addr, hk.Key.Type(), hk.Fingerprint)

	if !yes {
		fmt.Print("Vjerujes li ovom kljucu? [y/N] ")
		answer, _ := in.ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" && a != "d" && a != "da" {
			return errors.New("preskoceno")
		}
	}

	if err := sshclient.AddKnownHost(hk.Addr, hk.Key); err != nil {
		return err
	}

	path, _ := sshclient.KnownHostsPath()
	output.OK("%s — dodano u %s", s.Name, path)
	return nil
}

func init() {
	trustCmd.Flags().BoolP("yes", "y", false, "Ne pitaj za potvrdu (oprez: preskace provjeru fingerprinta)")
	rootCmd.AddCommand(trustCmd)
}
