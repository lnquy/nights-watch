package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
	"github.com/lnquy/nights-watch/server/config"
	"github.com/go-chi/chi"
	"net/http"
	"log"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/lnquy/nights-watch/server/router"
	"path"
	"github.com/lnquy/nights-watch/server/util"
)

var (
	fAddr     = flag.String("ip", "", "IP address where server bind to")
	fPort     = flag.Uint("port", 12345, "Port where server bind to")
	fUsername = flag.String("user", "", "Administrator username")
	fPassword = flag.String("pwd", "", "Administrator password")
	fLogLevel = flag.String("log", "info", "Log level")

	fSerialPort = flag.String("s_port", "", "Serial port to connect to Arduino")
	fSerialBaud = flag.Uint("s_baud", 9600, "Serial port baud speed")
)

func main() {
	cfg := config.LoadFromFile("")
	flag.Parse()
	overrideConfigs(cfg)
	if _, err := cfg.WriteToFile(""); err != nil {
		logrus.Fatalf("main: failed to write config to file: %s", err)
	}

	logrus.Infof("main: connecting to Arduino on %s@%d", cfg.Serial.Port, cfg.Serial.Baud)
	serialConn, err := serial.OpenPort(&serial.Config{
		Name: cfg.Serial.Port,
		Baud: int(cfg.Serial.Baud),
	})
	if err != nil {
		logrus.Fatal(err)
	}
	time.Sleep(1 * time.Second) // Sleep since Arduino will restart when new connection connected
	logrus.Infof("main: Arduino connected")
	// TODO: Write config here

	logrus.Infof("main: starting web server")
	r := chi.NewRouter()
	r.Use(middleware.DefaultLogger)
	r.Use(middleware.DefaultCompress)
	r.Use(middleware.Recoverer)

	handler := router.New(cfg, serialConn)
	go handler.WatchStats()

	// Routing
	dir := path.Join(util.GetWd(), "web", "static")
	fileServer(r, "/static", http.Dir(dir))
	r.Get("/", handler.GetIndexPage)
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/serial", func(r chi.Router) {
			r.Get("/", handler.GetCOMPorts)
		})
		r.Route("/config", func(r chi.Router) {
			r.Post("/", handler.UpdateConfig)
		})
		r.Route("/dev", func(r chi.Router) {
			r.Get("/tmpl/reload", handler.ReloadTemplate)
		})
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.IP, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      cors.Default().Handler(r), // TODO: Dev only
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt, os.Kill)
	go func() {
		logrus.Infof("main: web server is serving at %s", addr)
		log.Fatal(server.ListenAndServe())
		//logrus.Error(server.ListenAndServeTLS(tlsCrt, tlsKey))
	}()

	// Graceful shutdown
	<-stopChan
	logrus.Info("main: termination signal received. Exiting")
	handler.Stop()
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	server.Shutdown(ctx)
	logrus.Info("main: have a nice day, goodbye!")
}

func overrideConfigs(cfg *config.Config) {
	if *fAddr != "" {
		cfg.Server.IP = *fAddr
	}
	if *fPort != 12345 && *fPort > 0 && *fPort < 65536 {
		cfg.Server.Port = *fPort
	}
	if *fLogLevel != "" {
		cfg.Server.Log = *fLogLevel
	}
	if *fUsername != "" {
		cfg.Admin.Username = *fUsername
	}
	if *fPassword != "" {
		cfg.Admin.Password = *fPassword
		// TODO: Encrypt user password
	}
	if *fSerialPort != "" {
		cfg.Serial.Port = *fSerialPort
	}
	if *fSerialBaud > 0 && *fSerialBaud != 9600 {
		cfg.Serial.Baud = *fSerialBaud
	}

	lvl, err := logrus.ParseLevel(cfg.Server.Log)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(lvl)
	logrus.Infof("main: log level has been set to: %s", lvl)
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	//if strings.ContainsAny(path, ":*") {
	//	panic("FileServer does not permit URL parameters.")
	//}
	fs := http.StripPrefix(path, http.FileServer(root))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}
