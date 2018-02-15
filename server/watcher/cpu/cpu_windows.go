package cpu

import (
	"context"
	"time"

	"github.com/lnquy/nights-watch/server/util"
	pscpu "github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
)

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	stats := &Stats{}
	go func() {
		logrus.Infof("watcher: CPU watcher started")
		for {
			select {
			case <-ticker.C:
				percs, err := pscpu.Percent(interval, false)
				if err != nil {
					statsChan <- stats
					continue
				}
				stats.Load = util.GetAverage(percs)
				statsChan <- stats
			case <-ctx.Done():
				close(statsChan)
				ticker.Stop()
				logrus.Infof("watcher: CPU watcher stopped")
				return
			}
		}
	}()
	return statsChan
}
