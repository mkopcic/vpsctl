package store

import "time"

// Job predstavlja jednu izvrsenu operaciju — jedan redak u bazi
type Job struct {
	ID         int64
	Server     string
	Command    string
	Output     string
	OK         bool
	Error      string
	DurationMs int64
	Source     string // "cli" ili "api"
	CreatedAt  string
}

// Save sprema jedan job u bazu.
// Poziva se iz runOnServer (CLI) i API handlera.
func Save(j Job) error {
	db, err := Get()
	if err != nil {
		return err
	}

	// ok se sprema kao 0/1 jer SQLite nema bool tip
	ok := 0
	if j.OK {
		ok = 1
	}

	_, err = db.Exec(`
		INSERT INTO jobs (server, command, output, ok, error, duration_ms, source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, j.Server, j.Command, j.Output, ok, j.Error, j.DurationMs, j.Source)

	return err
}

// List vraca zadnjih N jobova, sortirano od najnovijeg
func List(limit int) ([]Job, error) {
	return query("SELECT id, server, command, output, ok, error, duration_ms, source, created_at FROM jobs ORDER BY id DESC LIMIT ?", limit)
}

// ListByServer vraca zadnjih N jobova za specificni server
func ListByServer(server string, limit int) ([]Job, error) {
	return query("SELECT id, server, command, output, ok, error, duration_ms, source, created_at FROM jobs WHERE server = ? ORDER BY id DESC LIMIT ?", server, limit)
}

// query je interni helper koji izvrsava SELECT i mapira redove u []Job
func query(sql string, args ...any) ([]Job, error) {
	db, err := Get()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job

	for rows.Next() {
		var j Job
		var ok int // citamo kao int, konvertiramo u bool

		err := rows.Scan(
			&j.ID, &j.Server, &j.Command, &j.Output,
			&ok, &j.Error, &j.DurationMs, &j.Source, &j.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		j.OK = ok == 1
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

// NewJob je helper za kreiranje Job strukture s automatskim trajanjem.
// start = time.Now() prije SSH poziva, source = "cli" ili "api"
func NewJob(server, command, output, errMsg, source string, ok bool, start time.Time) Job {
	return Job{
		Server:     server,
		Command:    command,
		Output:     output,
		OK:         ok,
		Error:      errMsg,
		DurationMs: time.Since(start).Milliseconds(),
		Source:     source,
	}
}
