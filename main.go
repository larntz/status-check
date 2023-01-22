// package main get's the people going
package main

import (
	"time"

	"github.com/larntz/status/controller"
	"github.com/larntz/status/worker"
)

func main() {
	go controller.StartController()
	time.Sleep(time.Second * 2)
	worker.StartWoker()
}
