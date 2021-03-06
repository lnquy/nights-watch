package cpu

import (
	"context"
	"time"
)

type (
	Watcher interface {
		GetStats(ctx context.Context, interval time.Duration) <-chan *Stats
	}

	Stats struct {
		Load float64
		Temp float64
	}

	watcher struct{}
)

func NewWatcher() Watcher {
	return &watcher{}
}
