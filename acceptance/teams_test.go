package acceptance

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

// The examples of slice S01, Create team, of specs/model/teams.em.yaml.

// example: S01/a new name creates a team
func TestS01ANewNameCreatesATeam(t *testing.T) {
	app := noTeams(t)

	res := app.createTeam("Acme")

	wantCreated(t, res, "Acme")
	wantTeamNames(t, app, "Acme")
	wantTeamCreated(t, res, "Acme")
}

// example: S01/the newest team is listed first
func TestS01TheNewestTeamIsListedFirst(t *testing.T) {
	app := teamAcme(t)

	res := app.createTeam("Beta")

	wantCreated(t, res, "Beta")
	wantTeamNames(t, app, "Beta", "Acme")
	wantTeamCreated(t, res, "Beta")
}

// example: S01/the spaces around a name are trimmed
func TestS01TheSpacesAroundANameAreTrimmed(t *testing.T) {
	app := noTeams(t)

	res := app.createTeam("  Acme  ")

	wantCreated(t, res, "Acme")
	wantTeamNames(t, app, "Acme")
	wantTeamCreated(t, res, "Acme")
}

// example: S01/a name of 60 characters is accepted
func TestS01ANameOf60CharactersIsAccepted(t *testing.T) {
	name := strings.Repeat("x", 60)
	if n := utf8.RuneCountInString(name); n != 60 {
		t.Fatalf("the name has %d characters, want 60", n)
	}
	app := noTeams(t)

	res := app.createTeam(name)

	wantCreated(t, res, name)
	wantTeamNames(t, app, name)
	wantTeamCreated(t, res, name)
}

// example: S01/an empty name is refused
func TestS01AnEmptyNameIsRefused(t *testing.T) {
	app := noTeams(t)

	res := app.createTeam("   ")

	wantRefused(t, res, http.StatusBadRequest, "TeamNameRequired", "Enter a team name")
	wantNoEvents(t, res)
}

// example: S01/a name of 61 characters is refused
func TestS01ANameOf61CharactersIsRefused(t *testing.T) {
	name := strings.Repeat("x", 61)
	if n := utf8.RuneCountInString(name); n != 61 {
		t.Fatalf("the name has %d characters, want 61", n)
	}
	app := noTeams(t)

	res := app.createTeam(name)

	wantRefused(t, res, http.StatusBadRequest, "TeamNameTooLong", "A team name can be up to 60 characters")
	wantNoEvents(t, res)
}

// example: S01/a taken name is refused whatever its case
func TestS01ATakenNameIsRefusedWhateverItsCase(t *testing.T) {
	app := teamAcme(t)

	res := app.createTeam("ACME")

	wantRefused(t, res, http.StatusConflict, "TeamNameTaken", "A team with this name already exists")
	wantNoEvents(t, res)
}
