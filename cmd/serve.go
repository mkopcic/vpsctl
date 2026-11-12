package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mkopcic/vpsctl/internal/api"
	"github.com/mkopcic/vpsctl/internal/logger"
	"github.com/mkopcic/vpsctl/internal/store"
	"github.com/mkopcic/vpsctl/internal/web"
	"github.com/spf13/cobra"
)

// shutdownTimeout: koliko cekamo da zapoceti requesti (SSH komande) zavrse
// nakon SIGTERM-a. Ispod Kubernetes defaulta od 30s, nakon kojeg stize SIGKILL.
const shutdownTimeout = 25 * time.Second

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP API server with web UI",
	Long: `Pokreni HTTP API i web UI.

API izvrsava komande na svim serverima, pa svaki request trazi token:
  Authorization: Bearer <token>

Token dolazi iz VPSCTL_API_TOKEN. Ako nije postavljen, a server slusa samo na
loopbacku, generira se novi pri svakom pokretanju i ispise u terminal.
Na bilo kojoj drugoj adresi server bez VPSCTL_API_TOKEN ne starta.`,
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		listen, _ := cmd.Flags().GetString("listen")

		token, generated, err := resolveToken(os.Getenv("VPSCTL_API_TOKEN"), listen)
		if err != nil {
			return err
		}

		// U containeru logove skuplja Docker/Kubernetes sa stdouta
		logger.UseStdout()

		// Otvori bazu odmah — greska u putu ili pravima treba srusiti start,
		// a ne prvi request
		if err := store.Init(); err != nil {
			return fmt.Errorf("baza %s: %w", store.Path(), err)
		}
		defer store.Close()

		// gin.Default() = Logger + Recovery middleware
		r := gin.Default()

		// REST API rute (/servers/*, /jobs) + /healthz, /readyz
		api.Setup(r, token)

		// Web UI — embedded staticki fajlovi iz internal/web/static/
		// Gin servira sve fajlove pod /ui/ path. Sami fajlovi nisu tajna;
		// UI salje token uz svaki API poziv.
		r.StaticFS("/ui", web.FS())

		// Redirect root / -> /ui/ da ne treba tipkati /ui
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/ui/")
		})

		addr := net.JoinHostPort(listen, strconv.Itoa(port))
		base := "http://" + addr

		fmt.Printf("vpsctl running on %s\n", base)
		fmt.Printf("API:     %s/servers  (Authorization: Bearer <token>)\n", base)
		if generated {
			// Dio iza # browser ne salje serveru — UI ga procita i spremi
			fmt.Printf("Web UI:  %s/ui/#token=%s\n", base, token)
			fmt.Println("Token je generiran za ovo pokretanje. Za stalan token postavi VPSCTL_API_TOKEN.")
		} else {
			fmt.Printf("Web UI:  %s/ui/\n", base)
		}

		// http.Server umjesto r.Run() — samo on ima Shutdown
		srv := &http.Server{
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.ListenAndServe()
		}()

		// cmd.Context() se prekida na Ctrl+C / SIGTERM (vidi root.go)
		select {
		case err := <-errCh:
			return err // port zauzet i slicno
		case <-cmd.Context().Done():
		}

		fmt.Println("gasim se — cekam da zapoceti requesti zavrse...")

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// Shutdown prestaje primati nove konekcije i ceka postojece requeste
		if err := srv.Shutdown(ctx); err != nil {
			// Timeout: prekini i ono sto jos traje. To zatvara request
			// contexte, pa se SSH komande prekidaju cisto.
			srv.Close()
			return fmt.Errorf("shutdown: %w", err)
		}

		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	},
}

// resolveToken odlucuje koji API token vrijedi.
// Postavljen token uvijek prolazi. Bez njega: na loopbacku generiramo novi,
// a na bilo kojoj drugoj adresi odbijamo startati.
func resolveToken(envToken, listen string) (token string, generated bool, err error) {
	if envToken != "" {
		return envToken, false, nil
	}

	if !api.IsLoopback(listen) {
		return "", false, fmt.Errorf("VPSCTL_API_TOKEN nije postavljen, a --listen %q nije loopback — "+
			"API bi bio dostupan s mreze bez zastite", listen)
	}

	token, err = api.NewToken()
	return token, true, err
}

// envInt cita cijeli broj iz env varijable, uz default
func envInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return n
	}
	return def
}

// envString cita env varijablu, uz default
func envString(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func init() {
	serveCmd.Flags().Int("port", envInt("VPSCTL_PORT", 8080), "Port on which the server listens (env VPSCTL_PORT)")
	serveCmd.Flags().String("listen", envString("VPSCTL_LISTEN", "127.0.0.1"),
		"Adresa na kojoj server slusa; 0.0.0.0 za sva sucelja (env VPSCTL_LISTEN)")
	rootCmd.AddCommand(serveCmd)
}
