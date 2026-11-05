package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/download"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/semver"
	"github.com/DasDarki/nwm/internal/ui"
)

const checksumsAsset = "checksums.txt"

const checkInterval = 24 * time.Hour

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type Release struct {
	Tag    string  `json:"tag_name"`
	URL    string  `json:"html_url"`
	Assets []Asset `json:"assets"`
}

type checkCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

func Latest(ctx context.Context) (*Release, error) {
	url := "https://api.github.com/repos/" + buildinfo.Repository + "/releases/latest"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "nwm/"+buildinfo.Version)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no release of %s published yet", buildinfo.Repository)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching latest release: %s", response.Status)
	}
	var release Release
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return nil, err
	}
	saveCache(checkCache{CheckedAt: time.Now(), Latest: release.Tag})
	return &release, nil
}

func AssetName() string {
	name := fmt.Sprintf("nwm_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func (r *Release) Find(name string) (*Asset, bool) {
	for index := range r.Assets {
		if r.Assets[index].Name == name {
			return &r.Assets[index], true
		}
	}
	return nil, false
}

func Apply(ctx context.Context, release *Release, report ui.Reporter) error {
	asset, found := release.Find(AssetName())
	if !found {
		return fmt.Errorf("release %s has no build for %s/%s", release.Tag, runtime.GOOS, runtime.GOARCH)
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.EvalSymlinks(current)
	if err != nil {
		return err
	}
	staged := current + ".new"
	if err := download.File(ctx, asset.URL, staged, ui.DownloadProgress(report, "downloading "+release.Tag)); err != nil {
		return err
	}
	if err := verify(ctx, release, staged); err != nil {
		os.Remove(staged)
		return err
	}
	if err := os.Chmod(staged, 0o755); err != nil {
		os.Remove(staged)
		return err
	}
	report.Progress(-1)
	report.Status("replacing executable")
	previous := current + ".old"
	os.Remove(previous)
	if err := os.Rename(current, previous); err != nil {
		os.Remove(staged)
		return fmt.Errorf("moving current executable aside: %w", err)
	}
	if err := os.Rename(staged, current); err != nil {
		os.Rename(previous, current)
		return fmt.Errorf("installing new executable: %w", err)
	}
	if runtime.GOOS != "windows" {
		os.Remove(previous)
	}
	return nil
}

func verify(ctx context.Context, release *Release, file string) error {
	checksums, found := release.Find(checksumsAsset)
	if !found {
		return errors.New("release has no checksums.txt, refusing to install an unverified binary")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, checksums.URL, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	expected := ""
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == AssetName() {
			expected = strings.ToLower(fields[0])
		}
	}
	if expected == "" {
		return fmt.Errorf("checksums.txt has no entry for %s", AssetName())
	}
	opened, err := os.Open(file)
	if err != nil {
		return err
	}
	defer opened.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, opened); err != nil {
		return err
	}
	if actual := hex.EncodeToString(hash.Sum(nil)); actual != expected {
		return fmt.Errorf("checksum mismatch for %s", AssetName())
	}
	return nil
}

func CleanupPrevious() {
	current, err := os.Executable()
	if err != nil {
		return
	}
	os.Remove(current + ".old")
}

func PendingNotice(ctx context.Context) string {
	if buildinfo.IsDevelopment() || os.Getenv("NWM_NO_UPDATE_CHECK") != "" {
		return ""
	}
	cache := loadCache()
	latest := cache.Latest
	if time.Since(cache.CheckedAt) > checkInterval {
		timeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		release, err := Latest(timeout)
		if err != nil {
			saveCache(checkCache{CheckedAt: time.Now(), Latest: cache.Latest})
			return ""
		}
		latest = release.Tag
	}
	if latest != "" && IsNewer(latest, buildinfo.Version) {
		return latest
	}
	return ""
}

func IsNewer(candidate string, current string) bool {
	candidateVersion, candidateOk := semver.Parse(candidate)
	currentVersion, currentOk := semver.Parse(current)
	return candidateOk && currentOk && semver.Compare(candidateVersion, currentVersion) > 0
}

func cacheFile() string {
	return filepath.Join(home.CacheDir(), "update-check.json")
}

func loadCache() checkCache {
	var cache checkCache
	content, err := os.ReadFile(cacheFile())
	if err == nil {
		json.Unmarshal(content, &cache)
	}
	return cache
}

func saveCache(cache checkCache) {
	if _, err := home.Ensure(home.CacheDir()); err != nil {
		return
	}
	content, err := json.Marshal(cache)
	if err == nil {
		os.WriteFile(cacheFile(), content, 0o644)
	}
}
