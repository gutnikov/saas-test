// Package store opens the service's SQLite database and applies its migrations.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"

	_ "modernc.org/sqlite" // the "sqlite" driver, in pure Go
)

//go:embed migrations
var migrations embed.FS

// Open opens the database at path, creating it if needed, and applies every migration it has not applied yet.
func Open(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+file+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", file, err)
	}
	if err := migrate(db, migrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// migrate applies the .sql files under migrations/ in name order, each once, and records them in
// schema_migrations. A migration that fails is rolled back whole.
func migrate(db *sql.DB, files fs.FS) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name       TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	names, err := fs.Glob(files, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		if err := apply(db, files, name); err != nil {
			return fmt.Errorf("migration %s: %w", path.Base(name), err)
		}
	}
	return nil
}

func apply(db *sql.DB, files fs.FS, name string) error {
	base := path.Base(name)
	var done int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, base).Scan(&done); err != nil {
		return err
	}
	if done > 0 {
		return nil
	}
	body, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(string(body)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, base); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
