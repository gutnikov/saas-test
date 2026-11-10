package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestMigrateAppliesEachMigrationOnce(t *testing.T) {
	db := openTemp(t)
	files := fstest.MapFS{
		"migrations/0001_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER PRIMARY KEY);`)},
		"migrations/0002_b.sql": {Data: []byte(`CREATE TABLE b (id INTEGER PRIMARY KEY); INSERT INTO a (id) VALUES (1);`)},
	}
	for run := 1; run <= 2; run++ {
		if err := migrate(db, files); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}
	if got := count(t, db, `SELECT COUNT(*) FROM schema_migrations`); got != 2 {
		t.Errorf("schema_migrations has %d rows, want 2", got)
	}
	if got := count(t, db, `SELECT COUNT(*) FROM a`); got != 1 {
		t.Errorf("table a has %d rows, want 1: a migration ran twice", got)
	}
}

func TestMigrateRollsBackAFailedMigration(t *testing.T) {
	db := openTemp(t)
	files := fstest.MapFS{
		"migrations/0001_bad.sql": {Data: []byte(`CREATE TABLE c (id INTEGER PRIMARY KEY); SELECT * FROM missing;`)},
	}
	if err := migrate(db, files); err == nil {
		t.Fatal("a failed migration returned no error")
	}
	if got := count(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'c'`); got != 0 {
		t.Error("table c exists: the failed migration was not rolled back")
	}
	if got := count(t, db, `SELECT COUNT(*) FROM schema_migrations`); got != 0 {
		t.Error("the failed migration is recorded as applied")
	}
}

func openTemp(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func count(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}
