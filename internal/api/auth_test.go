package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mkopcic/vpsctl/internal/store"
)

// newTestRouter slaze router kao serve.go, s bazom u privremenom direktoriju.
func newTestRouter(t *testing.T, token string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	t.Setenv("VPSCTL_DATA", filepath.Join(t.TempDir(), "data"))
	t.Cleanup(func() { store.Close() })

	r := gin.New()
	Setup(r, token)
	return r
}

func do(r http.Handler, method, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(`{"server":"all","command":"id"}`))
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTokenAuth(t *testing.T) {
	r := newTestRouter(t, "tajna")

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"bez headera", "", http.StatusUnauthorized},
		{"krivi token", "Bearer kriva", http.StatusUnauthorized},
		{"prefiks tocnog tokena", "Bearer tajn", http.StatusUnauthorized},
		{"bez Bearer prefiksa", "tajna", http.StatusUnauthorized},
		{"Basic umjesto Bearer", "Basic tajna", http.StatusUnauthorized},
		{"prazan Bearer", "Bearer ", http.StatusUnauthorized},
		{"tocan token", "Bearer tajna", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// /jobs ne treba servers.yaml ni SSH — cita samo praznu bazu
			if got := do(r, http.MethodGet, "/jobs", tt.header).Code; got != tt.want {
				t.Errorf("GET /jobs status = %d, want %d", got, tt.want)
			}
		})
	}
}

// Nijedna ruta koja dira servere ne smije proci bez tokena.
func TestAllAPIRoutesRequireToken(t *testing.T) {
	r := newTestRouter(t, "tajna")

	routes := []struct{ method, path string }{
		{http.MethodGet, "/servers"},
		{http.MethodGet, "/servers/ping"},
		{http.MethodPost, "/servers/exec"},
		{http.MethodPost, "/servers/update"},
		{http.MethodPost, "/servers/restart"},
		{http.MethodGet, "/jobs"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			if got := do(r, rt.method, rt.path, "").Code; got != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", got)
			}
		})
	}
}

// Prazan token ne smije znaciti "auth iskljucen".
func TestTokenAuth_EmptyTokenRejectsEverything(t *testing.T) {
	r := newTestRouter(t, "")

	for _, header := range []string{"", "Bearer ", "Bearer bilo-sto"} {
		if got := do(r, http.MethodGet, "/jobs", header).Code; got != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", header, got)
		}
	}
}

func TestHealthEndpointsAreOpen(t *testing.T) {
	r := newTestRouter(t, "tajna")

	for _, path := range []string{"/healthz", "/readyz"} {
		if got := do(r, http.MethodGet, path, "").Code; got != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, got)
		}
	}
}

func TestNewToken(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewToken()

	if len(a) != 64 {
		t.Errorf("duljina tokena = %d, want 64 hex znaka", len(a))
	}
	if a == b {
		t.Error("dva poziva dala su isti token")
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.53", true},
		{"::1", true},
		{"localhost", true},
		{"", false},
		{"0.0.0.0", false},
		{"::", false},
		{"10.0.0.5", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		if got := IsLoopback(tt.host); got != tt.want {
			t.Errorf("IsLoopback(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}
