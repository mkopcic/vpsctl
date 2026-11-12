package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromEnvPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "moji-serveri.yaml")
	yaml := `servers:
  - name: test
    host: example.com
    port: 2222
    user: deploy
    domains:
      - example.com
`
	if err := os.WriteFile(path, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VPSCTL_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	s, ok := cfg.Find("test")
	if !ok {
		t.Fatal("server 'test' nije pronadjen")
	}
	if s.Host != "example.com" || s.Port != 2222 || s.User != "deploy" {
		t.Errorf("krivo ucitan server: %+v", s)
	}
	if len(s.Domains) != 1 || s.Domains[0] != "example.com" {
		t.Errorf("Domains = %v", s.Domains)
	}
	if _, ok := cfg.Find("nema-ga"); ok {
		t.Error("Find je nasao server koji ne postoji")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("VPSCTL_CONFIG", filepath.Join(t.TempDir(), "nema.yaml"))

	if _, err := Load(); err == nil {
		t.Error("ocekivana greska za fajl koji ne postoji")
	}
}

func TestPath_Default(t *testing.T) {
	t.Setenv("VPSCTL_CONFIG", "")

	if got := Path(); got != defaultPath {
		t.Errorf("Path() = %q, want %q", got, defaultPath)
	}
}
