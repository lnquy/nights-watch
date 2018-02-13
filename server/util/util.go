package util

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	
	"github.com/sirupsen/logrus"
)

func GetCOMPorts() ([]string, error) {
	return nativeGetPorts()
}

func GetWd() string {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("failed to get wd: %v", err)
	}
	return wd
}

func EnsureDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0777) // TODO: Permission
	}
}

func EnsureFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fw, err := os.Create(path)
		if err != nil {
			logrus.Errorf("failed to create file: %v", err)
		}
		fw.Close()
	}
}

func WriteToFile(name string, data []byte) error {
	EnsureDir(path.Dir(name))
	return ioutil.WriteFile(name, data, 0666)
}

func Copy(src, des string) error {
	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	EnsureFile(des)
	destFile, err := os.OpenFile(des, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}
	if err := destFile.Sync(); err != nil {
		return err
	}
	return nil
}

func IsDirEmpty(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true
	}
	return false
}

func GetStaticLinks(data []byte, rgx string) []string {
	var r *regexp.Regexp
	var err error
	if r, err = regexp.Compile(rgx); err != nil {
		return []string{}
	}
	return r.FindAllString(string(data), -1)
}

func GetAverage(arr []float64) float64 {
	if len(arr) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range arr {
		sum += v
	}
	return sum / float64(len(arr))
}
