package recipe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/registry"
	"github.com/DasDarki/nwm/internal/semver"
	"github.com/DasDarki/nwm/internal/ui"
)

type Fetched struct {
	Source   Source
	Dir      string
	Manifest *Manifest
	Version  string
	Commit   string
	Remote   bool
}

func Fetch(ctx context.Context, source Source, refresh bool, report ui.Reporter) (*Fetched, error) {
	if source.Kind == SourceRegistry {
		report.Status("looking up " + source.Name + " in the registry")
		index, err := registry.Load(ctx, refresh)
		if err != nil {
			return nil, err
		}
		entry, err := index.Lookup(source.Name)
		if err != nil && !refresh {
			report.Status("refreshing the registry index")
			if index, refreshErr := registry.Load(ctx, true); refreshErr == nil {
				entry, err = index.Lookup(source.Name)
			}
		}
		if err != nil {
			return nil, err
		}
		source.Name = entry.Name
		source.URL = entry.Repository
		source.Subdir = entry.Subdir
	}
	fetched := &Fetched{Source: source}
	switch source.Kind {
	case SourceLocal:
		fetched.Dir = source.Path
		fetched.Version = "local"
	case SourceURL, SourceGist:
		if err := fetchSingleFile(ctx, fetched, report); err != nil {
			return nil, err
		}
		fetched.Remote = true
	default:
		if err := fetchGit(ctx, fetched, report); err != nil {
			return nil, err
		}
		fetched.Remote = true
	}
	var manifest *Manifest
	var err error
	if fetched.Source.File != "" {
		var document *Document
		if document, err = OpenDocumentFile(filepath.Join(fetched.Dir, fetched.Source.File)); err == nil {
			manifest = document.Manifest
		}
	} else {
		manifest, err = Load(fetched.Dir)
	}
	if err != nil {
		return nil, err
	}
	if fetched.Source.SingleFile() {
		if references := manifest.LocalReferences(); len(references) > 0 {
			return nil, fmt.Errorf("single-file recipes cannot reference local files (%s), publish them as a git repository instead", strings.Join(references, ", "))
		}
	}
	if err := manifest.Validate(fetched.Dir); err != nil {
		return nil, err
	}
	if source.Kind == SourceRegistry && manifest.Recipe.Name != source.Name {
		return nil, fmt.Errorf("registry entry %s points to a recipe named %q", source.Name, manifest.Recipe.Name)
	}
	fetched.Manifest = manifest
	return fetched, nil
}

func fetchGit(ctx context.Context, fetched *Fetched, report ui.Reporter) error {
	if err := requireGit(); err != nil {
		return err
	}
	source := fetched.Source
	report.Status("listing versions of " + source.URL)
	tags, err := RemoteTags(ctx, source.URL)
	if err != nil {
		return err
	}
	reference, err := chooseReference(source.Version, tags)
	if err != nil {
		return err
	}
	fetched.Version = reference
	label := reference
	if label == "" {
		label = "HEAD"
	}
	checkout := filepath.Join(home.CacheDir(), "recipes", source.cacheKey(), sanitize(label))
	isTag := reference != "" && containsTag(tags, reference)
	if !isTag || !fsutil.Exists(filepath.Join(checkout, ".git")) {
		os.RemoveAll(checkout)
		report.Status("cloning " + source.URL + " " + label)
		arguments := []string{"clone", "--depth", "1", "--quiet"}
		if reference != "" {
			arguments = append(arguments, "--branch", reference)
		}
		arguments = append(arguments, source.URL, checkout)
		command := exec.CommandContext(ctx, "git", arguments...)
		command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if err := process.Stream(command, report.Status); err != nil {
			os.RemoveAll(checkout)
			return err
		}
	}
	if commit, err := exec.CommandContext(ctx, "git", "-C", checkout, "rev-parse", "HEAD").Output(); err == nil {
		fetched.Commit = strings.TrimSpace(string(commit))
	}
	if fetched.Version == "" {
		fetched.Version = shortCommit(fetched.Commit)
	}
	directory, err := fsutil.SafeJoin(checkout, source.Subdir)
	if err != nil {
		return err
	}
	fetched.Dir = directory
	return nil
}

func chooseReference(requested string, tags []string) (string, error) {
	constraint, err := semver.ParseConstraint(requested)
	if err != nil {
		return requested, nil
	}
	if best, found := semver.Highest(tags, constraint); found {
		return best, nil
	}
	if requested == "" || requested == "latest" {
		return "", nil
	}
	return "", fmt.Errorf("no version matches %q (available: %s)", requested, strings.Join(semver.SortDescending(tags), ", "))
}

func RemoteTags(ctx context.Context, url string) ([]string, error) {
	command := exec.CommandContext(ctx, "git", "ls-remote", "--tags", "--refs", url)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("cannot reach %s: %s", url, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	var tags []string
	for _, line := range strings.Split(string(output), "\n") {
		_, reference, found := strings.Cut(strings.TrimSpace(line), "refs/tags/")
		if found {
			tags = append(tags, reference)
		}
	}
	return tags, nil
}

func CloneInto(ctx context.Context, url string, reference string, subdir string, target string, report ui.Reporter) error {
	if err := requireGit(); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(filepath.Dir(target), ".nwm-clone-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	arguments := []string{"clone", "--depth", "1", "--quiet"}
	if reference != "" {
		arguments = append(arguments, "--branch", reference)
	}
	command := exec.CommandContext(ctx, "git", append(arguments, url, filepath.Join(staging, "repo"))...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := process.Stream(command, report.Status); err != nil {
		return err
	}
	source, err := fsutil.SafeJoin(filepath.Join(staging, "repo"), subdir)
	if err != nil {
		return err
	}
	prepared := filepath.Join(staging, "prepared")
	if err := fsutil.CopyDir(source, prepared, fsutil.SkipGit); err != nil {
		return err
	}
	return fsutil.ReplaceDir(prepared, target)
}

func requireGit() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is required for remote recipes and git packages, please install it")
	}
	return nil
}

func containsTag(tags []string, reference string) bool {
	for _, tag := range tags {
		if tag == reference {
			return true
		}
	}
	return false
}

func sanitize(text string) string {
	return strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(text)
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
