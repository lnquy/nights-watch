package router

import (
	"github.com/tarm/serial"
	"context"
	"github.com/sirupsen/logrus"
	"fmt"
	"github.com/lnquy/nights-watch/server/watcher/cpu"
	"github.com/lnquy/nights-watch/server/watcher/mem"
	"github.com/lnquy/nights-watch/server/watcher/net"
	"github.com/lnquy/nights-watch/server/config"
	"time"
)

type (
	Router struct {
		cfg *config.Config
		sConn *serial.Port
		ctx context.Context
		cancel context.CancelFunc
	}
)

func New(cfg *config.Config, sConn *serial.Port) *Router {
	ctx, cancel := context.WithCancel(context.Background())
	return &Router {
		cfg: cfg,
		sConn: sConn,
		ctx: ctx,
		cancel: cancel,
	}
}

// First character dertermines the command type:
// 0: Config
// 1: CPU stats
// 2: Memory stats
// 3: GPU stats
// 4: Network stats
// z: Alert
func (r *Router) WatchStats() {
	interval := time.Duration(r.cfg.Stats.StatsConf.Interval) * time.Second
	cw := cpu.NewWatcher().GetStats(r.ctx, interval)
	mw := mem.NewWatcher().GetStats(r.ctx, interval)
	nw := net.NewWatcher().GetStats(r.ctx, interval)
	logrus.Info("router: watchers started")
	for {
		select {
		case s := <-cw:
			cmd := fmt.Sprintf("1|%.0f|%.0f$", s.Load, s.Temp)
			logrus.Debugf("CPU: %s", cmd)
			if _, err := r.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("Failed to write CPU stats to Arduino: %s", cmd)
			}
		case s := <-mw:
			cmd := fmt.Sprintf("2|%.0f|%d$", s.Load, s.Usage)
			logrus.Debugf("MEM: %s", cmd)
			if _, err := r.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("Failed to write MEM stats to Arduino: %s", cmd)
			}

			// TODO: Test threshold
			if _, err := r.sConn.Write([]byte(fmt.Sprintf("z|1|1$"))); err != nil {
				logrus.Errorf("Failed to write MEM alert to Arduino: %s", cmd)
			}
			if _, err := r.sConn.Write([]byte(fmt.Sprintf("z|2|1$"))); err != nil {
				logrus.Errorf("Failed to write MEM alert to Arduino: %s", cmd)
			}
			if _, err := r.sConn.Write([]byte(fmt.Sprintf("z|3|1$"))); err != nil {
				logrus.Errorf("Failed to write MEM alert to Arduino: %s", cmd)
			}
			if _, err := r.sConn.Write([]byte(fmt.Sprintf("z|4|1$"))); err != nil {
				logrus.Errorf("Failed to write MEM alert to Arduino: %s", cmd)
			}
		case s := <-nw:
			cmd := fmt.Sprintf("4|%d|%d$", s.Download, s.Upload)
			logrus.Debugf("NET: %s", cmd)
			if _, err := r.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("Failed to write NET stats to Arduino: %s", cmd)
			}
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *Router) Stop() {
	r.cancel()
	logrus.Infof("router: watchers stopped")
	r.sConn.Close()
	logrus.Infof("router: Arduino connection closed")
}
