package httptransport

import (
	"testing"

	"github.com/s95run/cup/internal/config"
)

func TestRoleForAthleteCode(t *testing.T) {
	cfg := &config.Config{Auth: config.AuthConfig{AdminAthleteCodes: []int64{770000001}}}
	srv := NewServer(cfg, Deps{})
	if got := srv.roleForAthleteCode(770000001); got != "admin" {
		t.Fatalf("admin code => %q", got)
	}
	if got := srv.roleForAthleteCode(770000099); got != "captain" {
		t.Fatalf("captain code => %q", got)
	}
}