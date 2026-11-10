package teams

import (
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	for _, c := range []struct {
		raw, want string
		err       error
	}{
		{"Acme", "Acme", nil},
		{"  Acme  ", "Acme", nil},
		{"\tAcme\n", "Acme", nil},
		{"", "", ErrNameRequired},
		{"   ", "", ErrNameRequired},
		{strings.Repeat("x", 60), strings.Repeat("x", 60), nil},
		{strings.Repeat("x", 61), "", ErrNameTooLong},
		{" " + strings.Repeat("x", 60) + " ", strings.Repeat("x", 60), nil},
		{strings.Repeat("ж", 60), strings.Repeat("ж", 60), nil}, // characters, not bytes
		{strings.Repeat("ж", 61), "", ErrNameTooLong},
	} {
		got, err := Name(c.raw)
		if got != c.want || err != c.err {
			t.Errorf("Name(%q) = %q, %v; want %q, %v", c.raw, got, err, c.want, c.err)
		}
	}
}

func TestKeyIgnoresCase(t *testing.T) {
	if Key("Acme") != Key("ACME") || Key("Acme") == Key("Acme Inc") {
		t.Fatalf("Key: Acme %q, ACME %q, Acme Inc %q", Key("Acme"), Key("ACME"), Key("Acme Inc"))
	}
}
