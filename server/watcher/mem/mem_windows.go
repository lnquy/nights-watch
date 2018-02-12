package mem

import (
	"context"
	"time"

	psmem "github.com/shirou/gopsutil/mem"
)

func (w *watcher) GetStats(ctx context.Context, interval time.Duration) <-chan *Stats {
	ticker := time.NewTicker(interval)
	statsChan := make(chan *Stats, 10)
	stats := &Stats{}
	go func() {
		for {
			select {
			case <-ticker.C:
				vm, err := psmem.VirtualMemory()
				if err != nil {
					statsChan <- stats
					continue
				}
				stats.Load = vm.UsedPercent
				stats.Usage = vm.Used / 1000000 // MB
				statsChan <- stats
			case <-ctx.Done():
				ticker.Stop()
			}
		}
	}()
	return statsChan
}
