// Package data abstracts access to our database, e.g., mongo
package data

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/larntz/status/internal/checks"
)

// TODO error handling: don't log.Fatal here. Just return false and the err.
// Need to configure error handling first.

// Ping verifies we can connect to the database
func Ping() bool {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		log.Fatal("DB_CONNECTION_STRING must be set. Exiting.")
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(connString))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	defer client.Disconnect(ctx)

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("Can't ping the primary", err)
	}

	return true
}

// GetChecks returns all checks assigned to a region
func GetChecks(region string) checks.Checks {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		log.Fatal("DB_CONNECTION_STRING must be set. Exiting.")
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(connString))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	defer client.Disconnect(ctx)

	filter := bson.D{{Key: "regions", Value: region}}
	statusChecksColl := client.Database("status").Collection("status_checks")
	cursor, err := statusChecksColl.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("Find() error: %s", err)
	}

	var statusChecks checks.Checks
	if err = cursor.All(context.TODO(), &statusChecks.StatusChecks); err != nil {
		log.Printf("cursor.All() error: %s", err)
	}

	return statusChecks
}

// CreateDevChecks will create a few checks we can use during development
func CreateDevChecks() {
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		log.Fatal("DB_CONNECTION_STRING must be set. Exiting.")
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(connString))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	defer client.Disconnect(ctx)

	// create some checks to use during development
	statusChecksColl := client.Database("status").Collection("status_checks")

	statusChecks := checks.Checks{}
	statusChecks.StatusChecks = append(statusChecks.StatusChecks, checks.StatusCheck{
		ID:       "devCheck1",
		URL:      "https://blue42.net/does-not-exist",
		Interval: 10,
		Regions:  []string{"us-east-1", "us-west-1"},
	})
	statusChecks.StatusChecks = append(statusChecks.StatusChecks, checks.StatusCheck{
		ID:       "devCheck2",
		URL:      "https://blue42.net/",
		Interval: 10,
		Regions:  []string{"us-east-1"},
	})
	statusChecks.StatusChecks = append(statusChecks.StatusChecks, checks.StatusCheck{
		ID:       "devCheck3",
		URL:      "http://lek.net/",
		Interval: 10,
		Regions:  []string{"us-east-1"},
	})
	statusChecks.StatusChecks = append(statusChecks.StatusChecks, checks.StatusCheck{
		ID:       "devCheck4",
		URL:      "http://slashdot.org",
		Interval: 10,
		Regions:  []string{"us-east-1"},
	})
	statusChecks.StatusChecks = append(statusChecks.StatusChecks, checks.StatusCheck{
		ID:       "devCheck5",
		URL:      "http://gitea.chacarntz.net",
		Interval: 10,
	})

	opts := options.FindOneAndReplace().SetUpsert(true)
	for _, check := range statusChecks.StatusChecks {
		filter := bson.D{{Key: "id", Value: check.ID}}
		var result bson.M
		err := statusChecksColl.FindOneAndReplace(ctx, filter, check, opts).Decode(&result)
		if err != nil {
			// ErrNoDocuments means that the filter did not match any documents in
			// the collection.
			if err == mongo.ErrNoDocuments {
				log.Printf("%s check not found", check.ID)
			} else {
				log.Fatalf("UpdateOne() failed: %s\nCheck: %+v", err, check)
			}
		}
		log.Printf("FindOneAndReplace: %s", result)
	}
}
