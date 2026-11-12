package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// useTempDB usmjerava store na praznu bazu u privremenom direktoriju.
// Close u Cleanup je obavezan — na Windowsu se otvoren fajl ne moze obrisati.
func useTempDB(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "nested", "data")
	t.Setenv("VPSCTL_DATA", dir)
	t.Cleanup(func() { Close() })
	return dir
}

func TestSaveAndList(t *testing.T) {
	dir := useTempDB(t)

	jobs := []Job{
		NewJob("alfa", "uptime", "up 3 days", "", "cli", true, time.Now()),
		NewJob("beta", "false", "", "exit status 1", "api", false, time.Now()),
		NewJob("alfa", "df -h", "/dev/sda1", "", "api", true, time.Now()),
	}
	for _, j := range jobs {
		if err := Save(j); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	// Get mora sam kreirati direktorij koji ne postoji
	if _, err := os.Stat(filepath.Join(dir, "vpsctl.db")); err != nil {
		t.Fatalf("baza nije kreirana u VPSCTL_DATA: %v", err)
	}

	all, err := List(10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("List vratio %d jobova, want 3", len(all))
	}
	// Najnoviji prvi
	if all[0].Command != "df -h" {
		t.Errorf("prvi job = %q, want najnoviji (df -h)", all[0].Command)
	}

	limited, err := List(2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("List(2) vratio %d jobova", len(limited))
	}

	beta, err := ListByServer("beta", 10)
	if err != nil {
		t.Fatalf("ListByServer: %v", err)
	}
	if len(beta) != 1 {
		t.Fatalf("ListByServer(beta) vratio %d jobova, want 1", len(beta))
	}
	if beta[0].OK || beta[0].Error != "exit status 1" || beta[0].Source != "api" {
		t.Errorf("krivo procitan job: %+v", beta[0])
	}
}

func TestPing(t *testing.T) {
	useTempDB(t)

	if err := Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestClose_ReopensOnNextGet(t *testing.T) {
	useTempDB(t)

	if err := Save(NewJob("alfa", "uptime", "", "", "cli", true, time.Now())); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Podaci moraju prezivjeti zatvaranje i ponovno otvaranje
	jobs, err := List(10)
	if err != nil {
		t.Fatalf("List nakon Close: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("nakon ponovnog otvaranja %d jobova, want 1", len(jobs))
	}
}

func TestPath(t *testing.T) {
	t.Setenv("VPSCTL_DATA", "")
	if got, want := Path(), filepath.Join("data", "vpsctl.db"); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}

	t.Setenv("VPSCTL_DATA", filepath.Join("x", "y"))
	if got, want := Path(), filepath.Join("x", "y", "vpsctl.db"); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}
