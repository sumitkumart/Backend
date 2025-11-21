package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/stocky/backend/internal/config"
	"github.com/stocky/backend/internal/models"
	"github.com/stocky/backend/internal/price"
	"github.com/stocky/backend/internal/repository"
)

type RewardInput struct {
	UserID     uuid.UUID
	Symbol     string
	Shares     decimal.Decimal
	EventID    string
	RewardedAt time.Time
}

type RewardService struct {
	repo     *repository.Repository
	priceSvc *price.Service
	fc       config.FeeConfig
}

func NewRewardService(repo *repository.Repository, priceSvc *price.Service, fc config.FeeConfig) *RewardService {
	return &RewardService{repo: repo, priceSvc: priceSvc, fc: fc}
}

func (s *RewardService) RewardUser(ctx context.Context, input RewardInput) (*models.RewardEvent, error) {
	if input.UserID == uuid.Nil {
		return nil, errors.New("user id is required")
	}
	if input.Symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if input.EventID == "" {
		return nil, errors.New("event id is required")
	}
	if !input.Shares.GreaterThan(decimal.Zero) {
		return nil, errors.New("shares must be positive")
	}
	if input.RewardedAt.IsZero() {
		input.RewardedAt = time.Now().UTC()
	}

	symbol := strings.ToUpper(input.Symbol)
	quote, err := s.priceSvc.EnsureQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	cost := quote.Price.Mul(input.Shares).Round(4)
	brokerage := applyBps(cost, s.fc.BrokerageBps)
	taxes := applyBps(cost, s.fc.TaxBps)
	total := cost.Add(brokerage).Add(taxes)

	params := repository.RewardCreationParams{
		UserID:     input.UserID,
		Symbol:     symbol,
		Shares:     input.Shares,
		GrantPrice: quote.Price,
		Brokerage:  brokerage,
		Taxes:      taxes,
		Total:      total,
		EventKey:   input.EventID,
		RewardedAt: input.RewardedAt,
	}

	reward, err := s.repo.CreateReward(ctx, params)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateReward) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return reward, nil
}

func applyBps(amount decimal.Decimal, bps int) decimal.Decimal {
	if bps == 0 {
		return decimal.Zero
	}
	return amount.Mul(decimal.NewFromInt(int64(bps))).Div(decimal.NewFromInt(10000)).Round(4)
}
