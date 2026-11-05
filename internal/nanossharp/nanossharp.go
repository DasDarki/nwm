package nanossharp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/DasDarki/nwm/internal/download"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/ui"
)

const releaseBase = "https://github.com/DasDarki/NanosSharp/releases/latest/download/"

func Installed(serverDirectory string) bool {
	marker := "NanosSharp.sh"
	if runtime.GOOS == "windows" {
		marker = "version.dll"
	}
	_, markerErr := os.Stat(filepath.Join(serverDirectory, marker))
	_, folderErr := os.Stat(filepath.Join(serverDirectory, "NanosSharp"))
	return markerErr == nil && folderErr == nil
}

func Install(ctx context.Context, serverDirectory string, report ui.Reporter) error {
	return runInstaller(ctx, serverDirectory, nil, report)
}

func Uninstall(ctx context.Context, serverDirectory string, report ui.Reporter) error {
	return runInstaller(ctx, serverDirectory, []string{"--uninstall"}, report)
}

func runInstaller(ctx context.Context, serverDirectory string, extra []string, report ui.Reporter) error {
	installer, err := fetchInstaller(ctx, report)
	if err != nil {
		return err
	}
	report.Progress(-1)
	arguments := append([]string{"server", "--yes"}, extra...)
	command := exec.CommandContext(ctx, installer, arguments...)
	command.Dir = serverDirectory
	return process.Stream(command, report.Status)
}

func fetchInstaller(ctx context.Context, report ui.Reporter) (string, error) {
	asset, err := installerAsset()
	if err != nil {
		return "", err
	}
	target := filepath.Join(home.CacheDir(), "nanossharp", asset)
	if err := download.File(ctx, releaseBase+asset, target, ui.DownloadProgress(report, "downloading NanosSharp installer")); err != nil {
		return "", err
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

func installerAsset() (string, error) {
	switch {
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "nanossharp-install-linux-x64", nil
	case runtime.GOOS == "windows" && runtime.GOARCH == "amd64":
		return "nanossharp-install-win-x64.exe", nil
	default:
		return "", fmt.Errorf("NanosSharp is not available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}
