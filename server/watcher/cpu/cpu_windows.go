package cpu

import (
	"context"
	"time"

	"github.com/lnquy/nights-watch/server/watcher/util"
	pscpu "github.com/shirou/gopsutil/cpu"
)

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	stats := &Stats{}
	go func() {
		for {
			select {
			case <-ticker.C:
				percs, err := pscpu.Percent(0, false)
				if err != nil {
					statsChan <- stats
					continue
				}
				stats.Load = util.GetAverage(percs)
				statsChan <- stats
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}()
	return statsChan
}
