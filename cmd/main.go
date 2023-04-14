// package main get's the people going
package main

import (
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/larntz/status/cmd/worker"
	"github.com/larntz/status/internal/data"
)

func main() {
	env, set := os.LookupEnv("ENVIRONMENT")
	var log *zap.Logger
	var err error
	if !set || env == "development" {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Println("Unable to setup logger. Exiting...")
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting")
	if len(os.Args) < 2 {
		log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}

	switch os.Args[1] {
	case "create-dev-checks":
		if len(os.Args) < 3 {
			log.Fatal("Not enough args, must specify a csv file for creating checks.")
		}
		data.CreateDevChecks(os.Args[2], log)
	case "controller":
		log.Fatal("controller ain't ready")
		// dbLogin(ctx, &app)
		// defer app.DbClient.Disconnect(ctx)
		// controller.StartController(&app)
	case "worker":
		var ok bool
		state := worker.NewState()
		state.Region, ok = os.LookupEnv("WORKER_REGION")
		if !ok {
			log.Fatal("WORKER_REGION env var not set. Exiting.")
		}
		state.Log = log
		state.HTTPTransport = &http.Transport{}
		state.DBClient = &data.MongoDB{}
		if err := state.DBClient.Connect(); err != nil {
			log.Fatal("Connect() to database failed.", zap.String("error", err.Error()))
		}
		defer state.DBClient.Disconnect()
		state.RunWorker()

	default:
		log.Fatal("Must specify subcommand: 'controller' or 'worker'")
	}
}
