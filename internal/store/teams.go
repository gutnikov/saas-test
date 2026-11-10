package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/gutnikov/saas-test/internal/teams"
)

// CreateTeam keeps a team under a name that teams.Name has already accepted. A name another team has, whatever its
// case, is teams.ErrNameTaken: the unique key decides, so two requests at once cannot both win.
func CreateTeam(ctx context.Context, db *sql.DB, name string) (teams.Team, error) {
	id, err := newID()
	if err != nil {
		return teams.Team{}, err
	}
	t := teams.Team{ID: id, Name: name, CreatedAt: time.Now().UTC()}
	_, err = db.ExecContext(ctx, `INSERT INTO teams (id, name, name_key, created_at) VALUES (?, ?, ?, ?)`,
		t.ID, t.Name, teams.Key(t.Name), t.CreatedAt.Format(time.RFC3339Nano))
	var se *sqlite.Error
	if errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return teams.Team{}, teams.ErrNameTaken
	}
	if err != nil {
		return teams.Team{}, fmt.Errorf("create team: %w", err)
	}
	return t, nil
}

// ListTeams returns every team, newest first.
func ListTeams(ctx context.Context, db *sql.DB) ([]teams.Team, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, created_at FROM teams ORDER BY seq DESC`)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()
	out := []teams.Team{}
	for rows.Next() {
		var t teams.Team
		var created string
		if err := rows.Scan(&t.ID, &t.Name, &created); err != nil {
			return nil, fmt.Errorf("list teams: %w", err)
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, t)
	}
	return out, rows.Err()
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("team id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
