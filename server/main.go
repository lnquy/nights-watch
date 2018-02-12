package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lnquy/nights-watch/server/watcher/cpu"
	"github.com/lnquy/nights-watch/server/watcher/mem"
	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

var (
	fSerialPort = flag.String("sp", "", "Serial port to connect to Arduino")
	fSerialBaud = flag.Int("sb", 9600, "Serial port baud speed")
	fLogLevel   = flag.String("log", "info", "Log level")
)

func main() {
	flag.Parse()
	lvl, err := logrus.ParseLevel(*fLogLevel)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetLevel(lvl)

	serialPort, err := serial.OpenPort(&serial.Config{
		Name: *fSerialPort,
		Baud: *fSerialBaud,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	time.Sleep(2 * time.Second) // Sleep since Arduino will restart when new connection connected
	// TODO: Write config here

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Second // TODO: Configurable
	logrus.Infof("Start watcher")
	go watchStats(ctx, serialPort, interval)
	// TODO: Cancel context and restart watcher when configuration changed here

	<-exitChan
	cancel()
	logrus.Infof("Server stopped")
}

// First character dertermines the message type:
// 0: Config
// 1: CPU stats
// 2: Memory stats
// 3: GPU stats
// 4: Network stats
func watchStats(ctx context.Context, serialPort *serial.Port, interval time.Duration) {
	cw, cm := cpu.NewWatcher().GetStats(ctx, interval), mem.NewWatcher().GetStats(ctx, interval)
	logrus.Info("Watcher started")
	for {
		select {
		case s := <-cw:
			msg := fmt.Sprintf("1|%.0f|%.0f$", s.Load, s.Temp)
			logrus.Debugf("CPU: %s", msg)
			if _, err := serialPort.Write([]byte(msg)); err != nil {
				logrus.Errorf("Failed to write CPU stats to Arduino: %s", msg)
			}
		case s := <-cm:
			msg := fmt.Sprintf("2|%.0f|%d$", s.Load, s.Usage)
			logrus.Debugf("MEM: %s", msg)
			if _, err := serialPort.Write([]byte(msg)); err != nil {
				logrus.Errorf("Failed to write CPU stats to Arduino: %s", msg)
			}
		}
	}
}
