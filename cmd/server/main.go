package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/stocky/backend/internal/config"
	"github.com/stocky/backend/internal/db"
	apihttp "github.com/stocky/backend/internal/http"
	"github.com/stocky/backend/internal/jobs"
	"github.com/stocky/backend/internal/price"
	"github.com/stocky/backend/internal/repository"
	"github.com/stocky/backend/internal/server"
	"github.com/stocky/backend/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer client.Disconnect(ctx)

	store := repository.New(client)

	priceFetcher := price.NewRandomFetcher(cfg.Price.RandomFloorPrice, cfg.Price.RandomCeilPrice)
	priceSvc := price.NewService(store, priceFetcher)
	rewardSvc := service.NewRewardService(store, priceSvc, cfg.Fees)
	statsSvc := service.NewStatsService(store, priceSvc)
	portfolioSvc := service.NewPortfolioService(store, priceSvc)

	handler := apihttp.NewHandler(rewardSvc, statsSvc, portfolioSvc)
	httpServer := server.New(cfg.HTTPPort, handler.Router())

	priceJob := jobs.NewPriceSyncJob(cfg.Price.JobInterval, priceSvc, store)
	go priceJob.Start(ctx)

	go func() {
		if err := httpServer.Start(); err != nil {
			log.Printf("http server stopped: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Stop(shutdownCtx); err != nil {
		log.Printf("http server shutdown: %v", err)
	}
}
