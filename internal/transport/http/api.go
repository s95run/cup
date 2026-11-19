package httptransport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"github.com/s95run/cup/internal/domain"
	"github.com/s95run/cup/internal/repository"
	"github.com/s95run/cup/internal/s95client"
	"github.com/s95run/cup/internal/service"
)

type session struct {
	UserID    int64
	AthleteID int64
	Role      string
	Name      string
}

func (s *Server) session(r *http.Request) (session, bool) {
	claims := s.sessionFromRequest(r)
	if claims == nil {
		return session{}, false
	}
	return session{
		UserID:    claimInt64(claims, "uid"),
		AthleteID: claimInt64(claims, "sub"),
		Role:      claimString(claims, "role"),
		Name:      claimString(claims, "name"),
	}, true
}

func claimInt64(claims jwt.MapClaims, key string) int64 {
	switch v := claims[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func claimString(claims jwt.MapClaims, key string) string {
	v, _ := claims[key].(string)
	return v
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.session(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.session(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if sess.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if s.store != nil {
			role, err := s.store.UserRoleByID(r.Context(), sess.UserID)
			if err != nil || role != "admin" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) ready(w http.ResponseWriter) bool {
	if s.store == nil || s.cup == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return false
	}
	return true
}

func (s *Server) handleLocations(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	rows, err := s.store.ListLocations(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []repository.Location{}
	}
	writeJSON(w, rows)
}

func (s *Server) handleSeasons(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	publishedOnly := true
	if r.URL.Query().Get("all") == "1" {
		if sess, ok := s.session(r); ok && sess.Role == "admin" {
			publishedOnly = false
		}
	}
	rows, err := s.store.ListSeasons(r.Context(), publishedOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []repository.Season{}
	}
	writeJSON(w, rows)
}

func (s *Server) handleSeason(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	year, _ := strconv.Atoi(chi.URLParam(r, "year"))
	season, err := s.store.SeasonByYear(r.Context(), year)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	stages, err := s.store.ListStages(r.Context(), season.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"season": season, "stages": stages})
}

func (s *Server) handleStandings(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	year, _ := strconv.Atoi(chi.URLParam(r, "year"))
	rows, err := s.cup.Standings(r.Context(), year)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, rows)
}

func (s *Server) handleStage(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	view, err := s.cup.StageView(r.Context(), id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, view)
}

func (s *Server) handleTeam(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	team, err := s.store.TeamBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	stages, results, err := s.store.TeamHistory(r.Context(), team.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"team": team, "stages": stages, "results": results})
}

func (s *Server) handleAthleteSearch(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	rows, err := s.s95.SearchAthletes(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	now := time.Now()
	out := make([]service.SignedAthlete, 0, len(rows))
	for _, row := range rows {
		out = append(out, service.SignAthleteCard(s.cfg.Auth.JWTSecret, row, now))
	}
	writeJSON(w, map[string]any{"athletes": out})
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	var body struct {
		Name          string `json:"name"`
		HomeEventCode string `json:"home_event_code"`
		ClubName      string `json:"club_name"`
	}
	if !decode(w, r, &body) || body.Name == "" || body.HomeEventCode == "" {
		http.Error(w, "name and home_event_code are required", http.StatusBadRequest)
		return
	}
	team, err := s.store.CreateTeam(r.Context(), repository.Team{
		Name: body.Name, Slug: service.Slug(body.Name), HomeEventCode: body.HomeEventCode,
		ClubName: body.ClubName, CaptainUserID: sess.UserID,
	})
	if errors.Is(err, repository.ErrConflict) {
		team, err = s.store.CreateTeam(r.Context(), repository.Team{
			Name: body.Name, Slug: service.Slug(body.Name) + "-" + strconv.FormatInt(sess.UserID, 10),
			HomeEventCode: body.HomeEventCode, ClubName: body.ClubName, CaptainUserID: sess.UserID,
		})
	}
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, team)
}

func (s *Server) handleUpdateTeam(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Name          string `json:"name"`
		HomeEventCode string `json:"home_event_code"`
		ClubName      string `json:"club_name"`
	}
	if !decode(w, r, &body) {
		return
	}
	err := s.store.UpdateTeam(r.Context(), repository.Team{
		ID: id, Name: body.Name, HomeEventCode: body.HomeEventCode, ClubName: body.ClubName, CaptainUserID: sess.UserID,
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetRegistration(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	stageID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	teamID, _ := strconv.ParseInt(r.URL.Query().Get("team_id"), 10, 64)
	regID, members, err := s.cup.CaptainRegistrationRoster(r.Context(), stageID, teamID, sess.UserID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	now := time.Now()
	athletes := make([]service.SignedAthlete, 0, len(members))
	for _, member := range members {
		athletes = append(athletes, service.SignStoredMember(
			s.cfg.Auth.JWTSecret, member.S95AthleteID, member.Name, member.Gender,
			member.HomeEvent, member.Club, member.QualRuns, member.QualVols, now,
		))
	}
	writeJSON(w, map[string]any{"registration_id": regID, "athletes": athletes})
}

func (s *Server) handlePutRegistration(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	stageID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		TeamID   int64                   `json:"team_id"`
		Athletes []service.SignedAthlete `json:"athletes"`
	}
	if !decode(w, r, &body) {
		return
	}
	now := time.Now()
	members := make([]s95client.AthleteSearchHit, 0, len(body.Athletes))
	for _, card := range body.Athletes {
		hit, err := service.VerifyAthleteCard(s.cfg.Auth.JWTSecret, card, now)
		if err != nil {
			writeAthleteCardErr(w, err)
			return
		}
		members = append(members, hit)
	}
	id, err := s.cup.SaveRegistration(r.Context(), service.RosterRequest{
		StageID: stageID, TeamID: body.TeamID, CaptainID: sess.UserID, Members: members,
	})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"registration_id": id})
}

func writeAthleteCardErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAthleteCardExpired), errors.Is(err, service.ErrAthleteCardInvalid):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (s *Server) handleDeleteRegistration(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	stageID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	teamID, _ := strconv.ParseInt(r.URL.Query().Get("team_id"), 10, 64)
	err := s.store.DeleteRoster(r.Context(), stageID, teamID, sess.UserID, time.Now())
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateSeason(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	var body repository.Season
	if !decode(w, r, &body) {
		return
	}
	season, err := s.store.CreateSeason(r.Context(), body)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, season)
}

func (s *Server) handleUpdateSeason(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body repository.Season
	if !decode(w, r, &body) {
		return
	}
	body.ID = id
	if err := s.store.UpdateSeason(r.Context(), body); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateStage(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	var body struct {
		SeasonID    int64  `json:"season_id"`
		StageNumber int    `json:"stage_number"`
		EventCode   string `json:"event_code"`
		StageDate   string `json:"stage_date"`
	}
	if !decode(w, r, &body) {
		return
	}
	date, err := time.Parse("2006-01-02", body.StageDate)
	if err != nil {
		http.Error(w, "stage_date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	stage, err := s.cup.CreateStage(r.Context(), body.SeasonID, body.StageNumber, body.EventCode, date)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, stage)
}

func (s *Server) handleAddBonus(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	regID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Points int    `json:"points"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &body) || body.Points < 1 || body.Points > 20 || len(body.Reason) < 3 || len(body.Reason) > 200 {
		http.Error(w, "points 1..20 and reason 3..200 chars are required", http.StatusBadRequest)
		return
	}
	id, err := s.store.AddBonus(r.Context(), 0, regID, sess.UserID, body.Points, body.Reason)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if stageID, err := s.store.BonusStage(r.Context(), id); err == nil {
		_ = s.cup.Rescore(r.Context(), stageID)
	}
	writeJSON(w, map[string]any{"id": id})
}

func (s *Server) handleDeleteBonus(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	stageID, err := s.store.BonusStage(r.Context(), id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if err := s.store.DeleteBonus(r.Context(), id); err != nil {
		writeStoreErr(w, err)
		return
	}
	_ = s.cup.Rescore(r.Context(), stageID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSyncStage(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := s.cup.SyncStage(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	view, err := s.cup.StageView(r.Context(), id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, view)
}

func (s *Server) handleMyTeams(w http.ResponseWriter, r *http.Request) {
	if !s.ready(w) {
		return
	}
	sess, _ := s.session(r)
	rows, err := s.store.TeamsByCaptain(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []repository.Team{}
	}
	writeJSON(w, rows)
}

func decode(w http.ResponseWriter, r *http.Request, dest any) bool {
	defer func() { _ = r.Body.Close() }()
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dest); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return false
	}
	return true
}

func writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, repository.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, repository.ErrConflict), errors.Is(err, domain.ErrRegistrationClosed):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrNotQualified), errors.Is(err, domain.ErrWrongTeamSize),
		errors.Is(err, domain.ErrNotEnoughFemales), errors.Is(err, domain.ErrDuplicateAthlete),
		errors.Is(err, domain.ErrMissingHomeLocation):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
