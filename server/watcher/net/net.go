package net

import (
	"context"
	"time"
)

type (
	Watcher interface {
		GetStats(ctx context.Context, interval time.Duration) <-chan *Stats
	}

	Stats struct {
		Download uint64
		Upload   uint64
	}

	watcher struct{}
)

func NewWatcher() Watcher {
	return &watcher{}
}
