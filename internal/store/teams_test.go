package store

import (
	"context"
	"testing"

	"github.com/gutnikov/saas-test/internal/teams"
)

func TestCreateTeamRefusesATakenNameWhateverItsCase(t *testing.T) {
	db, ctx := openTemp(t), context.Background()
	if _, err := CreateTeam(ctx, db, "Acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTeam(ctx, db, "ACME"); err != teams.ErrNameTaken {
		t.Fatalf("second Acme: %v, want %v", err, teams.ErrNameTaken)
	}
}

func TestListTeamsNewestFirst(t *testing.T) {
	db, ctx := openTemp(t), context.Background()
	for _, n := range []string{"Acme", "Beta", "Gamma"} {
		if _, err := CreateTeam(ctx, db, n); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ListTeams(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tm := range got {
		names = append(names, tm.Name)
	}
	if len(names) != 3 || names[0] != "Gamma" || names[2] != "Acme" {
		t.Fatalf("ListTeams: %v, want [Gamma Beta Acme]", names)
	}
}

func TestListTeamsOfAnEmptyStoreIsAnEmptyList(t *testing.T) {
	got, err := ListTeams(context.Background(), openTemp(t))
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("ListTeams: %#v, %v; want an empty, non-nil list", got, err)
	}
}
