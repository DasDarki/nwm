package steamcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/DasDarki/nwm/internal/download"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/ui"
)

const AppID = "1936830"

const (
	BranchPublic       = "public"
	BranchBleedingEdge = "bleeding-edge"
)

var progressPattern = regexp.MustCompile(`progress:\s*([0-9]+(?:\.[0-9]+)?)`)

type UpdateOptions struct {
	Directory string
	Branch    string
	Validate  bool
}

func Locate() (string, bool) {
	managed := managedExecutable()
	if _, err := os.Stat(managed); err == nil {
		return managed, true
	}
	if runtime.GOOS != "windows" {
		if system, err := exec.LookPath("steamcmd"); err == nil {
			return system, true
		}
	}
	return "", false
}

func Ensure(ctx context.Context, report ui.Reporter) (string, error) {
	if executable, found := Locate(); found {
		return executable, nil
	}
	url, archiveName, err := archiveSource()
	if err != nil {
		return "", err
	}
	directory, err := home.Ensure(home.SteamCMDDir())
	if err != nil {
		return "", err
	}
	archive := filepath.Join(home.CacheDir(), archiveName)
	if err := download.File(ctx, url, archive, ui.DownloadProgress(report, "downloading SteamCMD")); err != nil {
		return "", err
	}
	defer os.Remove(archive)
	report.Progress(-1)
	report.Status("extracting SteamCMD")
	if strings.HasSuffix(archiveName, ".zip") {
		err = download.ExtractZip(archive, directory)
	} else {
		err = download.ExtractTarGz(archive, directory)
	}
	if err != nil {
		return "", fmt.Errorf("extracting SteamCMD: %w", err)
	}
	return managedExecutable(), nil
}

func AppUpdate(ctx context.Context, executable string, options UpdateOptions, report ui.Reporter) error {
	directory, err := filepath.Abs(options.Directory)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	arguments := []string{"+force_install_dir", directory, "+login", "anonymous", "+app_update", AppID}
	if options.Branch != "" {
		arguments = append(arguments, "-beta", options.Branch)
	}
	if options.Validate {
		arguments = append(arguments, "validate")
	}
	arguments = append(arguments, "+quit")

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		installed := false
		command := exec.CommandContext(ctx, executable, arguments...)
		command.Dir = filepath.Dir(executable)
		err := process.Stream(command, func(line string) {
			if strings.Contains(line, "Success! App '"+AppID+"'") {
				installed = true
			}
			if match := progressPattern.FindStringSubmatch(line); match != nil {
				if percent, parseErr := strconv.ParseFloat(match[1], 64); parseErr == nil {
					report.Progress(percent / 100)
				}
			}
			report.Status(line)
		})
		if installed {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = err
		if lastErr == nil {
			lastErr = errors.New("SteamCMD finished without confirming the installation")
		}
		report.Progress(-1)
		report.Status("SteamCMD did not finish cleanly, retrying")
	}
	return lastErr
}

func managedExecutable() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(home.SteamCMDDir(), "steamcmd.exe")
	}
	return filepath.Join(home.SteamCMDDir(), "steamcmd.sh")
}

func archiveSource() (string, string, error) {
	switch runtime.GOOS {
	case "windows":
		return "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip", "steamcmd.zip", nil
	case "linux":
		return "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz", "steamcmd_linux.tar.gz", nil
	default:
		return "", "", fmt.Errorf("nanos world servers are not supported on %s", runtime.GOOS)
	}
}
