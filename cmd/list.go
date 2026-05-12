/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type Server struct {
	Name string
	Host string
}

var servers = []Server{
	{
		Name: "hetzner-main",
		Host: "1.2.3.4",
	},
	{
		Name: "backup-node",
		Host: "5.6.7.8",
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List servers",
	Run: func(cmd *cobra.Command, args []string) {

		for _, server := range servers {
			fmt.Printf(
				"%s (%s)\n",
				server.Name,
				server.Host,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
