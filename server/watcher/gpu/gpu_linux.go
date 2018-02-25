// +build linux

package gpu

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/mindprince/gonvml"
)

// GetStats periodically get GPU statistics from system and returns that value to a Stats channel.
// Currently supports for 2 GPU vendors:
//   - NVIDIA: Get stats via NVML binding.
//   - AMD:
func (w *watcher) GetStats(ctx context.Context, interval time.Duration, vendor GPUVendor) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	stats := &Stats{}

	var nvidiaCard gonvml.Device
	if vendor == NVIDIA {
		var err error
		var devices uint
		if err = gonvml.Initialize(); err != nil {
			logrus.Panicf("gpu: failed to detect NVIDIA card: %s", err)
		}
		if devices, err = gonvml.DeviceCount(); err != nil || devices == 0 {
			logrus.Panicf("gpu: failed to detect NVIDIA card: %s", err)
		}
		logrus.Infof("gpu: %d NVIDIA card(s) detected", devices)
		// TODO: Only get stats from first card for now. Support multiple cards later [?].
		if nvidiaCard, err = gonvml.DeviceHandleByIndex(uint(0)); err != nil {
			logrus.Panicf("gpu: failed to watch on first NVIDIA card: %s", err)
		}
	}

	go func() {
		logrus.Infof("watcher: GPU watcher started")
		for {
			select {
			case <-ticker.C:
				if vendor == NVIDIA {
					load, _, err := nvidiaCard.UtilizationRates()
					if err != nil {
						logrus.Error(err)
						statsChan <- stats
					}
					stats.Load = float64(load)
					_, used, err := nvidiaCard.MemoryInfo()
					if err != nil {
						logrus.Error(err)
						statsChan <- stats
					}
					stats.Mem = used / 1000000
					statsChan <- stats
				}
				// TODO: Other vendors
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
