package store

import (
	"database/sql"
	"sync"

	// modernc.org/sqlite je pure Go SQLite driver - ne treba CGO ni gcc
	// Registrira se automatski kao "sqlite" driver kroz blank import
	_ "modernc.org/sqlite"
)

const dbPath = "data/vpsctl.db"

// db je paketna varijabla — jedan konekcija za cijeli process
// sync.Once garantira da se Init pozove samo jednom, cak i iz vise goroutina
var (
	db   *sql.DB
	once sync.Once
)

// Init otvara konekciju i kreira tablice ako ne postoje.
// Poziva se automatski kroz Get() — ne treba rucno zvati.
func Init() error {
	var initErr error

	once.Do(func() {
		conn, err := sql.Open("sqlite", dbPath)
		if err != nil {
			initErr = err
			return
		}

		// WAL mode = bolji concurrent read/write performance
		if _, err = conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
			initErr = err
			return
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
			initErr = err
			return
		}

		db = conn
	})

	return initErr
}

// Get vraca aktivnu DB konekciju.
// Ako Init nije pozvan, poziva ga automatski.
func Get() (*sql.DB, error) {
	if db == nil {
		if err := Init(); err != nil {
			return nil, err
		}
	}
	return db, nil
}
