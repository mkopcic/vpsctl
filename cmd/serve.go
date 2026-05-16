package cmd

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mkopcic/vpsctl/internal/api"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP API server",

	RunE: func(cmd *cobra.Command, args []string) error {
		// Dohvati --port flag (default 8080)
		port, _ := cmd.Flags().GetInt("port")

		// gin.Default() uključuje Logger i Recovery middleware
		// Logger = svaki request se logira u terminal
		// Recovery = panic ne srusi server, vraca 500
		r := gin.Default()

		// Registriraj sve /servers/* rute iz internal/api/api.go
		api.Setup(r)

		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("vpsctl API server running on http://localhost%s\n", addr)

		// Pokreni server — blokira dok se ne ugasi (Ctrl+C)
		return r.Run(addr)
	},
}

func init() {
	// --port flag, default 8080
	serveCmd.Flags().Int("port", 8080, "Port on which the API server listens")
	rootCmd.AddCommand(serveCmd)
}
