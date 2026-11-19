package httptransport

import (
	"testing"

	"github.com/s95run/cup/internal/config"
)

func TestRoleForAthleteID(t *testing.T) {
	cfg := &config.Config{Auth: config.AuthConfig{AdminAthleteIDs: []int64{42}}}
	srv := NewServer(cfg, Deps{})
	if got := srv.roleForAthleteID(42); got != "admin" {
		t.Fatalf("admin id => %q", got)
	}
	if got := srv.roleForAthleteID(99); got != "captain" {
		t.Fatalf("captain id => %q", got)
	}
}