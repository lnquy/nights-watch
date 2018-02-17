package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
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
		cfg         *config.Config
		sConn       *serial.Port
		ctx         context.Context
		cancel      context.CancelFunc
		sleepCtx    context.Context
		sleepCancel context.CancelFunc
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
		r.sleepTimer()
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
func (rt *Router) watchStats() {
	interval := time.Duration(rt.cfg.Stats.Interval) * time.Second
	rt.sConn.Write([]byte(fmt.Sprintf("y|%d$", rt.cfg.Sleep.NormalBrightness)))

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

func (rt *Router) Stop(stopSleepContext bool) {
	if rt.cancel != nil {
		rt.cancel()
	}
	if stopSleepContext && rt.sleepCancel != nil {
		rt.sleepCancel()
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
		Name:   "nightswatch_uid",
		Value:  rt.cfg.Admin.Username,
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
		Name:    "nightswatch_uid",
		Value:   "",
		Expires: time.Unix(0, 0),
	})
	render.JSON(w, r, "Ok")
}

func (rt *Router) GetIndexPage(w http.ResponseWriter, r *http.Request) {
	if !rt.cfg.Admin.ForceLogin {
		// Allow anonymous to configure via webpage.
		// Check if nightswatch_uid cookie available or not, if not then write new guest cookie
		// so frontend can behave normally.
		if _, err := r.Cookie("nightswatch_uid"); err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:   "nightswatch_uid",
				Value:  "guest",
				MaxAge: 7 * 24 * 3600,
			})
		}
	} else {
		// Force administrator to login before configuring anything.
		// Check if nightswatch_uid cookie available or not, if available and is a "guest" cookie then
		// delete it so frontend will require login again.
		if c, err := r.Cookie("nightswatch_uid"); err == nil && c.Value == "guest" {
			http.SetCookie(w, &http.Cookie{
				Name:    "nightswatch_uid",
				Value:   "guest",
				Expires: time.Unix(0, 0),
			})
		}
	}
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
	rt.Stop(true)
	logrus.Infof("router: re-spawning Arduino connection and watchers")
	rt.sConn = newSerialConn(*rt.cfg)
	if rt.sConn == nil {
		http.Error(w, "Invalid serial configuration", http.StatusBadRequest)
		return
	}
	rt.ctx, rt.cancel = context.WithCancel(context.Background())
	rt.sleepTimer() // Check sleep time and decide to start watchers or not
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
		if rt.cfg.Admin.ForceLogin {
			if uid, err := sm.Load(r).GetString("uid"); err != nil || uid == "" {
				http.Error(w, "Please login first", http.StatusUnauthorized)
				return
			}
		} else {
			// Check if nightswatch_uid cookie available or not, if not then write new guest cookie
			// so frontend can behave as normal
			if _, err := r.Cookie("nightswatch_uid"); err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:   "nightswatch_uid",
					Value:  "guest",
					MaxAge: 7 * 24 * 3600,
				})
			}
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (rt *Router) sleepTimer() {
	if rt.cfg.Sleep.Start == rt.cfg.Sleep.End {
		logrus.Infof("router: sleep time disabled")
		go rt.watchStats()
		return
	}
	startSpl, endSpl := strings.Split(rt.cfg.Sleep.Start, ":"), strings.Split(rt.cfg.Sleep.End, ":")
	startHour, _ := strconv.Atoi(startSpl[0])
	startMin, _ := strconv.Atoi(startSpl[1])
	endHour, _ := strconv.Atoi(endSpl[0])
	endMin, _ := strconv.Atoi(endSpl[1])

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMin, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMin, 0, 0, now.Location())
	logrus.Debugf("TIME: %s - %s - %s", now.String(), start.String(), end.String())
	// Add up one day if timer was over
	if now.Unix() > end.Unix() {
		start = start.Add(24 * time.Hour)
		end = end.Add(24 * time.Hour)
	}
	logrus.Debugf("AFTER: %s - %s - %s", now.String(), start.String(), end.String())

	if now.Unix() >= start.Unix() && now.Unix() < end.Unix() { // In sleep time
		go rt.scheduleSleepEnd(start, end)
	} else { // In normal time
		go rt.watchStats()
		go rt.scheduleSleepStart(start, end)
	}
}

func (rt *Router) scheduleSleepStart(start, end time.Time) {
	logrus.Infof("sleep: scheduling new sleep start at %s", start.String())
	logrus.Debugf("sleep start: %s - %s", start, end)
	if rt.sConn != nil {
		rt.sConn.Write([]byte(fmt.Sprintf("y|%d$", rt.cfg.Sleep.NormalBrightness))) // Set LCD brightness to normal
	}
	rt.sleepCtx, rt.sleepCancel = context.WithCancel(context.Background())
	if start.Unix() < time.Now().Unix() {
		start = start.Add(24 * time.Hour)
	}
	startTimer := time.NewTicker(start.Sub(time.Now()))
	select {
	case <-startTimer.C:
		logrus.Infof("sleep start: timer ended. Stop all watchers and close Arduino connection")
		startTimer.Stop()
	case <-rt.sleepCtx.Done():
		logrus.Infof("sleep start: context done. Sleep start stopped")
		startTimer.Stop()
		return
	}

	go rt.scheduleSleepEnd(start, end)
	logrus.Infof("sleep start: done")
}

func (rt *Router) scheduleSleepEnd(start, end time.Time) {
	logrus.Infof("sleep: scheduling new sleep end at %s", end.String())
	logrus.Debugf("sleep end: %s - %s", start, end)
	if rt.sConn != nil {
		rt.sConn.Write([]byte(fmt.Sprintf("y|%d$", rt.cfg.Sleep.SleepBrightness))) // Dim the LCD
	}
	rt.sleepCtx, rt.sleepCancel = context.WithCancel(context.Background())
	rt.Stop(false)
	if end.Unix() < time.Now().Unix() {
		end = end.Add(24 * time.Hour)
	}
	endTimer := time.NewTimer(end.Sub(time.Now()))
	select {
	case <-endTimer.C:
		logrus.Infof("sleep end: timer ended. Initialize new Arduino connection and respawn all watchers")
		endTimer.Stop()
	case <-rt.sleepCtx.Done():
		logrus.Infof("sleep end: context done. Sleep end stopped")
		endTimer.Stop()
		return
	}

	rt.sConn = newSerialConn(*rt.cfg)
	if rt.sConn == nil {
		logrus.Panicf("router: invalid serial configuration")
	}
	rt.ctx, rt.cancel = context.WithCancel(context.Background())
	go rt.watchStats()
	go rt.scheduleSleepStart(start.Add(24*time.Hour), end.Add(24*time.Hour))
	logrus.Infof("sleep end: done")
}
