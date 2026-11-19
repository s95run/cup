package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/s95run/cup/internal/config"
)

func TestHealthz(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Addr: ":0", PublicURL: "http://localhost:3000"},
		DB:     config.DBConfig{Schema: "cup"},
		S95:    config.S95Config{SSOSecret: "test-secret-min-32-chars-xxxxxxxx"},
		Auth:   config.AuthConfig{JWTSecret: "jwt-secret-min-32-chars-xxxxxxxxxx"},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	NewServer(cfg, Deps{}).Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestCSRFBlocksMutation(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{PublicURL: "http://localhost:3000"},
		S95:    config.S95Config{SSOSecret: "test-secret-min-32-chars-xxxxxxxx"},
		Auth:   config.AuthConfig{JWTSecret: "jwt-secret-min-32-chars-xxxxxxxxxx", DevAuthEnabled: true},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	NewServer(cfg, Deps{}).Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}
