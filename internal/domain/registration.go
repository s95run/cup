package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrRegistrationClosed  = errors.New("registration closed")
	ErrWrongTeamSize       = errors.New("team must have exactly 5 members")
	ErrNotEnoughFemales    = errors.New("team must have at least 2 female members")
	ErrDuplicateAthlete    = errors.New("duplicate athlete in roster")
	ErrNotQualified        = errors.New("athlete does not meet qualification")
	ErrMissingHomeLocation = errors.New("team home location is required")
)

func RegistrationOpen(now, closesAt time.Time) bool {
	return !now.After(closesAt)
}

// RegistrationClosesAt is the local start of the stage minus the deadline.
// stageDate contributes its civil year-month-day; the clock is startHour:startMinute in timezone.
func RegistrationClosesAt(stageDate time.Time, timezone string, startHour, startMinute int, deadline time.Duration) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("location timezone: %w", err)
	}
	start := time.Date(stageDate.Year(), stageDate.Month(), stageDate.Day(), startHour, startMinute, 0, 0, loc)
	return start.Add(-deadline), nil
}

func ValidateRoster(
	members []AthleteQualification,
	homeEventCode string,
	now, closesAt time.Time,
) error {
	if !RegistrationOpen(now, closesAt) {
		return ErrRegistrationClosed
	}
	if homeEventCode == "" {
		return ErrMissingHomeLocation
	}
	if len(members) != TeamSize {
		return ErrWrongTeamSize
	}

	seen := make(map[int64]struct{}, TeamSize)
	females := 0
	for _, m := range members {
		if _, ok := seen[m.S95AthleteID]; ok {
			return ErrDuplicateAthlete
		}
		seen[m.S95AthleteID] = struct{}{}
		if m.Gender == GenderFemale {
			females++
		}
		if m.RunsCount < MinRuns && m.VolCount < MinVolunteering {
			return fmt.Errorf("%w: athlete %d", ErrNotQualified, m.S95AthleteID)
		}
	}
	if females < MinFemales {
		return ErrNotEnoughFemales
	}
	return nil
}
