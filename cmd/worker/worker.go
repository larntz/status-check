// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"time"

	"github.com/larntz/status/internal/checks"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func sendStatusCheckResult(dbClient *mongo.Client, log *zap.Logger, result *checks.StatusCheckResult) {
	coll := dbClient.Database("status").Collection("check_results")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	iResult, err := coll.InsertOne(ctx, result)
	if err != nil {
		log.Error("Failed to InsertMany", zap.String("err", err.Error()))
		return
	}
	log.Debug("Successfully inserted document", zap.Any("id", iResult.InsertedID), zap.String("request_id", result.Metadata.CheckID))
}
