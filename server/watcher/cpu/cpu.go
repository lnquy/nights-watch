package cpu

type (
	Watcher interface {
		GetStats() *Stats
	}

	Stats struct {
		Load float64
		Temp float64
	}

	watcher struct{}
)

func NewWatcher() Watcher {
	return &watcher{}
}
