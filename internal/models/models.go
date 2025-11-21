package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type RewardEvent struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"userId"`
	Symbol         string          `json:"symbol"`
	Shares         decimal.Decimal `json:"shares"`
	GrantedPrice   decimal.Decimal `json:"grantedPrice"`
	BrokerageInr   decimal.Decimal `json:"brokerageInr"`
	TaxesInr       decimal.Decimal `json:"taxesInr"`
	TotalCashOut   decimal.Decimal `json:"totalCashOutInr"`
	RewardedAt     time.Time       `json:"rewardedAt"`
	CreatedAt      time.Time       `json:"createdAt"`
	EventKey       string          `json:"eventKey"`
}

type TodayReward struct {
	Symbol     string          `json:"symbol"`
	Shares     decimal.Decimal `json:"shares"`
	RewardedAt time.Time       `json:"rewardedAt"`
}

type DailyINR struct {
	Date         time.Time       `json:"date"`
	TotalValueIn decimal.Decimal `json:"totalInr"`
}

type TodayTotals struct {
	Symbol string          `json:"symbol"`
	Shares decimal.Decimal `json:"shares"`
}

type StatsSummary struct {
	TotalsToday    []TodayTotals   `json:"totalsToday"`
	PortfolioValue decimal.Decimal `json:"portfolioInr"`
	PriceAsOf      time.Time       `json:"priceAsOf"`
}

type PortfolioPosition struct {
	Symbol            string          `json:"symbol"`
	Shares            decimal.Decimal `json:"shares"`
	AverageCost       decimal.Decimal `json:"avgAcqPriceInr"`
	CurrentPrice      decimal.Decimal `json:"currentPriceInr"`
	CurrentValue      decimal.Decimal `json:"currentValueInr"`
	UnrealizedPnl     decimal.Decimal `json:"unrealizedPnlInr"`
	LastPriceSnapshot time.Time       `json:"priceAsOf"`
}

type PriceQuote struct {
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Source    string          `json:"source"`
	FetchedAt time.Time       `json:"fetchedAt"`
}
