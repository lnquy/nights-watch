package gpu

import (
	"context"
	"time"
)

type GPUVendor string

const (
	NVIDIA GPUVendor = "nvidia"
	AMD    GPUVendor = "amd"
)

type (
	Watcher interface {
		GetStats(ctx context.Context, interval time.Duration, vendor GPUVendor) <-chan *Stats
	}

	Stats struct {
		Load float64
		Mem  uint64
	}

	watcher struct{}
)

func NewWatcher() Watcher {
	return &watcher{}
}
