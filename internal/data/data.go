// Package data abstracts access to our database, e.g., mongo
package data

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/checks"
)

// Connect verifies we can connect to the database
func Connect(ctx context.Context, app *application.State) error {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		app.Log.Fatal("DB_CONNECTION_STRING must be set. Exiting.")
	}

	var err error
	app.DbClient, err = mongo.NewClient(options.Client().ApplyURI(connString))
	if err != nil {
		app.Log.Error("NewClient() error", zap.String("err", err.Error()))
		return err
	}
	err = app.DbClient.Connect(ctx)
	if err != nil {
		app.Log.Error("Connect() error", zap.String("err", err.Error()))
		return err
	}

	// Ping the primary
	if err := app.DbClient.Ping(ctx, readpref.Primary()); err != nil {
		app.Log.Error("Can't ping the primary", zap.String("err", err.Error()))
		return err
	}

	return nil
}

// GetChecks returns all checks assigned to a region
func GetChecks(client *mongo.Client, region string, log *zap.Logger) checks.Checks {
	filter := bson.D{{Key: "regions", Value: region}}
	statusChecksColl := client.Database("status").Collection("status_checks")
	cursor, err := statusChecksColl.Find(context.TODO(), filter)
	if err != nil {
		log.Error("Find() error", zap.String("err", err.Error()))
	}

	var statusChecks checks.Checks
	if err = cursor.All(context.TODO(), &statusChecks.StatusChecks); err != nil {
		log.Error("cursor.All() error", zap.String("err", err.Error()))
	}

	return statusChecks
}
