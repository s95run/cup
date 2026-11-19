package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrJTIReplay = errors.New("jti already used")

type SSOTokens struct {
	pool *pgxpool.Pool
}

func NewSSOTokens(pool *pgxpool.Pool) *SSOTokens {
	return &SSOTokens{pool: pool}
}

func (s *SSOTokens) ConsumeJTI(ctx context.Context, jti string, expiresAt time.Time) error {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO used_sso_tokens (jti, expires_at) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		jti, expiresAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrJTIReplay
	}
	return nil
}

func (s *SSOTokens) PurgeExpired(ctx context.Context, now time.Time) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM used_sso_tokens WHERE expires_at < $1`, now)
	return err
}

// InMemorySSOTokens for tests and dev without DB.
type InMemorySSOTokens struct {
	seen map[string]struct{}
}

func NewInMemorySSOTokens() *InMemorySSOTokens {
	return &InMemorySSOTokens{seen: make(map[string]struct{})}
}

func (m *InMemorySSOTokens) ConsumeJTI(_ context.Context, jti string, _ time.Time) error {
	if _, ok := m.seen[jti]; ok {
		return ErrJTIReplay
	}
	m.seen[jti] = struct{}{}
	return nil
}

func (m *InMemorySSOTokens) PurgeExpired(context.Context, time.Time) error { return nil }

type JTIChecker interface {
	ConsumeJTI(ctx context.Context, jti string, expiresAt time.Time) error
}

var _ JTIChecker = (*SSOTokens)(nil)
var _ JTIChecker = (*InMemorySSOTokens)(nil)
