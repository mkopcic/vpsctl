package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	// modernc.org/sqlite je pure Go SQLite driver - ne treba CGO ni gcc
	// Registrira se automatski kao "sqlite" driver kroz blank import
	_ "modernc.org/sqlite"
)

// defaultDir vrijedi kad VPSCTL_DATA nije postavljen — relativno na
// working directory, kao i dosad.
const defaultDir = "data"

// db je paketna varijabla — jedna konekcija za cijeli process.
// Mutex umjesto sync.Once: Once pamti da je Init "odradjen" i kad je pao,
// pa bi svaki sljedeci Get vratio nil konekciju bez greske.
var (
	mu sync.Mutex
	db *sql.DB
)

// Path vraca put do SQLite fajla.
// VPSCTL_DATA je direktorij — u containeru to je volume, inace history
// nestane pri svakom restartu.
func Path() string {
	dir := os.Getenv("VPSCTL_DATA")
	if dir == "" {
		dir = defaultDir
	}
	return filepath.Join(dir, "vpsctl.db")
}

// Init otvara konekciju i kreira tablice ako ne postoje.
// Poziva se automatski kroz Get() — ne treba rucno zvati.
func Init() error {
	_, err := Get()
	return err
}

// Get vraca aktivnu DB konekciju. Prvi poziv je otvara.
func Get() (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	if db != nil {
		return db, nil
	}

	conn, err := open(Path())
	if err != nil {
		return nil, err
	}

	db = conn
	return db, nil
}

func open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// WAL mode = bolji concurrent read/write performance
	if _, err = conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, err
	}

	// Kreiraj jobs tablicu ako vec ne postoji
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			server      TEXT    NOT NULL,
			command     TEXT    NOT NULL,
			output      TEXT,
			ok          INTEGER NOT NULL DEFAULT 1,
			error       TEXT,
			duration_ms INTEGER,
			source      TEXT    NOT NULL DEFAULT 'cli',
			created_at  TEXT    NOT NULL DEFAULT (datetime('now', 'localtime'))
		)
	`)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// Ping provjerava da je baza otvorena i odgovara — koristi ga /readyz.
func Ping(ctx context.Context) error {
	conn, err := Get()
	if err != nil {
		return err
	}
	return conn.PingContext(ctx)
}

// Close zatvara konekciju. Sljedeci Get je otvara ponovno.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if db == nil {
		return nil
	}
	err := db.Close()
	db = nil
	return err
}
