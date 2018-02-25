// +build linux

// TODO: Linux and Darwin can be merged together later
// Also Windows, if can be supported via shirou/cpu

package cpu

import (
	"context"
	"time"

	"github.com/lnquy/nights-watch/server/util"
	pscpu "github.com/lnquy/gopsutil/cpu"
	pshost "github.com/lnquy/gopsutil/host"
	"github.com/sirupsen/logrus"
	"strings"
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
				temps, err := pshost.SensorsTemperatures()
				if err != nil {
					statsChan <- stats
					logrus.Error(err)
					continue
				}
				logrus.Debugf("TEMP: %v", temps)
				stats.Temp = getCPUTemperature(temps)
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

// Return the first CPU package temperature if possible, otherwise return the maximum temperature among all cores
func getCPUTemperature(temps []pshost.TemperatureStat) float64 {
	if temp := getPackageTemperature(temps); temp != 0.0 {
		return temp
	}
	max := 0.0
	for _, t := range temps {
		if strings.HasPrefix(t.SensorKey, "coretemp_core") && strings.HasSuffix(t.SensorKey, "_input") {
			if t.Temperature > max {
				max = t.Temperature
			}
		}
	}
	return max
}

func getPackageTemperature(temps []pshost.TemperatureStat) float64 {
	for _, t := range temps {
		if t.SensorKey == "coretemp_packageid0_input" {
			return t.Temperature
		}
	}
	return 0.0
}
