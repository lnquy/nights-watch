package config

import (
	"path"
	"io/ioutil"
	"encoding/json"

	"github.com/lnquy/nights-watch/server/util"
	"github.com/sirupsen/logrus"
)

type (
	Config struct {
		Server `json:"server"`
		Admin  `json:"admin"`
		Serial `json:"serial"`
		Stats  `json:"stats"`
		Sleep  `json:"sleep"`
	}

	Server struct {
		IP   string `json:"ip"`
		Port uint   `json:"port"`
		Log  string `json:"log"`
	}

	Admin struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	Serial struct {
		Port string `json:"port"`
		Baud uint   `json:"baud"`
	}

	Stats struct {
		Interval uint `json:"interval"`
		CPU           `json:"cpu"`
		Memory        `json:"memory"`
		GPU           `json:"gpu"`
		Network       `json:"network"`
	}

	Sleep struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}

	CPU struct {
		Enabled       bool `json:"enabled"`
		LoadThreshold uint `json:"load"`
		TempThreshold uint `json:"temp"`
	}

	Memory struct {
		Enabled       bool `json:"enabled"`
		LoadThreshold uint `json:"load"`
	}

	GPU struct {
		Enabled       bool `json:"enabled"`
		LoadThreshold uint `json:"load"`
		MemThreshold  uint `json:"mem"`
	}

	Network struct {
		Enabled           bool `json:"enabled"`
		DownloadThreshold uint `json:"download"`
		UploadThreshold   uint `json:"upload"`
	}
)

func LoadFromFile(fp string) *Config {
	if fp == "" {
		fp = path.Join(util.GetWd(), "nights_watch.conf") // Default path
	}
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		logrus.Fatalf("failed to open config file: %s", err)
	}

	// Default config values
	cfg := Config{
		Server: Server{
			Port: 12345,
			Log:  "info",
		},
		Admin: Admin{
			Username: "admin",
			Password: "admin",
		},
		Serial: Serial{
			Baud: 9600,
		},
		Sleep: Sleep{
			Start: "00:00",
			End:   "00:00",
		},
		Stats: Stats{
			Interval: 1,
		},
	}
	if err = json.Unmarshal(b, &cfg); err != nil {
		logrus.Fatalf("failed to parse config file: %v", err)
	}
	return &cfg
}

func (cfg *Config) WriteToFile(fp string) (string, error) {
	if fp == "" {
		fp = path.Join(util.GetWd(), "nights_watch.conf") // Default path
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if util.WriteToFile(fp, b) != nil {
		return "", err
	}
	logrus.Infof("config: wrote configuration to %s", fp)
	return fp, nil
}
