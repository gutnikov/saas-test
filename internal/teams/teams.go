// Package teams holds the rules of a team (spec TEAMS-1): what makes a name valid, and the errors that refuse one.
package teams

import (
	"strings"
	"time"
	"unicode/utf8"
)

// MaxNameLength is the longest name, in characters, once the spaces at both ends are trimmed (TEAMS-1#3).
const MaxNameLength = 60

// Error is a refusal of the command CreateTeam: its code is the error's name in the spec, its message the text a
// person reads (TEAMS-1#4).
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// The errors of CreateTeam.
var (
	ErrNameRequired = &Error{Code: "TeamNameRequired", Message: "Enter a team name"}
	ErrNameTooLong  = &Error{Code: "TeamNameTooLong", Message: "A team name can be up to 60 characters"}
	ErrNameTaken    = &Error{Code: "TeamNameTaken", Message: "A team with this name already exists"}
)

// Team is a team as the app keeps it.
type Team struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// Name returns a name as it is kept, trimmed, or the error that refuses it.
func Name(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", ErrNameRequired
	case utf8.RuneCountInString(name) > MaxNameLength:
		return "", ErrNameTooLong
	}
	return name, nil
}

// Key is what two names that differ only in case have in common: no two teams share it.
func Key(name string) string { return strings.ToLower(name) }
