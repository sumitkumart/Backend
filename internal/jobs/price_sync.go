package jobs

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/models"
	"github.com/stocky/backend/internal/price"
	"github.com/stocky/backend/internal/repository"
)

type PriceSyncJob struct {
	interval time.Duration
	priceSvc *price.Service
	repo     *repository.Repository
}

func NewPriceSyncJob(interval time.Duration, priceSvc *price.Service, repo *repository.Repository) *PriceSyncJob {
	return &PriceSyncJob{
		interval: interval,
		priceSvc: priceSvc,
		repo:     repo,
	}
}

func (j *PriceSyncJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.run(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.run(ctx)
		}
	}
}

func (j *PriceSyncJob) run(ctx context.Context) {
	quotes, err := j.priceSvc.RefreshAll(ctx)
	if err != nil {
		log.Printf("price sync: %v", err)
		return
	}
	quoteMap := make(map[string]models.PriceQuote, len(quotes))
	for _, quote := range quotes {
		quoteMap[quote.Symbol] = quote
	}

	if len(quoteMap) == 0 {
		return
	}

	positions, err := j.repo.ListAllPositions(ctx)
	if err != nil {
		log.Printf("valuation: %v", err)
		return
	}
	if len(positions) == 0 {
		return
	}

	userTotals := make(map[uuid.UUID]decimal.Decimal)
	for _, pos := range positions {
		quote, ok := quoteMap[pos.Symbol]
		if !ok {
			continue
		}
		value := pos.Shares.Mul(quote.Price).Round(2)
		if value.Equal(decimal.Zero) {
			continue
		}
		current := userTotals[pos.UserID]
		userTotals[pos.UserID] = current.Add(value)
	}

	if len(userTotals) == 0 {
		return
	}

	date := startOfDay(time.Now().UTC())
	for userID, total := range userTotals {
		if err := j.repo.UpsertDailyHolding(ctx, userID, date, total); err != nil {
			log.Printf("holdings upsert for user %s: %v", userID, err)
		}
	}
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
