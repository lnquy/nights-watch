package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/go-chi/render"
	"github.com/lnquy/nights-watch/server/config"
	"github.com/lnquy/nights-watch/server/util"
	"github.com/lnquy/nights-watch/server/watcher/cpu"
	"github.com/lnquy/nights-watch/server/watcher/gpu"
	"github.com/lnquy/nights-watch/server/watcher/mem"
	"github.com/lnquy/nights-watch/server/watcher/net"
	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

type (
	Router struct {
		cfg    *config.Config
		sConn  *serial.Port
		ctx    context.Context
		cancel context.CancelFunc
	}

	Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	alertType int
)

const (
	atConfig  alertType = iota
	atCPU
	atMemory
	atGPU
	atNetwork
)

var (
	indexPage []byte
	favicon   []byte
)

func init() {
	var err error
	web := path.Join(util.GetWd(), "web")
	indexPage, err = ioutil.ReadFile(path.Join(web, "index.html"))
	if err != nil {
		logrus.Fatalf("router: failed to load index page: %s", err)
	}
	if favicon, err = ioutil.ReadFile(path.Join(web, "static", "img", "favicon.ico")); err != nil {
		logrus.Errorf("router: failed to load favicon: %s", err)
	}
}

func New(cfg *config.Config) *Router {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Router{
		cfg:    cfg,
		sConn:  newSerialConn(*cfg),
		ctx:    ctx,
		cancel: cancel,
	}
	if r.sConn != nil {
		go r.WatchStats() // Start watchers
	}
	return r
}

func newSerialConn(cfg config.Config) *serial.Port {
	logrus.Infof("router: connecting to Arduino on %s@%d", cfg.Serial.Port, cfg.Serial.Baud)
	serialConn, err := serial.OpenPort(&serial.Config{
		Name: cfg.Serial.Port,
		Baud: int(cfg.Serial.Baud),
	})
	if err != nil {
		logrus.Errorf("router: failed to connect to Arduino on %s@%d: %s", cfg.Serial.Port, cfg.Serial.Baud, err)
		logrus.Warn("=> Please define the serial config in config file or configure via web page!")
		return nil
	}

	// Sleep since Arduino will restart when new connection connected
	time.Sleep(2 * time.Second)
	logrus.Infof("router: Arduino connected")
	return serialConn
}

// First character determines the command type:
// 0: Config
// 1: CPU stats
// 2: Memory stats
// 3: GPU stats
// 4: Network stats
// z: Alert
func (rt *Router) WatchStats() {
	interval := time.Duration(rt.cfg.Stats.Interval) * time.Second

	// Reset all old stats/alerts then init new watchers
	cw := make(<-chan *cpu.Stats)
	rt.sConn.Write([]byte("1|-|-$"))
	rt.sConn.Write([]byte("z|1|0$"))
	if rt.cfg.Stats.CPU.Enabled {
		cw = cpu.NewWatcher().GetStats(rt.ctx, interval)
	}

	mw := make(<-chan *mem.Stats)
	rt.sConn.Write([]byte("2|-|-$"))
	rt.sConn.Write([]byte("z|2|0$"))
	if rt.cfg.Stats.Memory.Enabled {
		mw = mem.NewWatcher().GetStats(rt.ctx, interval)
	}

	gw := make(<-chan *gpu.Stats)
	rt.sConn.Write([]byte("3|-|-$"))
	rt.sConn.Write([]byte("z|3|0$"))
	if rt.cfg.Stats.GPU.Enabled {
		gw = gpu.NewWatcher().GetStats(rt.ctx, interval)
	}

	nw := make(<-chan *net.Stats)
	rt.sConn.Write([]byte("4|-|-$"))
	rt.sConn.Write([]byte("z|4|0$"))
	if rt.cfg.Stats.Network.Enabled {
		nw = net.NewWatcher().GetStats(rt.ctx, interval)
	}

	// Flags holds current alert status (ON/OFF)
	cwa, mwa, gwa, nwa := false, false, false, false
	// Flags holds alert status of each alert threshold
	cwParms, mwParms, gwParms, nwParms := make([]bool, 2), make([]bool, 1), make([]bool, 2), make([]bool, 2)
	for {
		select {
		case s := <-cw:
			if s == nil {
				continue
			}
			cmd := fmt.Sprintf("1|%.0f|%.0f$", s.Load, s.Temp)
			logrus.Debugf("CPU: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("CPU: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.CPU.LoadThreshold, uint(s.Load), cwParms, 0)
			checkThreshold(rt.cfg.Stats.CPU.TempThreshold, uint(s.Temp), cwParms, 1)
			alert(rt.sConn, cwParms, &cwa, atCPU)
		case s := <-mw:
			if s == nil {
				continue
			}
			cmd := fmt.Sprintf("2|%.0f|%d$", s.Load, s.Usage)
			logrus.Debugf("MEM: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("MEM: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.Memory.LoadThreshold, uint(s.Load), mwParms, 0)
			alert(rt.sConn, mwParms, &mwa, atMemory)
		case s := <-gw:
			if s == nil {
				continue
			}
			cmd := fmt.Sprintf("3|%.0f|%d$", s.Load, s.Mem)
			logrus.Debugf("GPU: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("GPU: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.GPU.LoadThreshold, uint(s.Load), gwParms, 0)
			checkThreshold(rt.cfg.Stats.GPU.MemThreshold, uint(s.Mem), gwParms, 1)
			alert(rt.sConn, gwParms, &gwa, atGPU)
		case s := <-nw:
			if s == nil {
				continue
			}
			cmd := fmt.Sprintf("4|%d|%d$", s.Download, s.Upload)
			logrus.Debugf("NET: %s", cmd)
			if _, err := rt.sConn.Write([]byte(cmd)); err != nil {
				logrus.Errorf("NET: failed to write stats to Arduino: %s", cmd)
			}
			checkThreshold(rt.cfg.Stats.Network.DownloadThreshold, uint(s.Download), nwParms, 0)
			checkThreshold(rt.cfg.Stats.Network.UploadThreshold, uint(s.Upload), nwParms, 1)
			alert(rt.sConn, nwParms, &nwa, atNetwork)
		case <-rt.ctx.Done():
			// TODO
			return
		}
	}
}

func (rt *Router) Stop() {
	if rt.cancel != nil {
		rt.cancel()
	}
	if rt.sConn != nil {
		rt.sConn.Close()
		logrus.Infof("router: Arduino connection closed")
	}
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

func (rt *Router) Login(w http.ResponseWriter, r *http.Request) {
	lg := Login{}
	if err := json.NewDecoder(r.Body).Decode(&lg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if lg.Username != rt.cfg.Admin.Username || lg.Password != rt.cfg.Admin.Password {
		http.Error(w, "Invalid username/password", http.StatusBadRequest)
		return
	}

	if err := sm.Load(r).PutString(w, "uid", rt.cfg.Admin.Username); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "nightswatch_uid",
		Value: rt.cfg.Admin.Username,
		MaxAge: 7 * 24 * 3600,
	})
	render.JSON(w, r, "Ok")
}

func (rt *Router) Logout(w http.ResponseWriter, r *http.Request) {
	if err := sm.Load(r).Destroy(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "nightswatch_uid",
		Value: "",
		Expires: time.Unix(0, 0),
	})
	render.JSON(w, r, "Ok")
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
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ard := config.Arduino{}
	if err = json.Unmarshal(b, &ard); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !ard.Stats.CPU.Enabled && !ard.Stats.Memory.Enabled && !ard.Stats.GPU.Enabled && !ard.Stats.Network.Enabled {
		http.Error(w, "At least one system statistics must be enabled", http.StatusBadRequest)
		return
	}

	tmpArd := rt.cfg.Arduino
	rt.cfg.Arduino = ard
	if _, err := rt.cfg.WriteToFile(""); err != nil {
		rt.cfg.Arduino = tmpArd // Fall back to old config
		logrus.Errorf("router: failed to write config to file: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Kill old watchers and Arduino connection then spawn new ones by new config
	logrus.Infof("router: config updated. Terminating old watchers and Arduino connection")
	rt.Stop()
	logrus.Infof("router: re-spawning Arduino connection and watchers")
	rt.sConn = newSerialConn(*rt.cfg)
	if rt.sConn == nil {
		http.Error(w, "Invalid serial configuration", http.StatusBadRequest)
		return
	}
	rt.ctx, rt.cancel = context.WithCancel(context.Background())
	go rt.WatchStats()
	render.JSON(w, r, "Ok")
}

func (rt *Router) ReloadTemplate(w http.ResponseWriter, r *http.Request) {
	var err error
	indexPage, err = ioutil.ReadFile(path.Join(util.GetWd(), "web", "index.html"))
	if err != nil {
		logrus.Fatalf("router: failed to load index page: %s", err)
	}
	w.Write([]byte("Ok"))
}

// Middleware
func (rt *Router) Authentication(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if uid, err := sm.Load(r).GetString("uid"); err != nil || uid == "" {
			http.Error(w, "Please login first", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
