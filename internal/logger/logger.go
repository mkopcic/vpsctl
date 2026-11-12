package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// defaultFile vrijedi za CLI kad VPSCTL_LOG_FILE nije postavljen.
const defaultFile = "logs/vpsctl.log"

// out != nil znaci: pisi tamo umjesto u fajl (serve mod -> stdout).
var (
	mu  sync.Mutex
	out io.Writer
)

// UseStdout preusmjerava log na stdout, osim ako je VPSCTL_LOG_FILE postavljen.
// Zove ga `vpsctl serve`: u containeru logove skuplja Docker/Kubernetes sa
// stdouta, a fajl unutar containera nestane s njim.
// CLI ostaje na fajlu — stdout je tamo vec zauzet ispisom za korisnika.
func UseStdout() {
	if os.Getenv("VPSCTL_LOG_FILE") != "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	out = os.Stdout
}

// Path vraca put do log fajla.
func Path() string {
	if p := os.Getenv("VPSCTL_LOG_FILE"); p != "" {
		return p
	}
	return defaultFile
}

func Log(server, command, result string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s — %q\n%s\n---\n", timestamp, server, command, result)

	mu.Lock()
	defer mu.Unlock()

	if out != nil {
		_, err := io.WriteString(out, entry)
		return err
	}

	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}
