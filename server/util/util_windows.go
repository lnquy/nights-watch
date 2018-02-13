package util

import "os"

func GetHome() string {
	home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	return home
}

func LineBreak() string {
	return "\r\n"
}
