package domain

import (
	"testing"
	"time"
)

func qual(id int64, g Gender, runs, vols int) AthleteQualification {
	return AthleteQualification{S95AthleteID: id, Gender: g, RunsCount: runs, VolCount: vols}
}

func TestValidateRoster_ok(t *testing.T) {
	closes := time.Now().Add(24 * time.Hour)
	members := []AthleteQualification{
		qual(1, GenderFemale, 1, 0),
		qual(2, GenderFemale, 0, 4),
		qual(3, GenderMale, 5, 0),
		qual(4, GenderMale, 1, 0),
		qual(5, GenderMale, 2, 0),
	}
	if err := ValidateRoster(members, "angarskieprudy", time.Now(), closes); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
}

func TestValidateRoster_closed(t *testing.T) {
	closes := time.Now().Add(-time.Hour)
	members := make([]AthleteQualification, TeamSize)
	for i := range members {
		members[i] = qual(int64(i+1), GenderFemale, 1, 0)
	}
	err := ValidateRoster(members, "x", time.Now(), closes)
	if err != ErrRegistrationClosed {
		t.Fatalf("got %v", err)
	}
}

func TestValidateRoster_notEnoughFemales(t *testing.T) {
	closes := time.Now().Add(time.Hour)
	members := []AthleteQualification{
		qual(1, GenderFemale, 1, 0),
		qual(2, GenderMale, 1, 0),
		qual(3, GenderMale, 1, 0),
		qual(4, GenderMale, 1, 0),
		qual(5, GenderMale, 1, 0),
	}
	if err := ValidateRoster(members, "x", time.Now(), closes); err != ErrNotEnoughFemales {
		t.Fatalf("got %v", err)
	}
}

func TestRegistrationClosesAt_moscowSaturdayNine(t *testing.T) {
	stageDate := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC) // Saturday
	closes, err := RegistrationClosesAt(stageDate, "Europe/Moscow", 9, 0, 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	want := time.Date(2026, 5, 30, 9, 0, 0, 0, loc)
	if !closes.Equal(want) {
		t.Fatalf("closes %s want %s", closes, want)
	}
}

func TestRegistrationClosesAt_invalidTimezone(t *testing.T) {
	_, err := RegistrationClosesAt(time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC), "Not/AZone", 9, 0, time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistrationOpen_exactDeadline(t *testing.T) {
	closes := time.Date(2026, 5, 30, 6, 0, 0, 0, time.UTC)
	if !RegistrationOpen(closes, closes) {
		t.Fatal("deadline instant must still be open")
	}
	if RegistrationOpen(closes.Add(time.Nanosecond), closes) {
		t.Fatal("after deadline must be closed")
	}
}
