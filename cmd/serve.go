package cmd

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mkopcic/vpsctl/internal/api"
	"github.com/mkopcic/vpsctl/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP API server with web UI",

	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")

		// gin.Default() = Logger + Recovery middleware
		r := gin.Default()

		// REST API rute (/servers/*, /jobs)
		api.Setup(r)

		// Web UI — embedded staticki fajlovi iz internal/web/static/
		// Gin servira sve fajlove pod /ui/ path
		r.StaticFS("/ui", web.FS())

		// Redirect root / -> /ui/ da ne treba tipkati /ui
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/ui/")
		})

		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("vpsctl running on http://localhost%s\n", addr)
		fmt.Printf("Web UI:  http://localhost%s/ui/\n", addr)
		fmt.Printf("API:     http://localhost%s/servers\n", addr)

		return r.Run(addr)
	},
}

func init() {
	serveCmd.Flags().Int("port", 8080, "Port on which the server listens")
	rootCmd.AddCommand(serveCmd)
}
