package data

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/larntz/status/internal/checks"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateDevChecks will create a few checks we can use during development
func CreateDevChecks(client *mongo.Client, filename string) {
	// create some checks to use during development
	statusChecksColl := client.Database("status").Collection("status_checks")

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
	}
	reader := csv.NewReader(file)
	domains, _ := reader.ReadAll()

	statusChecks := checks.Checks{}

	for i, domain := range domains {
		statusChecks.StatusChecks = append(statusChecks.StatusChecks, checks.StatusCheck{
			ID:       fmt.Sprintf("devCheckList%d", i),
			URL:      "https://" + domain[1],
			Interval: 10,
			Regions:  []string{"us-dev-1"},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
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
