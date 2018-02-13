package util

import "os"

func GetHome() string {
	return os.Getenv("HOME")
}

func LineBreak() string {
	return "\n"
}
