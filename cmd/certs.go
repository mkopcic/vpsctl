package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/mkopcic/vpsctl/internal/certs"
	"github.com/mkopcic/vpsctl/internal/config"
	"github.com/mkopcic/vpsctl/internal/output"
	"github.com/spf13/cobra"
)

var certsCmd = &cobra.Command{
	Use:   "certs [domena...]",
	Short: "Check TLS certificate expiry for domains",
	Long: `Provjeri TLS certifikate: datum isteka, lanac do root CA i pokriva li cert domenu.

Bez argumenata provjerava sve domene iz "domains:" u servers.yaml.
Izlazi s kodom 1 ako ijedan cert treba paznju — pogodno za cron.`,
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		warnDays, _ := cmd.Flags().GetInt("warn")

		domains := args
		if len(domains) == 0 {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			domains = configDomains(cfg)
		}
		if len(domains) == 0 {
			return errors.New(`nema domena — dodaj "domains:" serveru u servers.yaml ili ih navedi kao argumente`)
		}

		results := certs.Checker{}.CheckAll(cmd.Context(), domains, 10)

		// Najhitnije prvo: greske, pa po broju dana do isteka
		slices.SortStableFunc(results, func(a, b certs.Result) int {
			return cmp.Or(
				cmp.Compare(b.Status(warnDays), a.Status(warnDays)),
				cmp.Compare(a.DaysLeft, b.DaysLeft),
			)
		})

		width := 0
		for _, r := range results {
			width = max(width, len(r.Domain))
		}

		problems := 0
		for _, r := range results {
			switch r.Status(warnDays) {
			case certs.OK:
				output.OK("%-*s %4d dana  %s  %s", width, r.Domain, r.DaysLeft, r.NotAfter.Format("2006-01-02"), r.Issuer)
			case certs.Warn:
				problems++
				output.Warn("%-*s %4d dana  %s  %s", width, r.Domain, r.DaysLeft, r.NotAfter.Format("2006-01-02"), r.Issuer)
			case certs.Fail:
				problems++
				if r.NotAfter.IsZero() {
					output.Fail("%-*s %s", width, r.Domain, certs.Explain(r.Err))
				} else {
					output.Fail("%-*s %4d dana  %s  %s", width, r.Domain, r.DaysLeft, r.NotAfter.Format("2006-01-02"), certs.Explain(r.Err))
				}
			}
		}

		if problems > 0 {
			return fmt.Errorf("%d od %d certifikata treba paznju", problems, len(results))
		}
		return nil
	},
}

// configDomains skuplja domene svih servera, bez duplikata, u redoslijedu iz YAML-a
func configDomains(cfg *config.Config) []string {
	seen := map[string]bool{}
	var domains []string
	for _, s := range cfg.Servers {
		for _, d := range s.Domains {
			if !seen[d] {
				seen[d] = true
				domains = append(domains, d)
			}
		}
	}
	return domains
}

func init() {
	certsCmd.Flags().Int("warn", 21, "Upozori ako cert istice za manje od N dana")
	rootCmd.AddCommand(certsCmd)
}
