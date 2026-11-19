package config

import "testing"

func TestParseAthleteIDs(t *testing.T) {
	got := parseAthleteIDs("1, 2;0 bad")
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("ids = %v", got)
	}
	if parseAthleteIDs("") != nil {
		t.Fatal("empty input must stay empty")
	}
}
