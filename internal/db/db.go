package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewPool creates a MongoDB client connection.
func NewPool(ctx context.Context, connURL string) (*mongo.Client, error) {
	opts := options.Client().
		ApplyURI(connURL).
		SetMaxPoolSize(8).
		SetMinPoolSize(1)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// GetDatabase returns the default database ("stocky") from MongoDB client.
func GetDatabase(client *mongo.Client) *mongo.Database {
	return client.Database("stocky")
}
