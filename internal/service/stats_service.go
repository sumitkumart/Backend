package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/models"
	"github.com/stocky/backend/internal/price"
	"github.com/stocky/backend/internal/repository"
)

type StatsService struct {
	repo     *repository.Repository
	priceSvc *price.Service
}

func NewStatsService(repo *repository.Repository, priceSvc *price.Service) *StatsService {
	return &StatsService{repo: repo, priceSvc: priceSvc}
}

func (s *StatsService) TodayRewards(ctx context.Context, userID uuid.UUID) ([]models.TodayReward, error) {
	now := time.Now().UTC()
	start := startOfDay(now)
	end := start.Add(24 * time.Hour)
	return s.repo.ListTodayRewards(ctx, userID, start, end)
}

func (s *StatsService) HistoricalINR(ctx context.Context, userID uuid.UUID) ([]models.DailyINR, error) {
	today := startOfDay(time.Now().UTC())
	return s.repo.HistoricalHoldings(ctx, userID, today)
}

func (s *StatsService) UserStats(ctx context.Context, userID uuid.UUID) (*models.StatsSummary, error) {
	now := time.Now().UTC()
	start := startOfDay(now)
	end := start.Add(24 * time.Hour)

	totals, err := s.repo.AggregateShares(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	positions, err := s.repo.ListUserPositions(ctx, userID)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, 0, len(positions))
	for _, pos := range positions {
		symbols = append(symbols, pos.Symbol)
	}

	var (
		portfolioValue decimal.Decimal
		priceAsOf      time.Time
	)
	if len(symbols) > 0 {
		quotes, err := s.priceSvc.QuotesFor(ctx, symbols)
		if err != nil {
			return nil, err
		}
		for _, pos := range positions {
			quote, ok := quotes[pos.Symbol]
			if !ok {
				continue
			}
			value := quote.Price.Mul(pos.Shares)
			portfolioValue = portfolioValue.Add(value)
			if quote.FetchedAt.After(priceAsOf) {
				priceAsOf = quote.FetchedAt
			}
		}
	}

	return &models.StatsSummary{
		TotalsToday:    totals,
		PortfolioValue: portfolioValue.Round(2),
		PriceAsOf:      priceAsOf,
	}, nil
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
