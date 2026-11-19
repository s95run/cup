package domain

import (
	"sort"
)

// ScoreStage ranks teams, applies DSQ rules, dynamic N..1 base points, and bonus totals.
func ScoreStage(teams []TeamStageInput) []StageTeamOutcome {
	out := make([]StageTeamOutcome, 0, len(teams))
	classified := make([]struct {
		idx           int
		totalTime     int
		fifthRunner   int
		slowestToFast []int
	}, 0)

	for _, t := range teams {
		o := StageTeamOutcome{
			RegistrationID: t.RegistrationID,
			BonusPoints:    t.BonusPoints,
		}
		if len(t.Members) != TeamSize {
			o.Disqualified = true
			o.DSQReason = "invalid roster size"
			out = append(out, o)
			continue
		}

		times := make([]int, 0, TeamSize)
		for _, m := range t.Members {
			if m.IsDNF || m.FinishTimeSec == nil {
				o.Disqualified = true
				o.DSQReason = "missing member at stage"
				break
			}
			times = append(times, *m.FinishTimeSec)
		}
		if o.Disqualified {
			out = append(out, o)
			continue
		}

		total := 0
		for _, sec := range times {
			total += sec
		}
		sort.Ints(times)
		fifth := times[len(times)-1]
		slowToFast := append([]int(nil), times...)
		for i, j := 0, len(slowToFast)-1; i < j; i, j = i+1, j-1 {
			slowToFast[i], slowToFast[j] = slowToFast[j], slowToFast[i]
		}

		avgDS := total * 10 / TeamSize
		o.TotalTimeSec = &total
		o.AvgTimeDS = &avgDS
		o.FifthRunnerTimeSec = &fifth
		out = append(out, o)
		classified = append(classified, struct {
			idx           int
			totalTime     int
			fifthRunner   int
			slowestToFast []int
		}{idx: len(out) - 1, totalTime: total, fifthRunner: fifth, slowestToFast: slowToFast})
	}

	sort.SliceStable(classified, func(i, j int) bool {
		if classified[i].totalTime != classified[j].totalTime {
			return classified[i].totalTime < classified[j].totalTime
		}
		return tieBreak(classified[i].slowestToFast, classified[j].slowestToFast)
	})

	n := len(classified)
	for rank, c := range classified {
		r := rank + 1
		out[c.idx].Rank = &r
		out[c.idx].BasePoints = n - rank
		out[c.idx].TotalPoints = out[c.idx].BasePoints + out[c.idx].BonusPoints
	}

	for i := range out {
		if out[i].Disqualified {
			out[i].BasePoints = 0
			out[i].TotalPoints = 0
		}
	}
	return out
}

func tieBreak(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

type SeasonStandingRow struct {
	TeamID       int64
	TotalPoints  int
	StagesCount  int
	FirstPlaces  int
	AvgTimeTotal int
}

// RankSeason keeps teams with at least minStages classified finishes and orders them.
// dynamic_n: total points, then more wins, then lower summed average time.
// avg_time: lower summed average time, then more stages, then more wins.
func RankSeason(mode ScoringMode, rows []SeasonStandingRow, minStages int) []SeasonStandingRow {
	eligible := make([]SeasonStandingRow, 0, len(rows))
	for _, row := range rows {
		if row.StagesCount >= minStages {
			eligible = append(eligible, row)
		}
	}
	if mode == ScoringModeAvgTime {
		sort.SliceStable(eligible, func(i, j int) bool {
			if eligible[i].AvgTimeTotal != eligible[j].AvgTimeTotal {
				return eligible[i].AvgTimeTotal < eligible[j].AvgTimeTotal
			}
			if eligible[i].StagesCount != eligible[j].StagesCount {
				return eligible[i].StagesCount > eligible[j].StagesCount
			}
			return eligible[i].FirstPlaces > eligible[j].FirstPlaces
		})
		return eligible
	}
	return SeasonStandings(eligible)
}

// SeasonStandings orders an already filtered points table.
func SeasonStandings(rows []SeasonStandingRow) []SeasonStandingRow {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].TotalPoints != rows[j].TotalPoints {
			return rows[i].TotalPoints > rows[j].TotalPoints
		}
		if rows[i].FirstPlaces != rows[j].FirstPlaces {
			return rows[i].FirstPlaces > rows[j].FirstPlaces
		}
		return rows[i].AvgTimeTotal < rows[j].AvgTimeTotal
	})
	return rows
}
