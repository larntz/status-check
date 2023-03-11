// package main get's the people going
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/larntz/status/cmd/worker"
	"github.com/larntz/status/internal/data"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Println("Unable to setup logger. Exiting...")
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting")
	if len(os.Args) < 2 {
		log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch os.Args[1] {
	case "create-dev-checks":
		if len(os.Args) < 3 {
			log.Fatal("Not enough args, must specify a csv file for creating checks.")
		}
		client := dbLogin(ctx, log)
		defer client.Disconnect(ctx)
		data.CreateDevChecks(client, os.Args[2], log)
	case "controller":
		log.Fatal("controller ain't ready")
		// dbLogin(ctx, &app)
		// defer app.DbClient.Disconnect(ctx)
		// controller.StartController(&app)
	case "worker":
		var ok bool
		scheduler := worker.NewScheduler()
		scheduler.Region, ok = os.LookupEnv("WORKER_REGION")
		if !ok {
			log.Fatal("WORKER_REGION env var not set. Exiting.")
		}
		scheduler.Log = log
		scheduler.DBClient = dbLogin(ctx, log)
		defer scheduler.DBClient.Disconnect(ctx)
		scheduler.Start()

	default:
		log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}
}

func dbLogin(ctx context.Context, log *zap.Logger) *mongo.Client {
	log.Debug("DB Login start")
	dbClient, err := data.Connect(ctx, log)
	if err != nil {
		log.Fatal("DB Login failed", zap.String("err", err.Error()))
	}
	log.Info("DB Login succesful")
	return dbClient
}
