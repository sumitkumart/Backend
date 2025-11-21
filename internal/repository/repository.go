package repository

import "go.mongodb.org/mongo-driver/mongo"

type Repository struct {
	client *mongo.Client
	db     *mongo.Database
}

func New(client *mongo.Client) *Repository {
	db := client.Database("stocky")
	return &Repository{
		client: client,
		db:     db,
	}
}

func (r *Repository) Client() *mongo.Client {
	return r.client
}

func (r *Repository) DB() *mongo.Database {
	return r.db
}
