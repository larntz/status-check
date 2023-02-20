// package main get's the people going
package main

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/larntz/status/cmd/controller"
	"github.com/larntz/status/cmd/worker"
	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/data"
)

func dbLogin(ctx context.Context, app *application.State) {
	log.Info("DB Login start")
	err := data.Connect(ctx, app)
	if err != nil {
		log.Fatalf("DB Login failed: %s", err)
	}
	log.Info("DB Login succesful")
}

func main() {
	log.Info("Starting up...")
	if len(os.Args) < 2 {
		log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var app application.State

	switch os.Args[1] {
	case "create-dev-checks":
		if len(os.Args) < 3 {
			log.Fatalf("Not enough args, must specify a csv file for creating checks.")
		}
		dbLogin(ctx, &app)
		defer app.DbClient.Disconnect(ctx)
		data.CreateDevChecks(app.DbClient, os.Args[2])
	case "controller":
		dbLogin(ctx, &app)
		defer app.DbClient.Disconnect(ctx)
		controller.StartController(&app)
	case "worker":
		region, ok := os.LookupEnv("WORKER_REGION")
		if !ok {
			log.Fatalf("WORKER_REGION env var not set. Exiting.")
		}
		app.Region = region
		dbLogin(ctx, &app)
		defer app.DbClient.Disconnect(ctx)
		worker.StartWorker(&app)
	default:
		log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}
}
