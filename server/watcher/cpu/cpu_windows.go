package cpu

import (
	"time"

	pscpu "github.com/shirou/gopsutil/cpu"
)

func (w *watcher) GetStats() *Stats {
	stats := &Stats{}
	percs, err := pscpu.Percent(time.Second, false)
	if err != nil {
		return stats
	}
	stats.Load = percs[0]
	return stats
}
