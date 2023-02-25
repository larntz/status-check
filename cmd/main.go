// package main get's the people going
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/larntz/status/cmd/controller"
	"github.com/larntz/status/cmd/worker"
	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/data"
)

func main() {
	var app application.State
	var err error
	app.Log, err = zap.NewProduction()
	if err != nil {
		fmt.Println("Unable to setup logger. Exiting...")
		os.Exit(1)
	}
	defer app.Log.Sync()

	app.Log.Info("Starting up...")
	if len(os.Args) < 2 {
		app.Log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch os.Args[1] {
	case "create-dev-checks":
		if len(os.Args) < 3 {
			app.Log.Fatal("Not enough args, must specify a csv file for creating checks.")
		}
		dbLogin(ctx, &app)
		defer app.DbClient.Disconnect(ctx)
		data.CreateDevChecks(app.DbClient, os.Args[2], app.Log)
	case "controller":
		dbLogin(ctx, &app)
		defer app.DbClient.Disconnect(ctx)
		controller.StartController(&app)
	case "worker":
		var ok bool
		app.Region, ok = os.LookupEnv("WORKER_REGION")
		if !ok {
			app.Log.Fatal("WORKER_REGION env var not set. Exiting.")
		}
		dbLogin(ctx, &app)
		defer app.DbClient.Disconnect(ctx)
		worker.StartWorker(&app)
	default:
		app.Log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}
}

func dbLogin(ctx context.Context, app *application.State) {
	app.Log.Info("DB Login start")
	err := data.Connect(ctx, app)
	if err != nil {
		app.Log.Fatal("DB Login failed", zap.String("err", err.Error()))
	}
	app.Log.Info("DB Login succesful")
}
