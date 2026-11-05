package server

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func CliExecutable(directory string) (string, error) {
	if runtime.GOOS == "windows" {
		return existing(directory, "NanosWorldServer.exe")
	}
	return existing(directory, "NanosWorldServer.sh")
}

func LaunchExecutable(directory string) (string, error) {
	if runtime.GOOS != "windows" {
		if launcher, err := existing(directory, "NanosSharp.sh"); err == nil {
			return launcher, nil
		}
	}
	return CliExecutable(directory)
}

func Installed(directory string) bool {
	_, err := CliExecutable(directory)
	return err == nil
}

func existing(directory string, name string) (string, error) {
	path := filepath.Join(directory, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%s not found in %s, is the server installed?", name, directory)
	}
	return path, nil
}
