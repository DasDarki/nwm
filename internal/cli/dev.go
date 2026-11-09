package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/dev"
	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/steamcmd"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

type devOptions struct {
	name        string
	directory   string
	local       bool
	reapply     bool
	allowUnsafe bool
	flags       recipeFlags
}

func newDevCommand() *cobra.Command {
	var options devOptions
	command := &cobra.Command{
		Use:   "dev [directory | file] [-- server arguments]",
		Short: "🛠️  Develop packages locally with live reload",
		Long: "Runs a development server for the recipe in the current folder. Packages from the recipe\n" +
			"folder (path) and local checkouts (local) are linked into the server, builders such as\n" +
			"nebra watch run in the background and every change reloads the package in the server.\n\n" +
			"The server lives in ~/.nwm/servers/<name>-dev, in --dir, or with --local in .nwm/ next\n" +
			"to the recipe. Type console commands, /help lists nwm commands, Ctrl+C stops everything.",
		Example: "  nwm dev\n" +
			"  nwm dev --local\n" +
			"  nwm dev ./staging.Recipefile -d ~/servers/staging\n" +
			"  nwm dev -- --log_level 3",
		Args: cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			location, extra := ".", []string(nil)
			if dash := command.ArgsLenAtDash(); dash >= 0 {
				extra = arguments[dash:]
				arguments = arguments[:dash]
			}
			if len(arguments) > 1 {
				return fmt.Errorf("expected at most one recipe folder or file")
			}
			if len(arguments) == 1 {
				location = arguments[0]
			}
			return runDev(location, extra, options)
		},
	}
	command.Flags().StringVarP(&options.name, "name", "n", "", "name of the development server (default: <recipe>-dev)")
	command.Flags().StringVarP(&options.directory, "dir", "d", "", "folder of the development server")
	command.Flags().BoolVar(&options.local, "local", false, "keep the development server in .nwm/ next to the recipe")
	command.Flags().BoolVar(&options.reapply, "reapply", false, "apply the whole recipe again even if it did not change")
	command.Flags().BoolVar(&options.allowUnsafe, "allow-unsafe-libs", false, "approve "+server.UnsafeLibsFlag+" from the recipe without asking")
	command.Flags().StringArrayVar(&options.flags.variables, "var", nil, "set a recipe variable, e.g. --var port=7790")
	command.Flags().StringArrayVar(&options.flags.secrets, "secret", nil, "set a secret instead of generating or asking")
	command.Flags().BoolVar(&options.flags.skipServices, "no-services", false, "do not start the recipe's services")
	command.Flags().BoolVar(&options.flags.skipHooks, "no-hooks", false, "do not run the recipe's hooks")
	command.MarkFlagDirname("dir")
	return command
}

func runDev(location string, extra []string, options devOptions) error {
	if !ui.StdinInteractive() {
		ui.Hint("no terminal attached, console input and /commands are unavailable")
	}
	document, err := openRecipeDocument(location)
	if err != nil {
		return err
	}
	manifest := document.Manifest.WithDev()
	if err := manifest.Validate(document.Dir); err != nil {
		return err
	}

	current, target, err := devServer(document, manifest, options)
	if err != nil {
		return err
	}
	variables, err := options.flags.parsedVariables()
	if err != nil {
		return err
	}
	source := recipe.Source{Kind: recipe.SourceLocal, Raw: "dev:" + document.Path(), Path: document.Dir, File: filepath.Base(document.Path())}
	fetched := &recipe.Fetched{Source: source, Dir: document.Dir, Manifest: manifest, Version: "dev"}
	recipeTarget := recipe.Target{Name: target.Name, Dir: target.Path}
	prepared, err := prepareRecipe(fetched, recipeTarget, options.flags, variables)
	if err != nil {
		return err
	}
	fetched.Commit = devFingerprint(prepared, variables)

	local, err := server.LoadLocal(target.Path)
	if err != nil {
		return err
	}
	unchanged := !options.reapply && local.Recipe != nil && local.Recipe.Source == source.Raw && local.Recipe.Commit == fetched.Commit
	var mounts []*recipe.Mount
	if unchanged {
		ui.Hint("recipe unchanged since the last run, refreshing local packages only (--reapply applies everything)")
		mounts, err = mountLocalPackages(document.Dir, target.Path, prepared, options.flags)
		if err != nil {
			return err
		}
		if !options.flags.skipServices {
			if err := upServices(target, local.Services); err != nil {
				return err
			}
		}
	} else {
		result, err := applyPreparedWith(fetched, prepared, recipeTarget, options.flags, true)
		if err != nil {
			return err
		}
		mounts = result.Mounts
	}
	if len(mounts) == 0 {
		return dev.ErrNoMounts
	}
	for _, mount := range mounts {
		mode := "linked"
		if !mount.Linked() {
			mode = "synced (links unavailable)"
		}
		ui.Detail(mount.Name, fmt.Sprintf("%s from %s", mode, mount.Plan.Root))
	}

	recipeArguments, err := guardUnsafeLibs(current, target, prepared.manifest.Server.Args, extra, options.allowUnsafe)
	if err != nil {
		return err
	}
	environment, err := server.EnvList(target.Path)
	if err != nil {
		return err
	}
	if info, running := server.Status(target.Name); running {
		return fmt.Errorf("server %q is already running (pid %d), stop it first", target.Name, info.PID)
	}
	modeEnvironment, err := server.ApplyMode(target.Path, server.ModeDev)
	if err != nil {
		return err
	}
	environment = append(environment, modeEnvironment...)
	ui.Step("🛠️ ", "Developing %s on %s %s", ui.Accent.Render(manifest.Recipe.Name), ui.Accent.Render(target.Name), ui.Muted.Render(target.Path))
	return dev.Run(&dev.Session{
		ServerName: target.Name,
		ServerDir:  target.Path,
		Arguments:  append(recipeArguments, extra...),
		Env:        environment,
		Mounts:     mounts,
	})
}

func devServer(document *recipe.Document, manifest *recipe.Manifest, options devOptions) (*state.State, *state.Server, error) {
	if options.local && options.directory != "" {
		return nil, nil, fmt.Errorf("use either --local or --dir")
	}
	name := options.name
	if name == "" {
		name = devServerName(manifest.Recipe.Name, document.Dir)
	}
	if !serverNamePattern.MatchString(name) {
		return nil, nil, fmt.Errorf("invalid server name %q, pass one with --name", name)
	}
	directory := options.directory
	if options.local {
		directory = filepath.Join(document.Dir, ".nwm", "server")
	}
	if directory == "" {
		directory = filepath.Join(home.ServersDir(), name)
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, nil, err
	}
	current, err := state.Load()
	if err != nil {
		return nil, nil, err
	}
	if existing, found := current.Find(name); found {
		if (options.local || options.directory != "") && filepath.Clean(existing.Path) != directory {
			return nil, nil, fmt.Errorf("server %q already lives in %s, pass another --name or remove it", name, existing.Path)
		}
		if server.Installed(existing.Path) {
			return current, existing, nil
		}
		directory = existing.Path
		current.Remove(name)
	}
	if options.local {
		if err := os.MkdirAll(filepath.Dir(directory), 0o755); err != nil {
			return nil, nil, err
		}
		os.WriteFile(filepath.Join(filepath.Dir(directory), ".gitignore"), []byte("*\n"), 0o644)
	}
	branch := manifest.Server.Branch
	if branch == "" {
		branch = steamcmd.BranchPublic
	}
	ui.Step("📦", "Setting up the development server %s", ui.Accent.Render(name))
	if err := installServer(directory, branch, true); err != nil {
		return nil, nil, err
	}
	now := time.Now()
	if err := current.Add(state.Server{Name: name, Path: directory, Branch: branch, CreatedAt: now, UpdatedAt: now}); err != nil {
		return nil, nil, err
	}
	if err := current.Save(); err != nil {
		return nil, nil, err
	}
	created, _ := current.Find(name)
	return current, created, nil
}

func devServerName(recipeName string, directory string) string {
	base := recipeName
	if index := strings.LastIndex(base, "/"); index >= 0 {
		base = base[index+1:]
	}
	if base == "" {
		base = strings.ToLower(filepath.Base(directory))
	}
	cleaned := strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9', character == '-', character == '_':
			return character
		case character >= 'A' && character <= 'Z':
			return character + ('a' - 'A')
		default:
			return '-'
		}
	}, base)
	return strings.Trim(cleaned, "-_") + "-dev"
}

func devFingerprint(prepared *preparedRecipe, variables map[string]string) string {
	hash := sha256.New()
	hash.Write([]byte(recipe.RenderDSL(prepared.manifest)))
	for key, value := range variables {
		hash.Write([]byte(key + "=" + value + "\n"))
	}
	return "dev-" + hex.EncodeToString(hash.Sum(nil))[:16]
}

func mountLocalPackages(recipeDir string, serverDir string, prepared *preparedRecipe, flags recipeFlags) ([]*recipe.Mount, error) {
	var mounts []*recipe.Mount
	options := recipe.ApplyOptions{Dev: true}
	err := ui.Run("Linking local packages", func(ctx context.Context, report ui.Reporter) error {
		for _, group := range []struct {
			kind      server.Kind
			resources []recipe.Resource
		}{{server.PackageKind, localOnly(prepared.manifest.Packages)}, {server.AssetKind, localOnly(prepared.manifest.Assets)}} {
			placed, err := recipe.PlaceResources(ctx, recipeDir, serverDir, group.kind, group.resources, options, report)
			mounts = append(mounts, placed...)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return mounts, err
}

func localOnly(resources []recipe.Resource) []recipe.Resource {
	var local []recipe.Resource
	for _, resource := range resources {
		if resource.Path != "" || resource.Local != "" {
			local = append(local, resource)
		}
	}
	return local
}
