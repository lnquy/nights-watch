package mem

import (
	"context"
	"time"
)

type (
	Watcher interface {
		GetStats(ctx context.Context, interval time.Duration) <-chan *Stats
	}

	Stats struct {
		Load  float64
		Usage uint64
	}

	watcher struct{}
)

func NewWatcher() Watcher {
	return &watcher{}
}
