package price

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/models"
	"github.com/stocky/backend/internal/repository"
)

type Service struct {
	repo    *repository.Repository
	fetcher Fetcher
}

func NewService(repo *repository.Repository, fetcher Fetcher) *Service {
	return &Service{
		repo:    repo,
		fetcher: fetcher,
	}
}

func (s *Service) EnsureQuote(ctx context.Context, symbol string) (*models.PriceQuote, error) {
	quote, err := s.repo.LatestQuote(ctx, symbol)
	if err == nil {
		return quote, nil
	}
	if !errors.Is(err, repository.ErrQuoteNotFound) {
		return nil, err
	}
	return s.fetchAndPersist(ctx, symbol)
}

func (s *Service) RefreshAll(ctx context.Context) ([]models.PriceQuote, error) {
	symbols, err := s.repo.ListTrackedSymbols(ctx)
	if err != nil {
		return nil, err
	}
	var quotes []models.PriceQuote
	for _, symbol := range symbols {
		quote, err := s.fetchAndPersist(ctx, symbol)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, *quote)
	}
	return quotes, nil
}

func (s *Service) QuotesFor(ctx context.Context, symbols []string) (map[string]models.PriceQuote, error) {
	result, err := s.repo.QuotesForSymbols(ctx, symbols)
	if err != nil {
		return nil, err
	}
	for _, symbol := range symbols {
		if _, ok := result[symbol]; ok {
			continue
		}
		quote, err := s.fetchAndPersist(ctx, symbol)
		if err != nil {
			return nil, err
		}
		result[symbol] = *quote
	}
	return result, nil
}

func (s *Service) fetchAndPersist(ctx context.Context, symbol string) (*models.PriceQuote, error) {
	if s.fetcher == nil {
		return nil, errors.New("no price fetcher configured")
	}
	price, err := s.fetcher.Fetch(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return s.saveQuote(ctx, symbol, price)
}

func (s *Service) saveQuote(ctx context.Context, symbol string, price decimal.Decimal) (*models.PriceQuote, error) {
	quote := models.PriceQuote{
		Symbol:    symbol,
		Price:     price,
		Source:    "mock-random",
		FetchedAt: time.Now().UTC(),
	}
	if err := s.repo.UpsertQuote(ctx, symbol, price, quote.Source, quote.FetchedAt); err != nil {
		return nil, err
	}
	return &quote, nil
}
