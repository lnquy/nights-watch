// +build linux

package gpu

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	stats := &Stats{}
	go func() {
		logrus.Infof("watcher: GPU watcher started")
		for {
			select {
			case <-ticker.C:
				// TODO
				stats.Load = 0
				stats.Mem = 0
				statsChan <- stats
			case <-ctx.Done():
				close(statsChan)
				ticker.Stop()
				logrus.Infof("watcher: GPU watcher stopped")
				return
			}
		}
	}()
	return statsChan
}
