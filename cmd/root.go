/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vpsctl",
	Short: "VPS management CLI",
	Long:  `CLI alat za upravljanje VPS serverima preko SSH-a. Omogućava jednostavno izvršavanje naredbi, upravljanje datotekama i nadzor resursa na udaljenim VPS serverima.`,
	// Gresku ispisuje Execute() — bez ovoga Cobra je ispise jos jednom s "Error:" prefiksom
	SilenceErrors: true,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Ctrl+C / SIGTERM prekida context — sve SSH komande ga dobivaju
	// kroz cmd.Context() i cisto se prekidaju umjesto da vise
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := rootCmd.ExecuteContext(ctx)
	stop()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vpsctl.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
