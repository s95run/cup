package service

import (
	"testing"
	"time"

	"github.com/s95run/cup/internal/s95client"
)

func TestAthleteCardSignVerify(t *testing.T) {
	secret := "jwt-secret-min-32-chars-xxxxxxxxxx"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	hit := s95client.AthleteSearchHit{
		ID: 101, Code: 770000101, Name: "A", Gender: "female",
		HomeEventCode: "x", HomeEventName: "X", ClubName: "C",
		RunsCount: 1, VolunteeringCount: 2,
	}
	card := SignAthleteCard(secret, hit, now)
	got, err := VerifyAthleteCard(secret, card, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != hit.ID || got.RunsCount != hit.RunsCount {
		t.Fatalf("got %+v", got)
	}
	card.RunsCount = 99
	if _, err := VerifyAthleteCard(secret, card, now); err != ErrAthleteCardInvalid {
		t.Fatalf("want invalid, got %v", err)
	}
}

func TestAthleteCardExpired(t *testing.T) {
	secret := "jwt-secret-min-32-chars-xxxxxxxxxx"
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	card := SignAthleteCard(secret, s95client.AthleteSearchHit{ID: 1, Code: 1, Gender: "male"}, now)
	if _, err := VerifyAthleteCard(secret, card, now.Add(48*time.Hour)); err != ErrAthleteCardExpired {
		t.Fatalf("got %v", err)
	}
}