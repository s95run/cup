package service

import (
	"context"
	"sort"
	"time"

	"github.com/s95run/cup/internal/domain"
	"github.com/s95run/cup/internal/repository"
)

type MemberView struct {
	AthleteID     int64  `json:"athlete_id"`
	Name          string `json:"name"`
	Gender        string `json:"gender"`
	HomeEvent     string `json:"home_event"`
	Club          string `json:"club"`
	FinishTimeSec *int   `json:"finish_time_sec"`
	Position      *int   `json:"position"`
	DNF           bool   `json:"dnf"`
}

type BonusView struct {
	ID     int64  `json:"id"`
	Points int    `json:"points"`
	Reason string `json:"reason"`
}

type TeamResultView struct {
	RegistrationID int64        `json:"registration_id"`
	TeamID         int64        `json:"team_id"`
	Name           string       `json:"name"`
	Slug           string       `json:"slug"`
	HomeEventCode  string       `json:"home_event_code"`
	ClubName       string       `json:"club_name"`
	Rank           *int         `json:"rank"`
	TotalTimeSec   *int         `json:"total_time_sec"`
	AvgTimeDS      *int         `json:"avg_time_ds"`
	BasePoints     int          `json:"base_points"`
	BonusPoints    int          `json:"bonus_points"`
	TotalPoints    int          `json:"total_points"`
	Disqualified   bool         `json:"disqualified"`
	DSQReason      string       `json:"dsq_reason"`
	Bonuses        []BonusView  `json:"bonuses"`
	Members        []MemberView `json:"members"`
}

type StageView struct {
	ID                   int64            `json:"id"`
	SeasonID             int64            `json:"season_id"`
	StageNumber          int              `json:"stage_number"`
	EventCode            string           `json:"event_code"`
	EventName            string           `json:"event_name"`
	StageDate            string           `json:"stage_date"`
	RegistrationClosesAt time.Time        `json:"registration_closes_at"`
	Status               string           `json:"status"`
	Teams                []TeamResultView `json:"teams"`
}

func (c *Cup) StageView(ctx context.Context, id int64) (StageView, error) {
	stage, err := c.Store.Stage(ctx, id)
	if err != nil {
		return StageView{}, err
	}
	regs, members, bonuses, results, err := c.Store.StageBundle(ctx, id)
	if err != nil {
		return StageView{}, err
	}
	mem := map[int64][]MemberView{}
	for _, member := range members {
		mem[member.RegistrationID] = append(mem[member.RegistrationID], MemberView{
			AthleteID: member.S95AthleteID, Name: member.Name, Gender: member.Gender,
			HomeEvent: member.HomeEvent, Club: member.Club, FinishTimeSec: member.FinishTimeSec,
			Position: member.ActivityPosition, DNF: member.IsDNF,
		})
	}
	bon := map[int64][]BonusView{}
	for _, bonus := range bonuses {
		bon[bonus.RegistrationID] = append(bon[bonus.RegistrationID], BonusView{ID: bonus.ID, Points: bonus.Points, Reason: bonus.Reason})
	}
	res := map[int64]repository.ResultRow{}
	for _, result := range results {
		res[result.RegistrationID] = result
	}
	view := StageView{
		ID: stage.ID, SeasonID: stage.SeasonID, StageNumber: stage.StageNumber,
		EventCode: stage.EventCode, EventName: stage.EventName,
		StageDate: stage.StageDate.Format("2006-01-02"), RegistrationClosesAt: stage.RegistrationClosesAt,
		Status: stage.Status,
	}
	for _, reg := range regs {
		item := TeamResultView{
			RegistrationID: reg.ID, TeamID: reg.TeamID, Name: reg.TeamName, Slug: reg.TeamSlug,
			HomeEventCode: reg.HomeEventCode, ClubName: reg.ClubName,
			Bonuses: bon[reg.ID], Members: mem[reg.ID],
		}
		if item.Bonuses == nil {
			item.Bonuses = []BonusView{}
		}
		if item.Members == nil {
			item.Members = []MemberView{}
		}
		if result, ok := res[reg.ID]; ok {
			item.Rank = result.Rank
			item.TotalTimeSec = result.TotalTimeSec
			item.AvgTimeDS = result.AvgTimeDS
			item.BasePoints = result.BasePoints
			item.BonusPoints = result.BonusPoints
			item.TotalPoints = result.TotalPoints
			item.Disqualified = result.Disqualified
			item.DSQReason = result.DSQReason
		}
		view.Teams = append(view.Teams, item)
	}
	sort.SliceStable(view.Teams, func(i, j int) bool {
		a, b := view.Teams[i], view.Teams[j]
		if a.Disqualified != b.Disqualified {
			return !a.Disqualified
		}
		if a.Rank != nil && b.Rank != nil && *a.Rank != *b.Rank {
			return *a.Rank < *b.Rank
		}
		return a.Name < b.Name
	})
	return view, nil
}

type StandingView struct {
	Place       int    `json:"place"`
	TeamID      int64  `json:"team_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Points      int    `json:"points"`
	BasePoints  int    `json:"base_points"`
	BonusPoints int    `json:"bonus_points"`
	Stages      int    `json:"stages"`
	FirstPlaces int    `json:"first_places"`
}

func (c *Cup) Standings(ctx context.Context, year int) ([]StandingView, error) {
	season, err := c.Store.SeasonByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	rows, err := c.Store.Standings(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	domainRows := make([]domain.SeasonStandingRow, len(rows))
	byID := map[int64]repository.StandingRow{}
	for i, row := range rows {
		domainRows[i] = domain.SeasonStandingRow{
			TeamID: row.TeamID, TotalPoints: row.TotalPoints, StagesCount: row.StagesCount,
			FirstPlaces: row.FirstPlaces, AvgTimeTotal: row.AvgTimeTotal,
		}
		byID[row.TeamID] = row
	}
	ranked := domain.RankSeason(domain.ScoringMode(season.ScoringMode), domainRows, season.MinStages)
	out := make([]StandingView, len(ranked))
	for i, row := range ranked {
		src := byID[row.TeamID]
		out[i] = StandingView{
			Place: i + 1, TeamID: row.TeamID, Name: src.Name, Slug: src.Slug,
			Points: row.TotalPoints, BasePoints: src.BasePoints, BonusPoints: src.BonusPoints,
			Stages: row.StagesCount, FirstPlaces: row.FirstPlaces,
		}
	}
	return out, nil
}
