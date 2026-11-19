package config

import "testing"

func TestParseAthleteCodes(t *testing.T) {
	got := parseAthleteCodes("770000001, 770000002;0 bad")
	if len(got) != 2 || got[0] != 770000001 || got[1] != 770000002 {
		t.Fatalf("codes = %v", got)
	}
	if parseAthleteCodes("") != nil {
		t.Fatal("empty input must stay empty")
	}
}
