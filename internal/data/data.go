// Package data abstracts access to our database, e.g., mongo
package data

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"github.com/larntz/status/internal/checks"
)

// Connect verifies we can connect to the database
func Connect(ctx context.Context, log *zap.Logger) (*mongo.Client, error) {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		log.Fatal("DB_CONNECTION_STRING must be set. Exiting.")
	}

	options := options.Client()
	options.ApplyURI(connString)
	maxPoolSize := uint64(500)
	options.MaxPoolSize = &maxPoolSize
	minPoolSize := uint64(50)
	options.MinPoolSize = &minPoolSize

	dbClient, err := mongo.NewClient(options)
	if err != nil {
		log.Error("NewClient() error", zap.String("err", err.Error()))
		return nil, err
	}
	err = dbClient.Connect(ctx)
	if err != nil {
		log.Error("Connect() error", zap.String("err", err.Error()))
		return nil, err
	}

	// Ping the primary
	if err := dbClient.Ping(ctx, readpref.Primary()); err != nil {
		log.Error("Can't ping the primary", zap.String("err", err.Error()))
		return nil, err
	}

	return dbClient, nil
}

// GetChecks returns all checks assigned to a region
func GetChecks(client *mongo.Client, region string, log *zap.Logger) checks.Checks {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	filter := bson.D{{Key: "regions", Value: region}}
	statusChecksColl := client.Database("status").Collection("status_checks")
	cursor, err := statusChecksColl.Find(ctx, filter)
	if err != nil {
		log.Error("Find() error", zap.String("err", err.Error()))
	}

	var statusChecks checks.Checks
	if err = cursor.All(ctx, &statusChecks.StatusChecks); err != nil {
		log.Error("cursor.All() error", zap.String("err", err.Error()))
	}

	return statusChecks
}
