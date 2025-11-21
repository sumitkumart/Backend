package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stocky/backend/internal/models"
)

var (
	ErrDuplicateReward = errors.New("reward event already exists")
)

type RewardCreationParams struct {
	UserID     uuid.UUID
	Symbol     string
	Shares     decimal.Decimal
	GrantPrice decimal.Decimal
	Brokerage  decimal.Decimal
	Taxes      decimal.Decimal
	Total      decimal.Decimal
	EventKey   string
	RewardedAt time.Time
}

func (r *Repository) CreateReward(ctx context.Context, params RewardCreationParams) (*models.RewardEvent, error) {
	session, err := r.client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	var result *models.RewardEvent
	err = mongo.WithSession(ctx, session, func(sessionCtx mongo.SessionContext) error {
		if err := sessionCtx.StartTransaction(); err != nil {
			return err
		}

		// Ensure user exists
		usersCollection := r.db.Collection("users")
		_, err = usersCollection.UpdateOne(
			sessionCtx,
			bson.M{"_id": params.UserID.String()},
			bson.M{"$setOnInsert": bson.M{"_id": params.UserID.String(), "created_at": time.Now()}},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			sessionCtx.AbortTransaction(sessionCtx)
			return err
		}

		// Ensure stock exists
		stocksCollection := r.db.Collection("stocks")
		_, err = stocksCollection.UpdateOne(
			sessionCtx,
			bson.M{"symbol": strings.ToUpper(params.Symbol)},
			bson.M{"$setOnInsert": bson.M{
				"symbol":     strings.ToUpper(params.Symbol),
				"name":       strings.ToUpper(params.Symbol),
				"exchange":   "NSE",
				"status":     "ACTIVE",
				"created_at": time.Now(),
			}},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			sessionCtx.AbortTransaction(sessionCtx)
			return err
		}

		// Check for duplicate reward event
		rewardCollection := r.db.Collection("reward_events")
		count, err := rewardCollection.CountDocuments(sessionCtx, bson.M{"event_key": params.EventKey})
		if err != nil {
			sessionCtx.AbortTransaction(sessionCtx)
			return err
		}
		if count > 0 {
			sessionCtx.AbortTransaction(sessionCtx)
			return ErrDuplicateReward
		}

		// Insert reward event
		id := uuid.New()
		reward := bson.M{
			"_id":                 id.String(),
			"user_id":             params.UserID.String(),
			"symbol":              strings.ToUpper(params.Symbol),
			"shares":              params.Shares.String(),
			"granted_price_inr":   params.GrantPrice.String(),
			"brokerage_inr":       params.Brokerage.String(),
			"taxes_inr":           params.Taxes.String(),
			"total_cash_out_inr":  params.Total.String(),
			"rewarded_at":         params.RewardedAt,
			"event_key":           params.EventKey,
			"created_at":          time.Now(),
		}
		_, err = rewardCollection.InsertOne(sessionCtx, reward)
		if err != nil {
			sessionCtx.AbortTransaction(sessionCtx)
			return err
		}

		// Insert ledger entries
		ledgerCollection := r.db.Collection("ledger_entries")
		stockAccount := fmt.Sprintf("stock_inventory:%s", strings.ToUpper(params.Symbol))

		entries := []interface{}{
			bson.M{
				"event_id":        id.String(),
				"account_code":    stockAccount,
				"account_type":    "asset",
				"symbol":          strings.ToUpper(params.Symbol),
				"debit_inr":       params.GrantPrice.Mul(params.Shares).String(),
				"credit_inr":      "0",
				"stock_units":     params.Shares.String(),
				"memo":            "Rewarded stock inventory",
				"created_at":      time.Now(),
			},
			bson.M{
				"event_id":     id.String(),
				"account_code": "brokerage_expense",
				"account_type": "expense",
				"debit_inr":    params.Brokerage.String(),
				"credit_inr":   "0",
				"memo":         "Brokerage charges",
				"created_at":   time.Now(),
			},
			bson.M{
				"event_id":     id.String(),
				"account_code": "tax_expense",
				"account_type": "expense",
				"debit_inr":    params.Taxes.String(),
				"credit_inr":   "0",
				"memo":         "Statutory taxes",
				"created_at":   time.Now(),
			},
			bson.M{
				"event_id":     id.String(),
				"account_code": "cash",
				"account_type": "asset",
				"debit_inr":    "0",
				"credit_inr":   params.Total.String(),
				"memo":         "Cash outflow for reward",
				"created_at":   time.Now(),
			},
		}

		_, err = ledgerCollection.InsertMany(sessionCtx, entries)
		if err != nil {
			sessionCtx.AbortTransaction(sessionCtx)
			return err
		}

		// Update user position
		positionsCollection := r.db.Collection("user_positions")
		filter := bson.M{"user_id": params.UserID.String(), "symbol": strings.ToUpper(params.Symbol)}
		opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)

		var position bson.M
		err = positionsCollection.FindOneAndUpdate(
			sessionCtx,
			filter,
			bson.M{"$inc": bson.M{"net_shares": params.Shares.String()}},
			opts,
		).Decode(&position)
		if err != nil && err != mongo.ErrNoDocuments {
			sessionCtx.AbortTransaction(sessionCtx)
			return err
		}

		if err := sessionCtx.CommitTransaction(sessionCtx); err != nil {
			return err
		}

		// Parse result
		result = &models.RewardEvent{
			ID:            id,
			UserID:        params.UserID,
			Symbol:        strings.ToUpper(params.Symbol),
			Shares:        params.Shares,
			GrantedPrice:  params.GrantPrice,
			BrokerageInr:  params.Brokerage,
			TaxesInr:      params.Taxes,
			TotalCashOut:  params.Total,
			RewardedAt:    params.RewardedAt,
			EventKey:      params.EventKey,
			CreatedAt:     time.Now(),
		}
		return nil
	})

	return result, err
}
