package cmd

import (
	"fmt"

	"github.com/mkopcic/vpsctl/internal/store"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history [server]",
	Short: "Show job history from database",
	Args:  cobra.MaximumNArgs(1), // opcionalni argument: ime servera

	RunE: func(cmd *cobra.Command, args []string) error {
		// Dohvati --limit flag (default 20)
		limit, _ := cmd.Flags().GetInt("limit")

		var jobs []store.Job
		var err error

		if len(args) == 1 {
			// vpsctl history mellon
			jobs, err = store.ListByServer(args[0], limit)
		} else {
			// vpsctl history
			jobs, err = store.List(limit)
		}

		if err != nil {
			return err
		}

		if len(jobs) == 0 {
			fmt.Println("No jobs found.")
			return nil
		}

		// Ispisi svaki job u citljivom formatu
		for _, j := range jobs {
			status := "OK  "
			if !j.OK {
				status = "FAIL"
			}

			fmt.Printf("[%s] [%s] %-20s %dms — %s\n",
				j.CreatedAt,
				status,
				j.Server,
				j.DurationMs,
				truncate(j.Command, 60),
			)

			// Ispisi gresku ako postoji
			if j.Error != "" {
				fmt.Printf("         error: %s\n", j.Error)
			}
		}

		return nil
	},
}

// truncate skracuje string na max znakova radi urednog ispisa
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func init() {
	historyCmd.Flags().Int("limit", 20, "Number of jobs to show")
	rootCmd.AddCommand(historyCmd)
}
