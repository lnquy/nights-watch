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
		Server *Server `json:"server"`
		Admin *Admin `json:"admin"`
		Serial *Serial `json:"serial"`
	}

	Server struct {
		IP string `json:"ip"`
		Port string `json:"port" default:"12345"`
		Log string `json:"log" default:"info"`
	}

	Admin struct {
		Username string `json:"username" default:"admin"`
		Password string `json:"password" default:"admin"`
	}

	Serial struct {
		Port string `json:"port"`
		Baud uint `json:"baud"`
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
	cfg := Config{}
	if json.Unmarshal(b, &cfg) != nil {
		logrus.Fatalf("failed to parse config file: %s", err)
	}
	return &cfg
}

func (cfg *Config) WriteToFile(fp string) (string, error) {
	if fp == "" {
		fp = path.Join(util.GetWd(), "nights_watch.conf") // Default path
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	if util.WriteToFile(fp, b) != nil {
		return "", err
	}
	return fp, nil
}
