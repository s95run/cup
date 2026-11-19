package domain

import (
	"testing"
)

func ptr(i int) *int { return &i }

func TestScoreStage_dynamicPointsAndDSQ(t *testing.T) {
	teams := []TeamStageInput{
		{
			RegistrationID: 1,
			Members: []RosterMember{
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1100)},
				{FinishTimeSec: ptr(1200)},
				{FinishTimeSec: ptr(1300)},
				{FinishTimeSec: ptr(1400)}, // sum 6000, avg 1200
			},
		},
		{
			RegistrationID: 2,
			Members: []RosterMember{
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)}, // sum 5000 wins
			},
			BonusPoints: 2,
		},
		{
			RegistrationID: 3,
			Members: []RosterMember{
				{FinishTimeSec: ptr(1000)},
				{IsDNF: true},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
			},
		},
	}

	out := ScoreStage(teams)
	if len(out) != 3 {
		t.Fatalf("len %d", len(out))
	}
	// team 2 rank 1 -> 2 points (N=2 classified) + 2 bonus
	if out[1].Rank == nil || *out[1].Rank != 1 || out[1].BasePoints != 2 || out[1].TotalPoints != 4 {
		t.Fatalf("team2: %+v", out[1])
	}
	if !out[2].Disqualified || out[2].TotalPoints != 0 {
		t.Fatalf("team3 dsq: %+v", out[2])
	}
}

func TestScoreStage_tieBreakFifthRunner(t *testing.T) {
	teams := []TeamStageInput{
		{
			RegistrationID: 1,
			Members: []RosterMember{
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1200)},
			},
		},
		{
			RegistrationID: 2,
			Members: []RosterMember{
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1000)},
				{FinishTimeSec: ptr(1100)},
			},
		},
	}
	out := ScoreStage(teams)
	if out[1].Rank == nil || *out[1].Rank != 1 {
		t.Fatalf("faster fifth should win: %+v vs %+v", out[0], out[1])
	}
}

func TestScoreStage_avgTimeDS(t *testing.T) {
	teams := []TeamStageInput{{
		RegistrationID: 1,
		Members: []RosterMember{
			{FinishTimeSec: ptr(1000)},
			{FinishTimeSec: ptr(1100)},
			{FinishTimeSec: ptr(1200)},
			{FinishTimeSec: ptr(1300)},
			{FinishTimeSec: ptr(1401)}, // sum 6001
		},
	}}
	out := ScoreStage(teams)
	if out[0].AvgTimeDS == nil || *out[0].AvgTimeDS != 12002 {
		t.Fatalf("avg tenths: %+v", out[0])
	}
}

func TestScoreStage_tieBreakCascadesToFourth(t *testing.T) {
	// Same total 6000 and same slowest (1600). Faster 4th runner wins.
	slow := []RosterMember{
		{FinishTimeSec: ptr(1000)},
		{FinishTimeSec: ptr(1000)},
		{FinishTimeSec: ptr(1000)},
		{FinishTimeSec: ptr(1400)},
		{FinishTimeSec: ptr(1600)},
	}
	fastFourth := []RosterMember{
		{FinishTimeSec: ptr(1100)},
		{FinishTimeSec: ptr(1100)},
		{FinishTimeSec: ptr(1100)},
		{FinishTimeSec: ptr(1100)},
		{FinishTimeSec: ptr(1600)},
	}
	out := ScoreStage([]TeamStageInput{
		{RegistrationID: 1, Members: slow},
		{RegistrationID: 2, Members: fastFourth},
	})
	if out[1].Rank == nil || *out[1].Rank != 1 {
		t.Fatalf("faster fourth should win: %+v vs %+v", out[0], out[1])
	}
}

func TestScoreStage_dsqDropsBonus(t *testing.T) {
	out := ScoreStage([]TeamStageInput{{
		RegistrationID: 1,
		BonusPoints:    5,
		Members: []RosterMember{
			{FinishTimeSec: ptr(1000)},
			{IsDNF: true},
			{FinishTimeSec: ptr(1000)},
			{FinishTimeSec: ptr(1000)},
			{FinishTimeSec: ptr(1000)},
		},
	}})
	if out[0].TotalPoints != 0 || out[0].BasePoints != 0 {
		t.Fatalf("dsq must score zero: %+v", out[0])
	}
}

func TestRankSeason_filtersMinStagesAndPoints(t *testing.T) {
	rows := []SeasonStandingRow{
		{TeamID: 1, TotalPoints: 40, StagesCount: 1, FirstPlaces: 1, AvgTimeTotal: 1000},
		{TeamID: 2, TotalPoints: 30, StagesCount: 2, FirstPlaces: 0, AvgTimeTotal: 2000},
		{TeamID: 3, TotalPoints: 30, StagesCount: 2, FirstPlaces: 1, AvgTimeTotal: 2500},
	}
	got := RankSeason(ScoringModeDynamicN, rows, 2)
	if len(got) != 2 || got[0].TeamID != 3 || got[1].TeamID != 2 {
		t.Fatalf("standings %+v", got)
	}
}

func TestRankSeason_avgTimeMode(t *testing.T) {
	rows := []SeasonStandingRow{
		{TeamID: 1, TotalPoints: 10, StagesCount: 2, AvgTimeTotal: 3000},
		{TeamID: 2, TotalPoints: 50, StagesCount: 2, AvgTimeTotal: 2500},
		{TeamID: 3, TotalPoints: 99, StagesCount: 1, AvgTimeTotal: 1000},
	}
	got := RankSeason(ScoringModeAvgTime, rows, 2)
	if len(got) != 2 || got[0].TeamID != 2 {
		t.Fatalf("avg time standings %+v", got)
	}
}
