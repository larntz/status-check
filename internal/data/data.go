// Package data abstracts access to our database, e.g., mongo
package data

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/checks"
)

// Connect verifies we can connect to the database
func Connect(ctx context.Context, app *application.State) error {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		log.Fatal("DB_CONNECTION_STRING must be set. Exiting.")
	}

	app.DBClientMutex.Lock()
	defer app.DBClientMutex.Unlock()

	var err error
	app.DbClient, err = mongo.NewClient(options.Client().ApplyURI(connString))
	if err != nil {
		log.Errorf("NewClient() error: %s", err)
		return err
	}
	err = app.DbClient.Connect(ctx)
	if err != nil {
		log.Errorf("Connect() error: %s", err)
		return err
	}

	// Ping the primary
	if err := app.DbClient.Ping(ctx, readpref.Primary()); err != nil {
		log.Errorf("Can't ping the primary: %s", err)
		return err
	}

	return nil
}

// GetChecks returns all checks assigned to a region
func GetChecks(client *mongo.Client, region string) checks.Checks {
	filter := bson.D{{Key: "regions", Value: region}}
	statusChecksColl := client.Database("status").Collection("status_checks")
	cursor, err := statusChecksColl.Find(context.TODO(), filter)
	if err != nil {
		log.Errorf("Find() error: %s", err)
	}

	var statusChecks checks.Checks
	if err = cursor.All(context.TODO(), &statusChecks.StatusChecks); err != nil {
		log.Errorf("cursor.All() error: %s", err)
	}

	return statusChecks
}
