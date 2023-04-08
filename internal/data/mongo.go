// Package data abstracts access to our database, e.g., mongo
package data

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/larntz/status/internal/checks"
)

// MongoDB struct implements the Database interface
type MongoDB struct {
	Client *mongo.Client
}

// Connect to mongo server
func (db *MongoDB) Connect() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db.Client, err = connect(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Connect verifies we can connect to the database
func connect(ctx context.Context) (*mongo.Client, error) {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		return nil, errors.New("no environment variable DB_CONNECTION_STRING")
	}

	options := options.Client()
	options.ApplyURI(connString)
	maxPoolSize := uint64(500)
	options.MaxPoolSize = &maxPoolSize
	minPoolSize := uint64(50)
	options.MinPoolSize = &minPoolSize

	dbClient, err := mongo.NewClient(options)
	if err != nil {
		return nil, err
	}
	err = dbClient.Connect(ctx)
	if err != nil {
		return nil, err
	}

	// Ping the primary
	if err := dbClient.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return dbClient, nil
}

// GetRegionChecks returns all checks assigned to a region
func (db MongoDB) GetRegionChecks(region string) (checks.Checks, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	filter := bson.D{{Key: "regions", Value: region}}
	statusChecksColl := db.Client.Database("status").Collection("status_checks")
	cursor, err := statusChecksColl.Find(ctx, filter)
	if err != nil {
		return checks.Checks{}, err
	}

	var statusChecks checks.Checks
	if err = cursor.All(ctx, &statusChecks.StatusChecks); err != nil {
		return checks.Checks{}, err
	}

	return statusChecks, nil
}

// SendResults to Mongo
func (db MongoDB) SendResults(results []interface{}) (string, error) {
	coll := db.Client.Database("status").Collection("check_results")
	result, err := coll.InsertMany(context.TODO(), results)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf("successfully inserted %d items", len(result.InsertedIDs))
	return summary, nil
}

// Disconnect Mongo
func (db MongoDB) Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db.Client.Disconnect(ctx)
}
