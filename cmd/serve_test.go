package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestResolveToken(t *testing.T) {
	tests := []struct {
		name          string
		envToken      string
		listen        string
		wantErr       bool
		wantGenerated bool
	}{
		{"token + loopback", "tajna", "127.0.0.1", false, false},
		{"token + sva sucelja", "tajna", "0.0.0.0", false, false},
		{"bez tokena + loopback", "", "127.0.0.1", false, true},
		{"bez tokena + localhost", "", "localhost", false, true},
		{"bez tokena + sva sucelja", "", "0.0.0.0", true, false},
		{"bez tokena + prazan host", "", "", true, false},
		{"bez tokena + javna adresa", "", "203.0.113.7", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, generated, err := resolveToken(tt.envToken, tt.listen)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if generated != tt.wantGenerated {
				t.Errorf("generated = %v, want %v", generated, tt.wantGenerated)
			}
			if token == "" {
				t.Error("token je prazan")
			}
			if tt.envToken != "" && token != tt.envToken {
				t.Errorf("token = %q, want onaj iz env varijable", token)
			}
		})
	}
}

// Prekid contexta (Ctrl+C / SIGTERM) mora ugasiti server cisto, bez greske.
func TestServe_GracefulShutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("VPSCTL_DATA", filepath.Join(t.TempDir(), "data"))
	t.Setenv("VPSCTL_API_TOKEN", "tajna")

	// Slobodan port: otvori :0, zapamti broj, zatvori
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	serveCmd.Flags().Set("listen", "127.0.0.1")
	serveCmd.Flags().Set("port", strconv.Itoa(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveCmd.SetContext(ctx)

	done := make(chan error, 1)
	go func() { done <- serveCmd.RunE(serveCmd, nil) }()

	// Cekaj da server pocne odgovarati
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	up := false
	for range 50 {
		if resp, err := http.Get(url); err == nil {
			resp.Body.Close()
			up = resp.StatusCode == http.StatusOK
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !up {
		t.Fatal("server se nije podigao")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve nakon prekida contexta vratio gresku: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve se nije ugasio nakon prekida contexta")
	}

	if _, err := http.Get(url); err == nil {
		t.Error("server i dalje odgovara nakon gasenja")
	}
}
