package home

import (
	"os"
	"path/filepath"
)

func Dir() string {
	if custom := os.Getenv("NWM_HOME"); custom != "" {
		return custom
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return ".nwm"
	}
	return filepath.Join(userHome, ".nwm")
}

func SteamCMDDir() string {
	return filepath.Join(Dir(), "steamcmd")
}

func CacheDir() string {
	return filepath.Join(Dir(), "cache")
}

func ServersDir() string {
	return filepath.Join(Dir(), "servers")
}

func RunDir() string {
	return filepath.Join(Dir(), "run")
}

func LogsDir() string {
	return filepath.Join(Dir(), "logs")
}

func StateFile() string {
	return filepath.Join(Dir(), "state.toml")
}

func Ensure(dir string) (string, error) {
	return dir, os.MkdirAll(dir, 0o755)
}
