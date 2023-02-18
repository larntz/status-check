// package main get's the people going
package main

import (
	"log"
	"time"

	"github.com/larntz/status/cmd/controller"
	"github.com/larntz/status/cmd/worker"
	"github.com/larntz/status/internal/data"
)

func main() {
	log.Println("INFO: DB Connection check start")
	if data.Ping() {
		log.Println("INFO: DB Connection check succesful")
	} else {
		log.Fatal("ERROR: DB Connection check failed ")
	}
	data.CreateDevChecks()

	go controller.StartController()
	time.Sleep(time.Second * 2)
	worker.StartWorker()
}
