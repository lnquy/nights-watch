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
		StatsConf `json:"stats_conf"`
		CPU       `json:"cpu"`
		Memory    `json:"memory"`
		GPU       `json:"gpu"`
		Network   `json:"network"`
	}

	StatsConf struct {
		Interval       uint   `json:"interval"`
		BeginSleepTime uint64 `json:"begin_sleep_time"`
		EndSleepTime   uint64 `json:"end_sleep_time"`
	}

	CPU struct {
		LoadThreshold float64 `json:"load_threshold"`
		TempThreshold float64 `json:"temp_threshold"`
	}

	Memory struct {
		LoadThreshold float64 `json:"load_threshold"`
	}

	GPU struct {
		LoadThreshold float64 `json:"load_threshold"`
		MemThreshold  float64 `json:"mem_threshold"`
	}

	Network struct {
		DownloadThreshold float64 `json:"download_threshold"`
		UploadThreshold   float64 `json:"upload_threshold"`
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
		Stats: Stats{
			StatsConf: StatsConf{
				Interval: 3,
			},
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
