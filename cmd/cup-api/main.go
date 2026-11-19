package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/s95run/cup/internal/config"
	"github.com/s95run/cup/internal/repository"
	httptransport "github.com/s95run/cup/internal/transport/http"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run DB migrations and exit")
	flag.Parse()

	cfgPath := os.Getenv("CUP_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.Getenv("CUP_ALLOW_EXAMPLE_CONFIG") == "true" {
			cfgPath = "configs/config.example.yaml"
			cfg, err = config.Load(cfgPath)
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	ctx := context.Background()
	if *migrateOnly {
		if err := repository.RunMigrations(ctx, cfg.DB, "migrations"); err != nil {
			log.Fatal(err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := httptransport.Run(ctx, cfg); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
