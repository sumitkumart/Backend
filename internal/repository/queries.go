package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stocky/backend/internal/models"
)

type UserPosition struct {
	Symbol  string
	Shares  decimal.Decimal
	AvgCost decimal.Decimal
}

type RawPosition struct {
	UserID uuid.UUID
	Symbol string
	Shares decimal.Decimal
}

func (r *Repository) ListTodayRewards(ctx context.Context, userID uuid.UUID, dayStart, dayEnd time.Time) ([]models.TodayReward, error) {
	collection := r.db.Collection("reward_events")
	opts := options.Find().SetSort(bson.M{"rewarded_at": 1})
	cursor, err := collection.Find(ctx, bson.M{
		"user_id":     userID.String(),
		"rewarded_at": bson.M{"$gte": dayStart, "$lt": dayEnd},
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.TodayReward
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		shares, _ := decimal.NewFromString(doc["shares"].(string))
		items = append(items, models.TodayReward{
			Symbol:     doc["symbol"].(string),
			Shares:     shares,
			RewardedAt: doc["rewarded_at"].(time.Time),
		})
	}
	return items, nil
}

func (r *Repository) AggregateShares(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]models.TodayTotals, error) {
	collection := r.db.Collection("reward_events")
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"user_id":     userID.String(),
				"rewarded_at": bson.M{"$gte": start, "$lt": end},
			},
		},
		{
			"$group": bson.M{
				"_id":    "$symbol",
				"total":  bson.M{"$sum": "$shares"},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.TodayTotals
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		totalStr := doc["total"].(string)
		if shares, ok := doc["total"].(float64); ok {
			totalStr = decimal.NewFromFloat(shares).String()
		}
		total, _ := decimal.NewFromString(totalStr)
		items = append(items, models.TodayTotals{
			Symbol: doc["_id"].(string),
			Shares: total,
		})
	}
	return items, nil
}

func (r *Repository) HistoricalHoldings(ctx context.Context, userID uuid.UUID, before time.Time) ([]models.DailyINR, error) {
	collection := r.db.Collection("daily_holdings")
	opts := options.Find().SetSort(bson.M{"date": 1})
	cursor, err := collection.Find(ctx, bson.M{
		"user_id": userID.String(),
		"date":    bson.M{"$lt": before},
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.DailyINR
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		value, _ := decimal.NewFromString(doc["total_value_inr"].(string))
		items = append(items, models.DailyINR{
			Date:         doc["date"].(time.Time),
			TotalValueIn: value,
		})
	}
	return items, nil
}

func (r *Repository) ListUserPositions(ctx context.Context, userID uuid.UUID) ([]UserPosition, error) {
	collection := r.db.Collection("user_positions")
	cursor, err := collection.Find(ctx, bson.M{"user_id": userID.String()})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []UserPosition
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		shares, _ := decimal.NewFromString(doc["net_shares"].(string))
		avg, _ := decimal.NewFromString(doc["avg_cost_inr"].(string))
		items = append(items, UserPosition{
			Symbol:  doc["symbol"].(string),
			Shares:  shares,
			AvgCost: avg,
		})
	}
	return items, nil
}

func (r *Repository) UpsertDailyHolding(ctx context.Context, userID uuid.UUID, date time.Time, value decimal.Decimal) error {
	collection := r.db.Collection("daily_holdings")
	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"user_id": userID.String(), "date": date},
		bson.M{"$set": bson.M{
			"user_id":          userID.String(),
			"date":             date,
			"total_value_inr":  value.String(),
			"updated_at":       time.Now(),
		}},
		opts,
	)
	return err
}

func (r *Repository) ListAllPositions(ctx context.Context) ([]RawPosition, error) {
	collection := r.db.Collection("user_positions")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []RawPosition
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		userID, _ := uuid.Parse(doc["user_id"].(string))
		shares, _ := decimal.NewFromString(doc["net_shares"].(string))
		items = append(items, RawPosition{
			UserID: userID,
			Symbol: doc["symbol"].(string),
			Shares: shares,
		})
	}
	return items, nil
}
