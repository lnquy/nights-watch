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

	alertType int
)

const (
	atConfig alertType = iota
	atCPU
	atMemory
	atGPU
	atNetwork
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

	// Flags holds alert status (ON/OFF)
	cwa, mwa, nwa := false, false, false
	for {
		select {
		case s := <-cw:
			cmd := fmt.Sprintf("1|%.0f|%.0f$", s.Load, s.Temp)
			logrus.Debugf("CPU: %s", cmd)
			if _, err := r.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("CPU: failed to write stats to Arduino: %s", cmd)
			}
			r.alert(r.cfg.Stats.CPU.LoadThreshold, uint(s.Load), &cwa, atCPU)
			r.alert(r.cfg.Stats.CPU.TempThreshold, uint(s.Temp), &cwa, atCPU)
		case s := <-mw:
			cmd := fmt.Sprintf("2|%.0f|%d$", s.Load, s.Usage)
			logrus.Debugf("MEM: %s", cmd)
			if _, err := r.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("MEM: failed to write stats to Arduino: %s", cmd)
			}
			r.alert(r.cfg.Stats.Memory.LoadThreshold, uint(s.Load), &mwa, atMemory)
		case s := <-nw:
			cmd := fmt.Sprintf("4|%d|%d$", s.Download, s.Upload)
			logrus.Debugf("NET: %s", cmd)
			if _, err := r.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("NET: failed to write stats to Arduino: %s", cmd)
			}
			r.alert(r.cfg.Stats.Network.DownloadThreshold, uint(s.Download), &nwa, atNetwork)
			r.alert(r.cfg.Stats.Network.UploadThreshold, uint(s.Upload), &nwa, atNetwork)
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

func (r *Router) alert(threshold, value uint, flag *bool, at alertType) {
	if threshold <= 0 {
		return
	}
	// Alert ON
	if value >= threshold && !*flag {
		cmd := fmt.Sprintf("z|%d|1$", at)
		if _, err := r.sConn.Write([]byte(cmd)); err != nil {
			logrus.Errorf("alert: failed to write alert to Arduino: %s", cmd)
			return
		}
		*flag = true
		return
	}
	// Alert OFF
	if value < threshold && *flag {
		cmd := fmt.Sprintf("z|%d|0$", at)
		if _, err := r.sConn.Write([]byte(cmd)); err != nil {
			logrus.Errorf("alert: failed to write alert to Arduino: %s", cmd)
			return
		}
		*flag = false
		return
	}
}
