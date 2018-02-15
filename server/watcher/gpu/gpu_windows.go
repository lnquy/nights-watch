package gpu

import (
	"context"
	"time"
)

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	stats := &Stats{}
	go func() {
		for {
			select {
			case <-ticker.C:
				// TODO
				stats.Load = 0
				stats.Mem = 0
				statsChan <- stats
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}()
	return statsChan
}
