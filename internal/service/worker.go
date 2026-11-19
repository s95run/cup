package service

import (
	"context"
	"log"
	"time"

	"github.com/s95run/cup/internal/config"
	"github.com/s95run/cup/internal/repository"
	"github.com/s95run/cup/internal/s95client"
)

// SyncWorker keeps locations, registration windows and stage protocols up to date.
type SyncWorker struct {
	cup *Cup
}

func NewSyncWorker(cfg *config.Config, store *repository.Store, api s95client.API) *SyncWorker {
	return &SyncWorker{cup: &Cup{Store: store, S95: api, Deadline: cfg.Registration.DeadlineBeforeStage}}
}

func (w *SyncWorker) Run(ctx context.Context, locationsEvery, protocolsEvery time.Duration) {
	w.tick(ctx, true, true)
	locations := time.NewTicker(locationsEvery)
	protocols := time.NewTicker(protocolsEvery)
	defer locations.Stop()
	defer protocols.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-locations.C:
			w.tick(ctx, true, false)
		case <-protocols.C:
			w.tick(ctx, false, true)
		}
	}
}

func (w *SyncWorker) tick(ctx context.Context, locations, protocols bool) {
	if locations {
		if err := w.cup.SyncLocations(ctx); err != nil {
			log.Printf("sync locations: %v", err)
		}
	}
	if _, err := w.cup.CloseRegistrations(ctx); err != nil {
		log.Printf("sync registration windows: %v", err)
	}
	if protocols {
		if err := w.cup.SyncDueProtocols(ctx); err != nil {
			log.Printf("sync protocols: %v", err)
		}
	}
}
