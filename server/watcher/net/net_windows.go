package net

import (
	"context"
	"time"

	psnet "github.com/shirou/gopsutil/net"
	"github.com/sirupsen/logrus"
)

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	lastStats, stats := &Stats{}, &Stats{}
	sec := uint64(interval / time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				netStats, err := psnet.IOCounters(false)
				if err != nil {
					logrus.Error(err)
					statsChan <- stats
				}
				stats.Download = (netStats[0].BytesRecv - lastStats.Download) / sec / 1000
				stats.Upload = (netStats[0].BytesSent - lastStats.Upload) / sec / 1000
				lastStats.Download = netStats[0].BytesRecv
				lastStats.Upload = netStats[0].BytesSent
				statsChan <- stats
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}()
	return statsChan
}
