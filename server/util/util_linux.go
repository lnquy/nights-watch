package util

import (
	"os"
	"regexp"
	"strings"
	"io/ioutil"
	"github.com/tarm/serial"
)

const (
	devFolder = "/dev"
	regexFilter = "(ttyS|ttyUSB|ttyACM|ttyAMA|rfcomm|ttyO)[0-9]{1,3}"
)

func GetHome() string {
	return os.Getenv("HOME")
}

func LineBreak() string {
	return "\n"
}

// Taken from https://github.com/bugst/go-serial with modifications
func nativeGetPorts() ([]string, error) {
	files, err := ioutil.ReadDir(devFolder)
	if err != nil {
		return nil, err
	}

	ports := make([]string, 0, len(files))
	for _, f := range files {
		// Skip folders
		if f.IsDir() {
			continue
		}

		// Keep only devices with the correct name
		match, err := regexp.MatchString(regexFilter, f.Name())
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}

		portName := devFolder + "/" + f.Name()

		// Check if serial port is real or is a placeholder serial port "ttySxx"
		if strings.HasPrefix(f.Name(), "ttyS") {
			if port, err := nativeOpen(portName); err != nil {
				continue
			} else {
				port.Close()
			}
		}

		// Save serial port in the resulting list
		ports = append(ports, portName)
	}

	return ports, nil
}

func nativeOpen(portName string) (*serial.Port, error) {
	return serial.OpenPort(&serial.Config{
		Name: portName,
		Baud: 9600,
	})
}
