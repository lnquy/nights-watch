package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

var (
	fSerialPort = flag.String("sp", "", "Serial port to connect to Arduino")
	fSerialBaud = flag.Int("sb", 9600, "Serial port baud speed")
)

func main() {
	flag.Parse()

	c := &serial.Config{Name: *fSerialPort, Baud: *fSerialBaud}
	s, err := serial.OpenPort(c)
	if err != nil {
		logrus.Fatal(err)
	}
	time.Sleep(2 * time.Second) // Sleep since Arduino will restart when new connection connected

	logrus.Infof("Write to Arduino")
	for i := 0; i < 100; i++ {
		_, err = s.Write([]byte(fmt.Sprintf("%d.", i)))
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Wrote: %d", i)
		time.Sleep(1 * time.Second)
	}

	logrus.Infof("Arduino wrote")

	// logrus.Infof("Read from Arduino")
	// buf := make([]byte, 128)
	// n, err = s.Read(buf)
	// if err != nil {
	// 	logrus.Fatal(err)
	// }
	// logrus.Infof("Arduino read")
	// logrus.Printf("%q", buf[:n])
}
