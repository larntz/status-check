// Package application package stores our app state
package application

import (
	"context"
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	"go.uber.org/zap"
)

// State type holds our controller state
type State struct {
	Checks          checks.Checks
	ChecksMutex     sync.Mutex
	ChecksTimestamp time.Time
	Ctx             context.Context
	DbClient        *data.Database
	Log             *zap.Logger
	Region          string
}
