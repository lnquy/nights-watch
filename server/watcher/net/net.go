package net

import (
	"context"
	"time"

	psnet "github.com/lnquy/gopsutil/net"
	"github.com/sirupsen/logrus"
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

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	lastStats, stats := &Stats{}, &Stats{}
	sec := uint64(interval / time.Second)

	go func() {
		logrus.Infof("watcher: NET watcher started")
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
				close(statsChan)
				ticker.Stop()
				logrus.Infof("watcher: NET watcher stopped")
				return
			}
		}
	}()
	return statsChan
}
