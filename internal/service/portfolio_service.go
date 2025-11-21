package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/models"
	"github.com/stocky/backend/internal/price"
	"github.com/stocky/backend/internal/repository"
)

type PortfolioService struct {
	repo     *repository.Repository
	priceSvc *price.Service
}

func NewPortfolioService(repo *repository.Repository, priceSvc *price.Service) *PortfolioService {
	return &PortfolioService{repo: repo, priceSvc: priceSvc}
}

func (s *PortfolioService) GetPortfolio(ctx context.Context, userID uuid.UUID) ([]models.PortfolioPosition, error) {
	positions, err := s.repo.ListUserPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(positions) == 0 {
		return []models.PortfolioPosition{}, nil
	}

	symbols := make([]string, 0, len(positions))
	for _, pos := range positions {
		symbols = append(symbols, pos.Symbol)
	}
	quotes, err := s.priceSvc.QuotesFor(ctx, symbols)
	if err != nil {
		return nil, err
	}

	result := make([]models.PortfolioPosition, 0, len(positions))
	for _, pos := range positions {
		quote, ok := quotes[pos.Symbol]
		if !ok {
			continue
		}
		currentValue := pos.Shares.Mul(quote.Price).Round(2)
		avgCost := pos.AvgCost.Round(4)
		unrealized := currentValue.Sub(pos.Shares.Mul(avgCost)).Round(2)
		result = append(result, models.PortfolioPosition{
			Symbol:            pos.Symbol,
			Shares:            pos.Shares,
			AverageCost:       avgCost,
			CurrentPrice:      quote.Price,
			CurrentValue:      currentValue,
			UnrealizedPnl:     unrealized,
			LastPriceSnapshot: quote.FetchedAt,
		})
	}
	return result, nil
}
