package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", ErrConflict, pgErr.ConstraintName)
	}
	return err
}

func (s *Store) UpsertLocations(ctx context.Context, rows []Location) error {
	for _, row := range rows {
		tz := row.Timezone
		if tz == "" {
			tz = "Europe/Moscow"
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO locations_cache (code_name, name, town, timezone, active, synced_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (code_name) DO UPDATE
			SET name = EXCLUDED.name, town = EXCLUDED.town, timezone = EXCLUDED.timezone,
			    active = EXCLUDED.active, synced_at = now()`,
			row.CodeName, row.Name, row.Town, tz, row.Active)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListLocations(ctx context.Context) ([]Location, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT code_name, name, town, timezone, active
		FROM locations_cache ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Location
	for rows.Next() {
		var loc Location
		if err := rows.Scan(&loc.CodeName, &loc.Name, &loc.Town, &loc.Timezone, &loc.Active); err != nil {
			return nil, err
		}
		out = append(out, loc)
	}
	return out, rows.Err()
}

func (s *Store) Location(ctx context.Context, code string) (Location, error) {
	var loc Location
	err := s.pool.QueryRow(ctx, `
		SELECT code_name, name, town, timezone, active FROM locations_cache WHERE code_name = $1`, code).
		Scan(&loc.CodeName, &loc.Name, &loc.Town, &loc.Timezone, &loc.Active)
	return loc, mapErr(err)
}

func (s *Store) UserRoleByID(ctx context.Context, id int64) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, id).Scan(&role)
	return role, mapErr(err)
}

func (s *Store) EnsureUser(ctx context.Context, u User) (User, error) {
	if u.Role == "" {
		u.Role = "captain"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (s95_user_id, s95_athlete_id, athlete_code, name, role)
		VALUES (NULLIF($1, 0), $2, $3, $4, $5)
		ON CONFLICT (s95_athlete_id) DO UPDATE
		SET name = EXCLUDED.name,
		    athlete_code = EXCLUDED.athlete_code,
		    role = CASE WHEN users.role = 'admin' THEN users.role ELSE EXCLUDED.role END
		RETURNING id, COALESCE(s95_user_id, 0), s95_athlete_id, athlete_code, name, role`,
		u.S95UserID, u.S95AthleteID, u.AthleteCode, u.Name, u.Role).
		Scan(&u.ID, &u.S95UserID, &u.S95AthleteID, &u.AthleteCode, &u.Name, &u.Role)
	return u, err
}

func (s *Store) CreateSeason(ctx context.Context, season Season) (Season, error) {
	if season.ScoringMode == "" {
		season.ScoringMode = "dynamic_n"
	}
	if season.MinStages == 0 {
		season.MinStages = 2
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO seasons (year, slug, title, scoring_mode, min_stages_required, is_published)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		season.Year, season.Slug, season.Title, season.ScoringMode, season.MinStages, season.Published).
		Scan(&season.ID)
	return season, mapErr(err)
}

func (s *Store) UpdateSeason(ctx context.Context, season Season) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE seasons SET title = $2, scoring_mode = $3, min_stages_required = $4, is_published = $5
		WHERE id = $1`,
		season.ID, season.Title, season.ScoringMode, season.MinStages, season.Published)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListSeasons(ctx context.Context, publishedOnly bool) ([]Season, error) {
	q := `SELECT id, year, slug, title, scoring_mode, min_stages_required, is_published FROM seasons`
	if publishedOnly {
		q += ` WHERE is_published`
	}
	q += ` ORDER BY year DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSeasons(rows)
}

func (s *Store) SeasonByYear(ctx context.Context, year int) (Season, error) {
	var season Season
	err := s.pool.QueryRow(ctx, `
		SELECT id, year, slug, title, scoring_mode, min_stages_required, is_published
		FROM seasons WHERE year = $1`, year).
		Scan(&season.ID, &season.Year, &season.Slug, &season.Title, &season.ScoringMode, &season.MinStages, &season.Published)
	return season, mapErr(err)
}

func scanSeasons(rows pgx.Rows) ([]Season, error) {
	var out []Season
	for rows.Next() {
		var season Season
		if err := rows.Scan(&season.ID, &season.Year, &season.Slug, &season.Title, &season.ScoringMode, &season.MinStages, &season.Published); err != nil {
			return nil, err
		}
		out = append(out, season)
	}
	return out, rows.Err()
}

func (s *Store) CreateStage(ctx context.Context, stage Stage) (Stage, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO stages (season_id, stage_number, event_code, stage_date, registration_closes_at, status)
		VALUES ($1, $2, $3, $4, $5, 'upcoming')
		RETURNING id`,
		stage.SeasonID, stage.StageNumber, stage.EventCode, stage.StageDate, stage.RegistrationClosesAt).
		Scan(&stage.ID)
	if err != nil {
		return stage, mapErr(err)
	}
	stage.Status = "upcoming"
	return stage, nil
}

func (s *Store) ListStages(ctx context.Context, seasonID int64) ([]Stage, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.season_id, s.stage_number, s.event_code, l.name, l.timezone, s.stage_date,
		       s.registration_closes_at, s.status, s.s95_activity_id, s.s95_activity_updated_at
		FROM stages s
		JOIN locations_cache l ON l.code_name = s.event_code
		WHERE s.season_id = $1
		ORDER BY s.stage_number`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStages(rows)
}

func (s *Store) Stage(ctx context.Context, id int64) (Stage, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.season_id, s.stage_number, s.event_code, l.name, l.timezone, s.stage_date,
		       s.registration_closes_at, s.status, s.s95_activity_id, s.s95_activity_updated_at
		FROM stages s
		JOIN locations_cache l ON l.code_name = s.event_code
		WHERE s.id = $1`, id)
	if err != nil {
		return Stage{}, err
	}
	defer rows.Close()
	stages, err := scanStages(rows)
	if err != nil {
		return Stage{}, err
	}
	if len(stages) == 0 {
		return Stage{}, ErrNotFound
	}
	return stages[0], nil
}

func scanStages(rows pgx.Rows) ([]Stage, error) {
	var out []Stage
	for rows.Next() {
		var stage Stage
		if err := rows.Scan(&stage.ID, &stage.SeasonID, &stage.StageNumber, &stage.EventCode, &stage.EventName,
			&stage.Timezone, &stage.StageDate, &stage.RegistrationClosesAt, &stage.Status, &stage.ActivityID, &stage.ActivityUpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, stage)
	}
	return out, rows.Err()
}

func (s *Store) CloseDueRegistrations(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE stages SET status = 'registration_closed'
		WHERE status = 'upcoming' AND registration_closes_at <= $1`, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) StagesToSync(ctx context.Context, now time.Time) ([]Stage, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.season_id, s.stage_number, s.event_code, l.name, l.timezone, s.stage_date,
		       s.registration_closes_at, s.status, s.s95_activity_id, s.s95_activity_updated_at
		FROM stages s
		JOIN locations_cache l ON l.code_name = s.event_code
		WHERE s.stage_date <= $1::date
		  AND (s.status <> 'completed' OR s.stage_date >= ($1::date - 2))
		ORDER BY s.stage_date`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStages(rows)
}

func (s *Store) CreateTeam(ctx context.Context, team Team) (Team, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO teams (name, slug, home_event_code, club_name, captain_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		team.Name, team.Slug, team.HomeEventCode, team.ClubName, team.CaptainUserID).Scan(&team.ID)
	return team, mapErr(err)
}

func (s *Store) UpdateTeam(ctx context.Context, team Team) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE teams SET name = $2, home_event_code = $3, club_name = $4
		WHERE id = $1 AND captain_user_id = $5`,
		team.ID, team.Name, team.HomeEventCode, team.ClubName, team.CaptainUserID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

func (s *Store) TeamByID(ctx context.Context, id int64) (Team, error) {
	var team Team
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, home_event_code, club_name, captain_user_id
		FROM teams WHERE id = $1`, id).
		Scan(&team.ID, &team.Name, &team.Slug, &team.HomeEventCode, &team.ClubName, &team.CaptainUserID)
	return team, mapErr(err)
}

func (s *Store) TeamBySlug(ctx context.Context, slug string) (Team, error) {
	var team Team
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, home_event_code, club_name, captain_user_id
		FROM teams WHERE slug = $1`, slug).
		Scan(&team.ID, &team.Name, &team.Slug, &team.HomeEventCode, &team.ClubName, &team.CaptainUserID)
	return team, mapErr(err)
}

func (s *Store) TeamsByCaptain(ctx context.Context, userID int64) ([]Team, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, slug, home_event_code, club_name, captain_user_id
		FROM teams WHERE captain_user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.Name, &team.Slug, &team.HomeEventCode, &team.ClubName, &team.CaptainUserID); err != nil {
			return nil, err
		}
		out = append(out, team)
	}
	return out, rows.Err()
}

func (s *Store) RosterForCaptain(ctx context.Context, stageID, teamID, captainID int64) (int64, []MemberRow, error) {
	var regID int64
	var owner int64
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, t.captain_user_id
		FROM stage_registrations r
		JOIN teams t ON t.id = r.team_id
		WHERE r.stage_id = $1 AND r.team_id = $2`, stageID, teamID).Scan(&regID, &owner)
	if err != nil {
		return 0, nil, mapErr(err)
	}
	if owner != captainID {
		return 0, nil, ErrForbidden
	}
	all, err := s.members(ctx, stageID)
	if err != nil {
		return 0, nil, err
	}
	var out []MemberRow
	for _, row := range all {
		if row.RegistrationID == regID {
			out = append(out, row)
		}
	}
	return regID, out, nil
}

func (s *Store) SaveRoster(ctx context.Context, stageID, teamID, captainID int64, now time.Time, members []MemberRow) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var closes time.Time
	if err := tx.QueryRow(ctx, `SELECT registration_closes_at FROM stages WHERE id = $1 FOR UPDATE`, stageID).Scan(&closes); err != nil {
		return 0, mapErr(err)
	}
	if now.After(closes) {
		return 0, ErrConflict
	}
	var owner int64
	if err := tx.QueryRow(ctx, `SELECT captain_user_id FROM teams WHERE id = $1`, teamID).Scan(&owner); err != nil {
		return 0, mapErr(err)
	}
	if owner != captainID {
		return 0, ErrForbidden
	}
	var regID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO stage_registrations (stage_id, team_id) VALUES ($1, $2)
		ON CONFLICT (stage_id, team_id) DO UPDATE SET submitted_at = now()
		RETURNING id`, stageID, teamID).Scan(&regID)
	if err != nil {
		return 0, mapErr(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM stage_roster_members WHERE registration_id = $1`, regID); err != nil {
		return 0, err
	}
	for _, m := range members {
		if _, err := tx.Exec(ctx, `
			INSERT INTO stage_roster_members (
				registration_id, stage_id, s95_athlete_id, athlete_name, gender,
				home_event_snapshot, club_snapshot, qualification_runs, qualification_vols)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			regID, stageID, m.S95AthleteID, m.Name, m.Gender, m.HomeEvent, m.Club, m.QualRuns, m.QualVols); err != nil {
			return 0, mapErr(err)
		}
	}
	return regID, tx.Commit(ctx)
}

func (s *Store) DeleteRoster(ctx context.Context, stageID, teamID, captainID int64, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var closes time.Time
	var owner int64
	err = tx.QueryRow(ctx, `
		SELECT s.registration_closes_at, t.captain_user_id
		FROM stage_registrations r
		JOIN stages s ON s.id = r.stage_id
		JOIN teams t ON t.id = r.team_id
		WHERE r.stage_id = $1 AND r.team_id = $2
		FOR UPDATE OF s`, stageID, teamID).Scan(&closes, &owner)
	if err != nil {
		return mapErr(err)
	}
	if owner != captainID {
		return ErrForbidden
	}
	if now.After(closes) {
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM stage_registrations WHERE stage_id = $1 AND team_id = $2`, stageID, teamID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) StageBundle(ctx context.Context, stageID int64) ([]RegistrationHead, []MemberRow, []BonusRow, []ResultRow, error) {
	regs, err := s.registrationHeads(ctx, stageID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	members, err := s.members(ctx, stageID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	bonuses, err := s.bonuses(ctx, stageID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	results, err := s.results(ctx, stageID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return regs, members, bonuses, results, nil
}

func (s *Store) registrationHeads(ctx context.Context, stageID int64) ([]RegistrationHead, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.stage_id, r.team_id, t.name, t.slug, t.home_event_code, t.club_name, t.captain_user_id
		FROM stage_registrations r
		JOIN teams t ON t.id = r.team_id
		WHERE r.stage_id = $1
		ORDER BY t.name`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RegistrationHead
	for rows.Next() {
		var row RegistrationHead
		if err := rows.Scan(&row.ID, &row.StageID, &row.TeamID, &row.TeamName, &row.TeamSlug, &row.HomeEventCode, &row.ClubName, &row.CaptainUserID); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) members(ctx context.Context, stageID int64) ([]MemberRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT registration_id, s95_athlete_id, athlete_name, gender, home_event_snapshot, club_snapshot,
		       qualification_runs, qualification_vols, finish_time_sec, activity_position, is_dnf
		FROM stage_roster_members WHERE stage_id = $1 ORDER BY id`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberRow
	for rows.Next() {
		var row MemberRow
		if err := rows.Scan(&row.RegistrationID, &row.S95AthleteID, &row.Name, &row.Gender, &row.HomeEvent, &row.Club,
			&row.QualRuns, &row.QualVols, &row.FinishTimeSec, &row.ActivityPosition, &row.IsDNF); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) bonuses(ctx context.Context, stageID int64) ([]BonusRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, registration_id, points, reason FROM stage_bonus_awards WHERE stage_id = $1 ORDER BY id`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BonusRow
	for rows.Next() {
		var row BonusRow
		if err := rows.Scan(&row.ID, &row.RegistrationID, &row.Points, &row.Reason); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) results(ctx context.Context, stageID int64) ([]ResultRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT registration_id, is_disqualified, dsq_reason, total_time_sec, avg_time_ds, fifth_runner_time_sec,
		       rank, base_points, bonus_points, total_points
		FROM stage_team_results WHERE stage_id = $1`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ResultRow
	for rows.Next() {
		var row ResultRow
		if err := rows.Scan(&row.RegistrationID, &row.Disqualified, &row.DSQReason, &row.TotalTimeSec, &row.AvgTimeDS,
			&row.FifthRunnerTimeSec, &row.Rank, &row.BasePoints, &row.BonusPoints, &row.TotalPoints); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) AddBonus(ctx context.Context, stageID, registrationID, adminID int64, points int, reason string) (int64, error) {
	var id int64
	var gotStage int64
	if err := s.pool.QueryRow(ctx, `SELECT stage_id FROM stage_registrations WHERE id = $1`, registrationID).Scan(&gotStage); err != nil {
		return 0, mapErr(err)
	}
	if gotStage != stageID && stageID != 0 {
		return 0, ErrNotFound
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO stage_bonus_awards (registration_id, stage_id, points, reason, awarded_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		registrationID, gotStage, points, reason, adminID).Scan(&id)
	return id, err
}

func (s *Store) DeleteBonus(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM stage_bonus_awards WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) BonusStage(ctx context.Context, bonusID int64) (int64, error) {
	var stageID int64
	err := s.pool.QueryRow(ctx, `SELECT stage_id FROM stage_bonus_awards WHERE id = $1`, bonusID).Scan(&stageID)
	return stageID, mapErr(err)
}

type SavedResult struct {
	RegistrationID     int64
	AthleteID          int64
	FinishTimeSec      *int
	Position           *int
	IsDNF              bool
	Disqualified       bool
	DSQReason          string
	TotalTimeSec       *int
	AvgTimeDS          *int
	FifthRunnerTimeSec *int
	Rank               *int
	BasePoints         int
	BonusPoints        int
	TotalPoints        int
}

func (s *Store) SaveProtocol(ctx context.Context, stageID, activityID int64, updatedAt time.Time, rows []SavedResult) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	seenReg := map[int64]struct{}{}
	for _, row := range rows {
		if _, err := tx.Exec(ctx, `
			UPDATE stage_roster_members
			SET finish_time_sec = $1, activity_position = $2, is_dnf = $3
			WHERE registration_id = $4 AND s95_athlete_id = $5`,
			row.FinishTimeSec, row.Position, row.IsDNF, row.RegistrationID, row.AthleteID); err != nil {
			return err
		}
		if _, ok := seenReg[row.RegistrationID]; ok {
			continue
		}
		seenReg[row.RegistrationID] = struct{}{}
		if _, err := tx.Exec(ctx, `
			INSERT INTO stage_team_results (
				registration_id, stage_id, is_disqualified, dsq_reason, total_time_sec, avg_time_ds,
				fifth_runner_time_sec, rank, base_points, bonus_points, total_points)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (registration_id) DO UPDATE SET
				is_disqualified = EXCLUDED.is_disqualified,
				dsq_reason = EXCLUDED.dsq_reason,
				total_time_sec = EXCLUDED.total_time_sec,
				avg_time_ds = EXCLUDED.avg_time_ds,
				fifth_runner_time_sec = EXCLUDED.fifth_runner_time_sec,
				rank = EXCLUDED.rank,
				base_points = EXCLUDED.base_points,
				bonus_points = EXCLUDED.bonus_points,
				total_points = EXCLUDED.total_points`,
			row.RegistrationID, stageID, row.Disqualified, row.DSQReason, row.TotalTimeSec, row.AvgTimeDS,
			row.FifthRunnerTimeSec, row.Rank, row.BasePoints, row.BonusPoints, row.TotalPoints); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE stages SET status = 'completed', s95_activity_id = $2, s95_activity_updated_at = $3 WHERE id = $1`,
		stageID, activityID, updatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) TeamHistory(ctx context.Context, teamID int64) ([]Stage, []ResultRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.season_id, s.stage_number, s.event_code, l.name, l.timezone, s.stage_date,
		       s.registration_closes_at, s.status, s.s95_activity_id, s.s95_activity_updated_at,
		       COALESCE(res.is_disqualified, false), COALESCE(res.dsq_reason, ''), res.total_time_sec, res.avg_time_ds,
		       res.fifth_runner_time_sec, res.rank, COALESCE(res.base_points, 0), COALESCE(res.bonus_points, 0), COALESCE(res.total_points, 0),
		       r.id
		FROM stage_registrations r
		JOIN stages s ON s.id = r.stage_id
		JOIN locations_cache l ON l.code_name = s.event_code
		LEFT JOIN stage_team_results res ON res.registration_id = r.id
		WHERE r.team_id = $1
		ORDER BY s.stage_date`, teamID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var stages []Stage
	var results []ResultRow
	for rows.Next() {
		var stage Stage
		var result ResultRow
		if err := rows.Scan(&stage.ID, &stage.SeasonID, &stage.StageNumber, &stage.EventCode, &stage.EventName, &stage.Timezone,
			&stage.StageDate, &stage.RegistrationClosesAt, &stage.Status, &stage.ActivityID, &stage.ActivityUpdatedAt,
			&result.Disqualified, &result.DSQReason, &result.TotalTimeSec, &result.AvgTimeDS, &result.FifthRunnerTimeSec,
			&result.Rank, &result.BasePoints, &result.BonusPoints, &result.TotalPoints, &result.RegistrationID); err != nil {
			return nil, nil, err
		}
		stages = append(stages, stage)
		results = append(results, result)
	}
	return stages, results, rows.Err()
}

func (s *Store) Standings(ctx context.Context, seasonID int64) ([]StandingRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug,
		       COALESCE(SUM(res.total_points), 0),
		       COALESCE(SUM(res.base_points) FILTER (WHERE res.is_disqualified = false), 0),
		       COALESCE(SUM(res.bonus_points) FILTER (WHERE res.is_disqualified = false), 0),
		       COUNT(*) FILTER (WHERE res.is_disqualified = false),
		       COUNT(*) FILTER (WHERE res.rank = 1 AND res.is_disqualified = false),
		       COALESCE(SUM(res.avg_time_ds) FILTER (WHERE res.is_disqualified = false), 0)
		FROM teams t
		JOIN stage_registrations reg ON reg.team_id = t.id
		JOIN stages st ON st.id = reg.stage_id
		JOIN stage_team_results res ON res.registration_id = reg.id
		WHERE st.season_id = $1
		GROUP BY t.id, t.name, t.slug`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StandingRow
	for rows.Next() {
		var row StandingRow
		if err := rows.Scan(&row.TeamID, &row.Name, &row.Slug, &row.TotalPoints, &row.BasePoints, &row.BonusPoints, &row.StagesCount, &row.FirstPlaces, &row.AvgTimeTotal); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
