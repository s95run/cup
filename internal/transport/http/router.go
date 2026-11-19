package httptransport

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"

	"github.com/s95run/cup/internal/config"
	"github.com/s95run/cup/internal/repository"
	"github.com/s95run/cup/internal/s95client"
	"github.com/s95run/cup/internal/service"
)

type Deps struct {
	Store *repository.Store
	S95   s95client.API
	JTI   repository.JTIChecker
}

type Server struct {
	cfg   *config.Config
	s95   s95client.API
	jtis  repository.JTIChecker
	store *repository.Store
	cup   *service.Cup
}

func NewServer(cfg *config.Config, deps Deps) *Server {
	api := deps.S95
	if api == nil {
		if cfg.S95.Mock {
			api = s95client.NewMock()
		} else {
			api = s95client.New(cfg.S95.BaseURL, cfg.S95.APIKey, cfg.S95.RateLimitRPS)
		}
	}
	jtis := deps.JTI
	if jtis == nil {
		jtis = repository.NewInMemorySSOTokens()
	}
	var cup *service.Cup
	if deps.Store != nil {
		cup = &service.Cup{Store: deps.Store, S95: api, Deadline: cfg.Registration.DeadlineBeforeStage}
	}
	return &Server{cfg: cfg, s95: api, jtis: jtis, store: deps.Store, cup: cup}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)
	r.Use(csrfMiddleware(s.cfg.Server.CookieSecure))

	r.Get("/api/v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/locations", s.handleLocations)
		api.Get("/seasons", s.handleSeasons)
		api.Get("/seasons/{year}", s.handleSeason)
		api.Get("/seasons/{year}/standings", s.handleStandings)
		api.Get("/stages/{id}", s.handleStage)
		api.Get("/teams/{slug}", s.handleTeam)

		api.Route("/auth", func(auth chi.Router) {
			auth.Get("/csrf", func(w http.ResponseWriter, r *http.Request) {
				token, _ := r.Context().Value(csrfContextKey{}).(string)
				writeJSON(w, map[string]string{"token": token})
			})
			auth.Get("/me", s.handleAuthMe)
			auth.Get("/s95/login", s.handleS95Login)
			auth.Get("/s95/callback", s.handleS95Callback)
			auth.Post("/logout", s.handleLogout)
			if s.cfg.Auth.DevAuthEnabled {
				auth.Get("/dev-login", func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(w, map[string]bool{"enabled": true})
				})
				auth.Post("/dev-login", s.handleDevLogin)
			}
		})

		api.Group(func(priv chi.Router) {
			priv.Use(s.requireAuth)
			priv.Get("/captain/teams", s.handleMyTeams)
			priv.Get("/athletes/search", s.handleAthleteSearch)
			priv.Post("/teams", s.handleCreateTeam)
			priv.Patch("/teams/{id}", s.handleUpdateTeam)
			priv.Get("/stages/{id}/registration", s.handleGetRegistration)
			priv.Put("/stages/{id}/registration", s.handlePutRegistration)
			priv.Delete("/stages/{id}/registration", s.handleDeleteRegistration)
		})

		api.Group(func(admin chi.Router) {
			admin.Use(s.requireAdmin)
			admin.Post("/admin/seasons", s.handleCreateSeason)
			admin.Patch("/admin/seasons/{id}", s.handleUpdateSeason)
			admin.Post("/admin/stages", s.handleCreateStage)
			admin.Post("/admin/registrations/{id}/bonuses", s.handleAddBonus)
			admin.Delete("/admin/bonuses/{id}", s.handleDeleteBonus)
			admin.Post("/admin/stages/{id}/sync", s.handleSyncStage)
		})
	})
	return r
}

func (s *Server) sessionFromRequest(r *http.Request) jwt.MapClaims {
	c, err := r.Cookie("cup_session")
	if err != nil || c.Value == "" {
		return nil
	}
	tok, err := jwt.Parse(c.Value, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return nil
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}
	return claims
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.session(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]any{"user_id": sess.UserID, "athlete_id": sess.AthleteID, "role": sess.Role, "name": sess.Name})
}

func (s *Server) handleS95Login(w http.ResponseWriter, r *http.Request) {
	state := randomToken()
	http.SetCookie(w, &http.Cookie{
		Name: "cup_oauth_state", Value: state, Path: "/", HttpOnly: true,
		Secure: s.cfg.Server.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: 600,
	})
	redirect := s.cfg.S95.BaseURL + "/cup_sso?return_to=" + s.cfg.Server.PublicURL + "/api/v1/auth/s95/callback&state=" + state
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (s *Server) handleS95Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("cup_oauth_state")
	queryState := r.URL.Query().Get("state")
	if err != nil || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(queryState)) != 1 {
		http.Error(w, "state mismatch", http.StatusUnauthorized)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	payload, err := service.VerifySSOToken(s.cfg.S95.SSOSecret, token, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if payload.State != "" && payload.State != stateCookie.Value {
		http.Error(w, "state mismatch", http.StatusUnauthorized)
		return
	}
	expiresAt := time.Unix(payload.Exp, 0)
	if err := s.jtis.ConsumeJTI(r.Context(), payload.JTI, expiresAt); err != nil {
		http.Error(w, "sso token replay", http.StatusUnauthorized)
		return
	}
	role := s.roleForAthleteCode(payload.AthleteCode)
	var userID int64
	if s.store != nil {
		user, err := s.store.EnsureUser(r.Context(), repository.User{
			S95UserID: payload.UserID, S95AthleteID: payload.AthleteID, AthleteCode: payload.AthleteCode,
			Name: payload.Name, Role: role,
		})
		if err != nil {
			http.Error(w, "user persist failed", http.StatusInternalServerError)
			return
		}
		userID = user.ID
		role = user.Role
	}
	s.setSession(w, userID, payload.AthleteID, role, payload.Name)
	http.SetCookie(w, &http.Cookie{Name: "cup_oauth_state", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, s.cfg.Server.PublicURL+"/", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "cup_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) roleForAthleteCode(code int64) string {
	for _, adminCode := range s.cfg.Auth.AdminAthleteCodes {
		if adminCode == code {
			return "admin"
		}
	}
	return "captain"
}

func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AthleteID   int64  `json:"athlete_id"`
		AthleteCode int64  `json:"athlete_code"`
		Name        string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if body.AthleteCode == 0 {
		body.AthleteCode = body.AthleteID
	}
	role := s.roleForAthleteCode(body.AthleteCode)
	var userID int64
	if s.store != nil {
		user, err := s.store.EnsureUser(r.Context(), repository.User{
			S95AthleteID: body.AthleteID, AthleteCode: body.AthleteCode, Name: body.Name, Role: role,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		userID = user.ID
		role = user.Role
	}
	s.setSession(w, userID, body.AthleteID, role, body.Name)
	writeJSON(w, map[string]any{"status": "ok", "user_id": userID})
}

func (s *Server) setSession(w http.ResponseWriter, userID, athleteID int64, role, name string) {
	jwtTok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID, "sub": athleteID, "role": role, "name": name,
		"exp": time.Now().Add(s.cfg.Auth.JWTTTL).Unix(),
	})
	signed, err := jwtTok.SignedString([]byte(s.cfg.Auth.JWTSecret))
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "cup_session", Value: signed, Path: "/", HttpOnly: true,
		Secure: s.cfg.Server.CookieSecure, SameSite: http.SameSiteLaxMode,
		MaxAge: int(s.cfg.Auth.JWTTTL.Seconds()),
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func Run(ctx context.Context, cfg *config.Config) error {
	pool, err := repository.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	store := repository.NewStore(pool)
	var api s95client.API
	if cfg.S95.Mock {
		api = s95client.NewMock()
	} else {
		api = s95client.New(cfg.S95.BaseURL, cfg.S95.APIKey, cfg.S95.RateLimitRPS)
	}
	worker := service.NewSyncWorker(cfg, store, api)
	go worker.Run(ctx, cfg.Sync.LocationsInterval, cfg.Sync.ProtocolPollInterval)

	srv := &http.Server{Addr: cfg.Server.Addr, Handler: NewServer(cfg, Deps{Store: store, S95: api, JTI: repository.NewSSOTokens(pool)}).Router()}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	return srv.ListenAndServe()
}
