package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/s95run/cup/internal/config"
	"github.com/s95run/cup/internal/repository"
	"github.com/s95run/cup/internal/s95client"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := config.DBConfig{
		Host: "localhost", Port: 5433, DBName: "cup_dev", User: "cup_app",
		Password: "cup_app_dev", SSLMode: "disable", Schema: "cup",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := repository.NewPool(ctx, cfg)
	if err != nil {
		t.Skip(err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skip(err)
	}
	if err := repository.RunMigrations(ctx, cfg, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestProtocolKeepsBonus(t *testing.T) {
	pool := testPool(t)
	store := repository.NewStore(pool)
	ctx := context.Background()
	cup := &Cup{
		Store: store, S95: s95client.NewMock(), Deadline: 7 * 24 * time.Hour,
		Now: func() time.Time { return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) },
	}
	if err := cup.SyncLocations(ctx); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	user, err := store.EnsureUser(ctx, repository.User{
		S95AthleteID: suffix, Name: "Капитан тест", Role: "captain",
	})
	if err != nil {
		t.Fatal(err)
	}
	season, err := store.CreateSeason(ctx, repository.Season{
		Year: int(3000 + suffix%100000), Slug: fmt.Sprintf("it-%d", suffix), Title: "integration",
		ScoringMode: "dynamic_n", MinStages: 2, Published: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM seasons WHERE id = $1`, season.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM teams WHERE captain_user_id = $1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
	})

	stage, err := cup.CreateStage(ctx, season.ID, 1, "angarskieprudy", time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.CreateTeam(ctx, repository.Team{
		Name: "Интеграция", Slug: fmt.Sprintf("it-team-%d", suffix), HomeEventCode: "angarskieprudy", CaptainUserID: user.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	mock := cup.S95.(*s95client.Mock)
	members := make([]s95client.AthleteSearchHit, 0, 5)
	for _, id := range []int64{101, 102, 103, 104, 105} {
		for _, athlete := range mock.Athletes {
			if athlete.ID == id {
				members = append(members, athlete)
				break
			}
		}
	}
	regID, err := cup.SaveRegistration(ctx, RosterRequest{
		StageID: stage.ID, TeamID: team.ID, CaptainID: user.ID, Members: members,
	})
	if err != nil {
		t.Fatal(err)
	}

	missing := s95client.ProtocolResponse{Published: true, ActivityID: 1, UpdatedAt: time.Now().Add(-time.Hour)}
	for _, id := range []int64{101, 102, 103, 104} {
		missing.Results = append(missing.Results, s95client.ProtocolResult{AthleteID: id, Position: 1, TotalTimeSec: 1500})
	}
	if err := cup.Recalculate(ctx, stage.ID, missing); err != nil {
		t.Fatal(err)
	}
	view, err := cup.StageView(ctx, stage.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Teams) != 1 || !view.Teams[0].Disqualified || view.Teams[0].TotalPoints != 0 {
		t.Fatalf("expected dsq, got %+v", view.Teams[0])
	}

	full := missing
	full.UpdatedAt = time.Now()
	full.Results = append(full.Results, s95client.ProtocolResult{AthleteID: 105, Position: 5, TotalTimeSec: 1600})
	if err := cup.Recalculate(ctx, stage.ID, full); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddBonus(ctx, stage.ID, regID, user.ID, 2, "Единая форма команды"); err != nil {
		t.Fatal(err)
	}
	if err := cup.Rescore(ctx, stage.ID); err != nil {
		t.Fatal(err)
	}
	view, err = cup.StageView(ctx, stage.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := view.Teams[0]
	if got.Disqualified || got.BasePoints != 1 || got.BonusPoints != 2 || got.TotalPoints != 3 || len(got.Bonuses) != 1 {
		t.Fatalf("scored team %+v", got)
	}
	if got.AvgTimeDS == nil {
		t.Fatal("missing average")
	}
}
