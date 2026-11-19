package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/s95run/cup/internal/domain"
	"github.com/s95run/cup/internal/repository"
	"github.com/s95run/cup/internal/s95client"
)

type Cup struct {
	Store    *repository.Store
	S95      s95client.API
	Deadline time.Duration
	Now      func() time.Time
}

func (c *Cup) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Cup) SyncLocations(ctx context.Context) error {
	events, err := c.S95.ListEvents(ctx)
	if err != nil {
		return err
	}
	rows := make([]repository.Location, 0, len(events))
	for _, event := range events {
		if event.CodeName == "" {
			continue
		}
		rows = append(rows, repository.Location{
			CodeName: event.CodeName,
			Name:     event.Name,
			Town:     event.Town,
			Timezone: "Europe/Moscow",
			Active:   event.Active,
		})
	}
	return c.Store.UpsertLocations(ctx, rows)
}

func (c *Cup) CloseRegistrations(ctx context.Context) (int64, error) {
	return c.Store.CloseDueRegistrations(ctx, c.now())
}

type RosterRequest struct {
	StageID   int64
	TeamID    int64
	CaptainID int64
	Members   []s95client.AthleteSearchHit
}

func (c *Cup) SaveRegistration(ctx context.Context, req RosterRequest) (int64, error) {
	stage, err := c.Store.Stage(ctx, req.StageID)
	if err != nil {
		return 0, err
	}
	team, err := c.Store.TeamByID(ctx, req.TeamID)
	if err != nil {
		return 0, err
	}
	if team.CaptainUserID != req.CaptainID {
		return 0, repository.ErrForbidden
	}
	qual := make([]domain.AthleteQualification, 0, len(req.Members))
	rows := make([]repository.MemberRow, 0, len(req.Members))
	seen := map[int64]struct{}{}
	for _, athlete := range req.Members {
		if athlete.ID == 0 {
			return 0, fmt.Errorf("%w: missing athlete id", domain.ErrNotQualified)
		}
		if _, ok := seen[athlete.ID]; ok {
			return 0, domain.ErrDuplicateAthlete
		}
		seen[athlete.ID] = struct{}{}
		qual = append(qual, domain.AthleteQualification{
			S95AthleteID: athlete.ID,
			Gender:       domain.Gender(athlete.Gender),
			RunsCount:    athlete.RunsCount,
			VolCount:     athlete.VolunteeringCount,
		})
		rows = append(rows, repository.MemberRow{
			S95AthleteID: athlete.ID,
			Name:         athlete.Name,
			Gender:       athlete.Gender,
			HomeEvent:    athlete.HomeEventName,
			Club:         athlete.ClubName,
			QualRuns:     athlete.RunsCount,
			QualVols:     athlete.VolunteeringCount,
		})
	}
	if err := domain.ValidateRoster(qual, team.HomeEventCode, c.now(), stage.RegistrationClosesAt); err != nil {
		return 0, err
	}
	return c.Store.SaveRoster(ctx, req.StageID, req.TeamID, req.CaptainID, c.now(), rows)
}

func (c *Cup) CaptainRegistrationRoster(ctx context.Context, stageID, teamID, captainID int64) (int64, []repository.MemberRow, error) {
	return c.Store.RosterForCaptain(ctx, stageID, teamID, captainID)
}

func (c *Cup) CreateStage(ctx context.Context, seasonID int64, number int, eventCode string, date time.Time) (repository.Stage, error) {
	loc, err := c.Store.Location(ctx, eventCode)
	if err != nil {
		return repository.Stage{}, err
	}
	closes, err := domain.RegistrationClosesAt(date, loc.Timezone, 9, 0, c.Deadline)
	if err != nil {
		return repository.Stage{}, err
	}
	return c.Store.CreateStage(ctx, repository.Stage{
		SeasonID:             seasonID,
		StageNumber:          number,
		EventCode:            eventCode,
		StageDate:            date,
		RegistrationClosesAt: closes,
	})
}

func (c *Cup) Recalculate(ctx context.Context, stageID int64, protocol s95client.ProtocolResponse) error {
	return c.applyProtocol(ctx, stageID, protocol, false)
}

// Rescore reapplies the stored finish times after a bonus change and keeps manual awards.
func (c *Cup) Rescore(ctx context.Context, stageID int64) error {
	stage, err := c.Store.Stage(ctx, stageID)
	if err != nil || stage.Status != "completed" {
		return err
	}
	_, members, _, _, err := c.Store.StageBundle(ctx, stageID)
	if err != nil {
		return err
	}
	protocol := s95client.ProtocolResponse{Published: true, UpdatedAt: c.now()}
	if stage.ActivityID != nil {
		protocol.ActivityID = *stage.ActivityID
	}
	for _, member := range members {
		if member.FinishTimeSec == nil {
			continue
		}
		pos := 0
		if member.ActivityPosition != nil {
			pos = *member.ActivityPosition
		}
		protocol.Results = append(protocol.Results, s95client.ProtocolResult{
			AthleteID: member.S95AthleteID, Position: pos, TotalTimeSec: *member.FinishTimeSec,
		})
	}
	return c.applyProtocol(ctx, stageID, protocol, true)
}

func (c *Cup) applyProtocol(ctx context.Context, stageID int64, protocol s95client.ProtocolResponse, force bool) error {
	if !protocol.Published {
		return nil
	}
	stage, err := c.Store.Stage(ctx, stageID)
	if err != nil {
		return err
	}
	if !force && stage.ActivityUpdatedAt != nil && !protocol.UpdatedAt.After(*stage.ActivityUpdatedAt) {
		return nil
	}
	regs, members, bonuses, _, err := c.Store.StageBundle(ctx, stageID)
	if err != nil {
		return err
	}
	finish := map[int64]s95client.ProtocolResult{}
	for _, result := range protocol.Results {
		finish[result.AthleteID] = result
	}
	bonusByReg := map[int64]int{}
	for _, bonus := range bonuses {
		bonusByReg[bonus.RegistrationID] += bonus.Points
	}
	membersByReg := map[int64][]repository.MemberRow{}
	for _, member := range members {
		membersByReg[member.RegistrationID] = append(membersByReg[member.RegistrationID], member)
	}
	inputs := make([]domain.TeamStageInput, 0, len(regs))
	order := make([]int64, 0, len(regs))
	for _, reg := range regs {
		input := domain.TeamStageInput{RegistrationID: reg.ID, BonusPoints: bonusByReg[reg.ID]}
		for _, member := range membersByReg[reg.ID] {
			row := domain.RosterMember{S95AthleteID: member.S95AthleteID, Gender: domain.Gender(member.Gender)}
			if hit, ok := finish[member.S95AthleteID]; ok {
				sec := hit.TotalTimeSec
				row.FinishTimeSec = &sec
			} else {
				row.IsDNF = true
			}
			input.Members = append(input.Members, row)
		}
		inputs = append(inputs, input)
		order = append(order, reg.ID)
	}
	outcomes := domain.ScoreStage(inputs)
	byReg := map[int64]domain.StageTeamOutcome{}
	for _, outcome := range outcomes {
		byReg[outcome.RegistrationID] = outcome
	}
	var saved []repository.SavedResult
	for _, regID := range order {
		outcome := byReg[regID]
		for _, member := range membersByReg[regID] {
			row := repository.SavedResult{
				RegistrationID:     regID,
				AthleteID:          member.S95AthleteID,
				Disqualified:       outcome.Disqualified,
				DSQReason:          outcome.DSQReason,
				TotalTimeSec:       outcome.TotalTimeSec,
				AvgTimeDS:          outcome.AvgTimeDS,
				FifthRunnerTimeSec: outcome.FifthRunnerTimeSec,
				Rank:               outcome.Rank,
				BasePoints:         outcome.BasePoints,
				BonusPoints:        outcome.BonusPoints,
				TotalPoints:        outcome.TotalPoints,
				IsDNF:              true,
			}
			if hit, ok := finish[member.S95AthleteID]; ok {
				sec := hit.TotalTimeSec
				pos := hit.Position
				row.FinishTimeSec = &sec
				row.Position = &pos
				row.IsDNF = false
			}
			saved = append(saved, row)
		}
	}
	return c.Store.SaveProtocol(ctx, stageID, protocol.ActivityID, protocol.UpdatedAt, saved)
}

func (c *Cup) SyncStage(ctx context.Context, stageID int64) error {
	stage, err := c.Store.Stage(ctx, stageID)
	if err != nil {
		return err
	}
	protocol, err := c.S95.FetchProtocol(ctx, stage.EventCode, stage.StageDate.Format("2006-01-02"), stage.ActivityUpdatedAt)
	if err != nil {
		if errors.Is(err, s95client.ErrNotModified) {
			return nil
		}
		return err
	}
	return c.applyProtocol(ctx, stageID, *protocol, false)
}

func (c *Cup) SyncDueProtocols(ctx context.Context) error {
	stages, err := c.Store.StagesToSync(ctx, c.now())
	if err != nil {
		return err
	}
	var first error
	for _, stage := range stages {
		if err := c.SyncStage(ctx, stage.ID); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func Slug(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(name) {
		if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fmt.Sprintf("team-%d", time.Now().Unix())
	}
	return out
}
