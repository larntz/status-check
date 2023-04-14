package data

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/larntz/status/internal/checks"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// CreateDevChecks will create a few checks we can use during development
func CreateDevChecks(filename string, log *zap.Logger) {
	// create some checks to use during development
	connString, ok := os.LookupEnv("DB_CONNECTION_STRING")
	if !ok {
		log.Fatal("no environment variable DB_CONNECTION_STRING")
	}

	options := options.Client()
	options.ApplyURI(connString)
	maxPoolSize := uint64(500)
	options.MaxPoolSize = &maxPoolSize
	minPoolSize := uint64(50)
	options.MinPoolSize = &minPoolSize

	client, err := mongo.NewClient(options)
	if err != nil {
		log.Fatal("Unable to create mongo client.", zap.String("error", err.Error()))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal("Unable to connect to mongo.", zap.String("error", err.Error()))
	}
	defer client.Disconnect(context.Background())

	// Ping the server
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("Ping mongo failed.", zap.String("error", err.Error()))
	}
	log.Info("Dropping collections")
	client.Database("status").Collection("status_checks").Drop(context.TODO())
	client.Database("status").Collection("check_results").Drop(context.TODO())

	statusChecksColl := client.Database("status").Collection("status_checks")

	file, err := os.Open(filename)
	if err != nil {
		log.Error("failed to open file", zap.String("err", err.Error()))
	}
	reader := csv.NewReader(file)
	domains, _ := reader.ReadAll()

	var statusChecks []interface{}
	interval := []int{15, 30, 60, 120}

	for i, domain := range domains {
		randI := rand.Intn(3)
		statusChecks = append(statusChecks, checks.StatusCheck{
			ID:          fmt.Sprintf("dev-check-%d", i),
			URL:         domain[1],
			Interval:    interval[randI],
			HTTPTimeout: (randI + 1) * 5,
			Regions:     []string{"us-dev-1", "us-dev-2"},
			Modified:    time.Now().UTC(),
			Serial:      1,
			Active:      true,
		})
	}

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// TODO add step to delete all docments with region us-dev-1 or us-dev-2
	log.Info("Starting InsertMany")
	result, err := statusChecksColl.InsertMany(ctx, statusChecks)
	if err != nil {
		log.Fatal("Failed to InsertMany", zap.String("err", err.Error()))
	}
	log.Info("successfully created checks", zap.Int("check_count", len(result.InsertedIDs)))
}
