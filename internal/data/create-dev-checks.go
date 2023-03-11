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
	"go.uber.org/zap"
)

// CreateDevChecks will create a few checks we can use during development
func CreateDevChecks(client *mongo.Client, filename string, log *zap.Logger) {
	// create some checks to use during development
	statusChecksColl := client.Database("status").Collection("status_checks")

	file, err := os.Open(filename)
	if err != nil {
		log.Error("failed to open file", zap.String("err", err.Error()))
	}
	reader := csv.NewReader(file)
	domains, _ := reader.ReadAll()

	var statusChecks []interface{}
	interval := []int{60, 300, 900}

	for i, domain := range domains {
		randI := rand.Intn(2)
		statusChecks = append(statusChecks, checks.StatusCheck{
			ID:       fmt.Sprintf("dev-check-%d", i),
			URL:      domain[1],
			Interval: interval[randI],
			Regions:  []string{"us-dev-1", "us-dev-2"},
			Modified: time.Now().UTC(),
			Serial:   1,
			Active:   true,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	// TODO add step to delete all docments with region us-dev-1 or us-dev-2
	log.Info("Starting InsertMany")
	result, err := statusChecksColl.InsertMany(ctx, statusChecks)
	if err != nil {
		log.Fatal("Failed to InsertMany", zap.String("err", err.Error()))
	}
	log.Info("successfully created checks", zap.Int("check_count", len(result.InsertedIDs)))
}
