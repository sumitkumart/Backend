package repository

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stocky/backend/internal/models"
)

var ErrQuoteNotFound = errors.New("quote not found")

func (r *Repository) ListTrackedSymbols(ctx context.Context) ([]string, error) {
	collection := r.db.Collection("stocks")
	opts := options.Find().SetProjection(bson.M{"symbol": 1})
	cursor, err := collection.Find(ctx, bson.M{"status": "ACTIVE"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var symbols []string
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if symbol, ok := doc["symbol"].(string); ok {
			symbols = append(symbols, symbol)
		}
	}
	return symbols, nil
}

func (r *Repository) UpsertQuote(ctx context.Context, symbol string, price decimal.Decimal, source string, fetchedAt time.Time) error {
	collection := r.db.Collection("price_quotes")

	// Upsert price quote
	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"symbol": symbol},
		bson.M{"$set": bson.M{
			"symbol":     symbol,
			"price_inr":  price.String(),
			"source":     source,
			"fetched_at": fetchedAt,
		}},
		opts,
	)
	if err != nil {
		return err
	}

	// Insert price history
	historyCollection := r.db.Collection("price_history")
	_, err = historyCollection.InsertOne(ctx, bson.M{
		"symbol":     symbol,
		"price_inr":  price.String(),
		"as_of":      fetchedAt,
		"source":     source,
		"created_at": time.Now(),
	})
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return err
	}

	return nil
}

func (r *Repository) LatestQuote(ctx context.Context, symbol string) (*models.PriceQuote, error) {
	collection := r.db.Collection("price_quotes")
	var result bson.M
	err := collection.FindOne(ctx, bson.M{"symbol": symbol}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrQuoteNotFound
		}
		return nil, err
	}

	price, _ := decimal.NewFromString(result["price_inr"].(string))
	return &models.PriceQuote{
		Symbol:    result["symbol"].(string),
		Price:     price,
		Source:    result["source"].(string),
		FetchedAt: result["fetched_at"].(time.Time),
	}, nil
}

func (r *Repository) QuotesForSymbols(ctx context.Context, symbols []string) (map[string]models.PriceQuote, error) {
	if len(symbols) == 0 {
		return map[string]models.PriceQuote{}, nil
	}

	collection := r.db.Collection("price_quotes")
	cursor, err := collection.Find(ctx, bson.M{"symbol": bson.M{"$in": symbols}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]models.PriceQuote)
	var docs []bson.M
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		symbol := doc["symbol"].(string)
		price, _ := decimal.NewFromString(doc["price_inr"].(string))
		result[symbol] = models.PriceQuote{
			Symbol:    symbol,
			Price:     price,
			Source:    doc["source"].(string),
			FetchedAt: doc["fetched_at"].(time.Time),
		}
	}
	return result, nil
}
