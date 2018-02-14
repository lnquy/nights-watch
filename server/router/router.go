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
	"net/http"
	"io/ioutil"
	"path"
	"github.com/lnquy/nights-watch/server/util"
	"github.com/go-chi/render"
	"encoding/json"
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

var (
	indexPage []byte
	favicon []byte
)

func init() {
	var err error
	web := path.Join(util.GetWd(), "web")
	indexPage , err = ioutil.ReadFile(path.Join(web, "index.html"))
	if err != nil {
		logrus.Fatalf("router: failed to load index page: %s", err)
	}
	if favicon, err = ioutil.ReadFile(path.Join(web, "static", "img", "favicon.ico")); err != nil {
		logrus.Errorf("router: failed to load favicon: %s", err)
	}
}

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
func (rt *Router) WatchStats() {
	// TODO: Enable/Disable watcher
	interval := time.Duration(rt.cfg.Stats.Interval) * time.Second
	cw := cpu.NewWatcher().GetStats(rt.ctx, interval)
	mw := mem.NewWatcher().GetStats(rt.ctx, interval)
	nw := net.NewWatcher().GetStats(rt.ctx, interval)
	logrus.Info("router: watchers started")

	// Flags holds current alert status (ON/OFF)
	cwa, mwa, nwa := false, false, false
	// Flags holds alert status of each alert threshold
	cwParms, mwParms, nwParms := make([]bool, 2), make([]bool, 1), make([]bool, 2)
	for {
		select {
		case s := <-cw:
			cmd := fmt.Sprintf("1|%.0f|%.0f$", s.Load, s.Temp)
			logrus.Debugf("CPU: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("CPU: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.CPU.LoadThreshold, uint(s.Load), cwParms, 0)
			checkThreshold(rt.cfg.Stats.CPU.TempThreshold, uint(s.Temp), cwParms, 1)
			alert(rt.sConn, cwParms, &cwa, atCPU)
		case s := <-mw:
			cmd := fmt.Sprintf("2|%.0f|%d$", s.Load, s.Usage)
			logrus.Debugf("MEM: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("MEM: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.Memory.LoadThreshold, uint(s.Load), mwParms, 0)
			alert(rt.sConn, mwParms, &mwa, atMemory)
		case s := <-nw:
			cmd := fmt.Sprintf("4|%d|%d$", s.Download, s.Upload)
			logrus.Debugf("NET: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("NET: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.Network.DownloadThreshold, uint(s.Download), nwParms, 0)
			checkThreshold(rt.cfg.Stats.Network.UploadThreshold, uint(s.Upload), nwParms, 1)
			alert(rt.sConn, nwParms, &nwa, atNetwork)
		case <-rt.ctx.Done():
			return
		}
	}
}

func (rt *Router) Stop() {
	rt.cancel()
	logrus.Infof("router: watchers stopped")
	rt.sConn.Close()
	logrus.Infof("router: Arduino connection closed")
}

func checkThreshold(threshold, value uint, parms []bool, idx int) {
	if threshold <= 0 || value < threshold {
		parms[idx] = false
	} else { //  // Threshold reached
		parms[idx] = true
	}
}

func alert(sConn *serial.Port, parms []bool, flag *bool, at alertType) {
	for _, v := range parms {
		if v { // Threshold reached
			if !*flag { // Alert is not fired yet -> Turn on alert and update status
				cmd := fmt.Sprintf("z|%d|1$", at)
				if _, err := sConn.Write([]byte(cmd)); err != nil {
					logrus.Errorf("alert: failed to write alert to Arduino: %s", cmd)
					return
				}
				*flag = true
			}
			return
		}
	}
	// Back to normal state but current alert is ON -> Turn off alert and update status
	if *flag {
		cmd := fmt.Sprintf("z|%d|0$", at)
		if _, err := sConn.Write([]byte(cmd)); err != nil {
			logrus.Errorf("alert: failed to write alert to Arduino: %s", cmd)
			return
		}
		*flag = false
	}
}

// Routing
func (rt *Router) Favicon(w http.ResponseWriter, r *http.Request) {
	w.Write(favicon)
}

func (rt *Router) GetIndexPage(w http.ResponseWriter, r *http.Request) {
	w.Write(indexPage)
}

func (rt *Router) GetCOMPorts(w http.ResponseWriter, r *http.Request) {
	ports, err := util.GetCOMPorts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logrus.Infof("router: COM ports: %v", ports)
	render.JSON(w, r, ports)
}

func (rt *Router) GetConfig(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(rt.cfg.Arduino)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func (rt *Router) UpdateConfig(w http.ResponseWriter, r *http.Request) {

}

func (rt *Router) ReloadTemplate(w http.ResponseWriter, r *http.Request) {
	var err error
	indexPage , err = ioutil.ReadFile(path.Join(util.GetWd(), "web", "index.html"))
	if err != nil {
		logrus.Fatalf("router: failed to load index page: %s", err)
	}
	w.Write([]byte("Ok"))
}
