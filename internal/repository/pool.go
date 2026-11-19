package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/s95run/cup/internal/config"
)

func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO "+pgx.Identifier{cfg.Schema}.Sanitize())
		return err
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

func RunMigrations(ctx context.Context, cfg config.DBConfig, migrationsDir string) error {
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	// goose via database/sql bridge
	return migrateGoose(cfg.DSN(), migrationsDir)
}
