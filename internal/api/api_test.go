package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Config bez servera: liste moraju biti [] a ne null, inace UI pada na .map().
func TestEmptyConfig_ReturnsEmptyArrays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfgPath := filepath.Join(t.TempDir(), "servers.yaml")
	if err := os.WriteFile(cfgPath, []byte("servers: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VPSCTL_CONFIG", cfgPath)

	r := gin.New()
	Setup(r, "tajna")

	for _, path := range []string{"/servers", "/servers/ping"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer tajna")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if body := strings.TrimSpace(w.Body.String()); body != "[]" {
				t.Errorf("body = %s, want []", body)
			}
		})
	}
}
