// Package application package stores our app state
package application

import (
	"context"
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
	"go.mongodb.org/mongo-driver/mongo"
)

// State type holds our application state
type State struct {
	Checks          checks.Checks
	ChecksMutex     sync.Mutex
	ChecksTimestamp time.Time
	Ctx             context.Context
	DbClient        *mongo.Client // is this abstracted enough?
	Region          string
}
