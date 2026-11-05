package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/registry"
	"github.com/DasDarki/nwm/internal/secrets"
	"github.com/DasDarki/nwm/internal/semver"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

type recipeFlags struct {
	variables    []string
	secrets      []string
	skipServices bool
	skipHooks    bool
	assumeYes    bool
	refresh      bool
}

func (f *recipeFlags) register(command *cobra.Command) {
	command.Flags().StringArrayVar(&f.variables, "var", nil, "set a recipe variable, e.g. --var port=7790")
	command.Flags().StringArrayVar(&f.secrets, "secret", nil, "set a secret instead of generating or asking, e.g. --secret api_key=...")
	command.Flags().BoolVar(&f.skipServices, "no-services", false, "do not start the recipe's services")
	command.Flags().BoolVar(&f.skipHooks, "no-hooks", false, "do not run the recipe's hooks")
	command.Flags().BoolVarP(&f.assumeYes, "yes", "y", false, "run hooks of remote recipes without asking")
	command.Flags().BoolVar(&f.refresh, "refresh", false, "refresh the registry index first")
}

func (f *recipeFlags) parsedVariables() (map[string]string, error) {
	return parseAssignments(f.variables, "--var")
}

func parseAssignments(entries []string, flag string) (map[string]string, error) {
	values := map[string]string{}
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if !found || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%s expects name=value, got %q", flag, entry)
		}
		values[strings.TrimSpace(key)] = value
	}
	return values, nil
}

type preparedRecipe struct {
	manifest  *recipe.Manifest
	variables map[string]string
	secrets   map[string]string
}

func prepareRecipe(fetched *recipe.Fetched, target recipe.Target, flags recipeFlags, variables map[string]string) (*preparedRecipe, error) {
	provided, err := parseAssignments(flags.secrets, "--secret")
	if err != nil {
		return nil, err
	}
	store, err := secrets.Open(target.Name)
	if err != nil {
		return nil, err
	}
	var prompt recipe.PromptFunc
	if ui.StdinInteractive() {
		prompt = func(name string) (string, error) {
			return ui.PromptSecret("Value for secret " + name)
		}
	}
	created, err := recipe.EnsureSecrets(fetched.Manifest, store, provided, prompt)
	if err != nil {
		return nil, err
	}
	if err := store.Save(); err != nil {
		return nil, err
	}
	if len(created) > 0 {
		ui.Step("🔑", "Stored %d new secret(s) for %s: %s", len(created), target.Name, strings.Join(created, ", "))
	}
	manifest, err := recipe.Resolve(fetched, target, variables, store.Values())
	if err != nil {
		return nil, err
	}
	return &preparedRecipe{manifest: manifest, variables: variables, secrets: store.Values()}, nil
}

func newRecipeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "recipe",
		Aliases: []string{"recipes", "r"},
		Short:   "📜 Create, share and apply server recipes",
		Long: "Recipes describe a complete server setup: Config.toml, packages, assets, C modules,\n" +
			"files, .env values, Docker services and hooks. They live in git repositories and\n" +
			"can be published to the registry at github.com/" + registry.Repository + ".",
	}
	command.AddCommand(
		newRecipeInitCommand(),
		newRecipeEditCommand(),
		newRecipeAddCommand(),
		newRecipeRemoveCommand(),
		newRecipeShowCommand(),
		newRecipeConvertCommand(),
		newRecipeValidateCommand(),
		newRecipeApplyCommand(),
		newRecipeUpgradeCommand(),
		newRecipeInfoCommand(),
		newRecipeSearchCommand(),
		newRecipePublishCommand(),
		newRecipeUnpublishCommand(),
	)
	return command
}

func fetchRecipe(spec string, refresh bool) (*recipe.Fetched, error) {
	source, err := recipe.ParseSource(spec)
	if err != nil {
		return nil, err
	}
	var fetched *recipe.Fetched
	err = ui.Run("Fetching recipe "+spec, func(ctx context.Context, report ui.Reporter) error {
		result, err := recipe.Fetch(ctx, source, refresh, report)
		fetched = result
		return err
	})
	if err != nil {
		return nil, err
	}
	ui.Detail("recipe", fetched.Manifest.Recipe.Name+" "+ui.Muted.Render(fetched.Version))
	return fetched, nil
}

func applyRecipe(fetched *recipe.Fetched, target recipe.Target, flags recipeFlags, variables map[string]string) error {
	prepared, err := prepareRecipe(fetched, target, flags, variables)
	if err != nil {
		return err
	}
	return applyPrepared(fetched, prepared, target, flags)
}

func applyPrepared(fetched *recipe.Fetched, prepared *preparedRecipe, target recipe.Target, flags recipeFlags) error {
	manifest := prepared.manifest
	if len(manifest.Hooks) > 0 && !flags.skipHooks && fetched.Remote && !flags.assumeYes {
		ui.Warn("this recipe runs commands on your machine:")
		for _, hook := range manifest.Hooks {
			fmt.Fprintln(os.Stderr, "   "+ui.Accent.Render(strings.Join(hook.Run, " ")))
		}
		if !ui.Confirm("Run these hooks?") {
			flags.skipHooks = true
			ui.Hint("hooks skipped")
		}
	}
	return recipe.Apply(fetched, manifest, target, recipe.ApplyOptions{
		Variables:    prepared.variables,
		Secrets:      prepared.secrets,
		SkipServices: flags.skipServices,
		SkipHooks:    flags.skipHooks,
	})
}

func newRecipeInitCommand() *cobra.Command {
	var name string
	var useTOML bool
	command := &cobra.Command{
		Use:   "init [directory]",
		Short: "Create a Recipefile to start from",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			directory := "."
			if len(arguments) > 0 {
				directory = arguments[0]
			}
			directory, err := filepath.Abs(directory)
			if err != nil {
				return err
			}
			if name == "" {
				name = strings.ToLower(filepath.Base(directory))
			}
			format := recipe.FormatDSL
			if useTOML {
				format = recipe.FormatTOML
			}
			path, err := writeNewRecipe(directory, name, format)
			if err != nil {
				return err
			}
			ui.Success("Created %s", path)
			ui.Hint("edit it with %s, try it with %s", ui.Command("nwm recipe edit"), ui.Command("nwm server create test --recipe "+directory))
			return nil
		},
	}
	command.Flags().StringVar(&name, "name", "", "recipe name, e.g. @you/my-recipe")
	command.Flags().BoolVar(&useTOML, "toml", false, "write recipe.toml instead of a Recipefile")
	return command
}

func newRecipeValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [directory]",
		Short: "Check a local recipe for mistakes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			directory := "."
			if len(arguments) > 0 {
				directory = arguments[0]
			}
			document, err := openRecipeDocument(directory)
			if err != nil {
				return err
			}
			manifest := document.Manifest
			if err := manifest.Validate(document.Dir); err != nil {
				return err
			}
			ui.Success("%s is valid %s", manifest.Recipe.Name, ui.Muted.Render("("+filepath.Base(document.Path())+")"))
			for _, line := range manifest.Summary() {
				ui.Detail("contains", line)
			}
			return nil
		},
	}
}

func newRecipeApplyCommand() *cobra.Command {
	var flags recipeFlags
	var serverName string
	command := &cobra.Command{
		Use:   "apply <source>",
		Short: "Apply a recipe to an existing server",
		Example: "  nwm recipe apply @dasdarki/roleplay -s dev\n" +
			"  nwm recipe apply github:me/recipes/rp@^1 -s dev --var port=7790\n" +
			"  nwm recipe apply ./my-recipe -s dev",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			if err := ensureStopped(target); err != nil {
				return err
			}
			variables, err := flags.parsedVariables()
			if err != nil {
				return err
			}
			fetched, err := fetchRecipe(arguments[0], flags.refresh)
			if err != nil {
				return err
			}
			if err := applyRecipe(fetched, recipe.Target{Name: target.Name, Dir: target.Path}, flags, variables); err != nil {
				return err
			}
			ui.Step("🎉", "Applied %s to %s", fetched.Manifest.Recipe.Name, ui.Accent.Render(target.Name))
			return nil
		},
	}
	addServerFlag(command, &serverName)
	flags.register(command)
	return command
}

func newRecipeUpgradeCommand() *cobra.Command {
	var flags recipeFlags
	var serverName string
	command := &cobra.Command{
		Use:   "upgrade",
		Short: "Re-apply the server's recipe, picking up new matching versions",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			if err := ensureStopped(target); err != nil {
				return err
			}
			local, err := server.LoadLocal(target.Path)
			if err != nil {
				return err
			}
			if local.Recipe == nil {
				return fmt.Errorf("%s was not created from a recipe, use %s", target.Name, ui.Command("nwm recipe apply"))
			}
			variables, err := flags.parsedVariables()
			if err != nil {
				return err
			}
			merged := map[string]string{}
			for key, value := range local.Recipe.Variables {
				merged[key] = value
			}
			for key, value := range variables {
				merged[key] = value
			}
			previous := local.Recipe.Version
			fetched, err := fetchRecipe(local.Recipe.Source, true)
			if err != nil {
				return err
			}
			if fetched.Version == previous && fetched.Commit == local.Recipe.Commit && len(variables) == 0 && fetched.Remote {
				ui.Success("%s is already on %s", target.Name, previous)
				return nil
			}
			if err := applyRecipe(fetched, recipe.Target{Name: target.Name, Dir: target.Path}, flags, merged); err != nil {
				return err
			}
			ui.Step("🎉", "%s: %s %s -> %s", target.Name, fetched.Manifest.Recipe.Name, previous, fetched.Version)
			return nil
		},
	}
	addServerFlag(command, &serverName)
	flags.register(command)
	return command
}

func newRecipeInfoCommand() *cobra.Command {
	var refresh bool
	command := &cobra.Command{
		Use:   "info <source>",
		Short: "Show what a recipe contains and which versions exist",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			fetched, err := fetchRecipe(arguments[0], refresh)
			if err != nil {
				return err
			}
			manifest := fetched.Manifest
			ui.Step("📜", "%s", ui.Accent.Render(manifest.Recipe.Name))
			ui.Detail("description", manifest.Recipe.Description)
			ui.Detail("authors", strings.Join(manifest.Recipe.Authors, ", "))
			ui.Detail("branch", manifest.Server.Branch)
			ui.Detail("nanossharp", fmt.Sprint(manifest.Server.NanosSharp))
			for _, line := range manifest.Summary() {
				ui.Detail("contains", line)
			}
			for name, value := range manifest.Variables {
				ui.Detail("variable", fmt.Sprintf("%s = %s", name, server.FormatValue(value)))
			}
			if fetched.Remote {
				var tags []string
				if err := ui.Run("Listing versions", func(ctx context.Context, _ ui.Reporter) error {
					found, err := recipe.RemoteTags(ctx, fetched.Source.URL)
					tags = found
					return err
				}); err == nil {
					versions := semver.SortDescending(tags)
					if len(versions) > 10 {
						versions = versions[:10]
					}
					ui.Detail("versions", strings.Join(versions, ", "))
				}
			}
			return nil
		},
	}
	command.Flags().BoolVar(&refresh, "refresh", false, "refresh the registry index first")
	return command
}

func newRecipeSearchCommand() *cobra.Command {
	var refresh bool
	command := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the recipe registry",
		Args:  cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, arguments []string) error {
			var index *registry.Index
			if err := ui.Run("Loading registry", func(ctx context.Context, _ ui.Reporter) error {
				loaded, err := registry.Load(ctx, refresh)
				index = loaded
				return err
			}); err != nil {
				return err
			}
			results := index.Search(strings.Join(arguments, " "))
			if len(results) == 0 {
				ui.Hint("no recipes found")
				return nil
			}
			rows := make([][]string, 0, len(results))
			for _, entry := range results {
				rows = append(rows, []string{entry.Name, entry.Description, strings.Join(entry.Keywords, ", ")})
			}
			fmt.Println(renderTable([]string{"NAME", "DESCRIPTION", "KEYWORDS"}, rows))
			return nil
		},
	}
	command.Flags().BoolVar(&refresh, "refresh", false, "refresh the registry index first")
	return command
}

func newRecipePublishCommand() *cobra.Command {
	var browser bool
	command := &cobra.Command{
		Use:   "publish [directory]",
		Short: "Publish a recipe from a GitHub repository to the registry",
		Long: "Publishing opens an issue in " + registry.Repository + ". A workflow checks that the\n" +
			"repository contains the recipe and that you own its scope, then adds it to the index.\n" +
			"Versions are git tags like v1.2.0, so tag and push releases of your recipe.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			directory := "."
			if len(arguments) > 0 {
				directory = arguments[0]
			}
			request, err := buildPublishRequest(directory)
			if err != nil {
				return err
			}
			ui.Step("📜", "Publishing %s", ui.Accent.Render(request.Name))
			ui.Detail("repository", request.Repository)
			ui.Detail("subdirectory", request.Subdir)
			return submitIssue(request.Title(), request.Body(), request.IssueURL(), browser)
		},
	}
	command.Flags().BoolVar(&browser, "browser", false, "open the issue form in the browser instead of using the gh CLI")
	return command
}

func newRecipeUnpublishCommand() *cobra.Command {
	var browser bool
	command := &cobra.Command{
		Use:   "unpublish <name>",
		Short: "Remove a recipe you own from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			name := strings.ToLower(arguments[0])
			if !registry.NamePattern.MatchString(name) {
				return fmt.Errorf("%q is not a scoped recipe name like @you/name", name)
			}
			return submitIssue(registry.UnpublishTitle+name, registry.UnpublishBody(name), registry.UnpublishURL(name), browser)
		},
	}
	command.Flags().BoolVar(&browser, "browser", false, "open the issue form in the browser instead of using the gh CLI")
	return command
}

func buildPublishRequest(directory string) (registry.PublishRequest, error) {
	var request registry.PublishRequest
	directory, err := filepath.Abs(directory)
	if err != nil {
		return request, err
	}
	manifest, err := recipe.Load(directory)
	if err != nil {
		return request, err
	}
	if err := manifest.Validate(directory); err != nil {
		return request, err
	}
	if !registry.NamePattern.MatchString(manifest.Recipe.Name) {
		return request, fmt.Errorf("published recipes need a scoped name like @your-github-name/%s", manifest.Recipe.Name)
	}
	root, err := gitOutput(directory, "rev-parse", "--show-toplevel")
	if err != nil {
		return request, fmt.Errorf("%s is not inside a git repository", directory)
	}
	remote, err := gitOutput(directory, "remote", "get-url", "origin")
	if err != nil {
		return request, fmt.Errorf("the repository has no origin remote")
	}
	repository, err := githubURL(remote)
	if err != nil {
		return request, err
	}
	subdir, err := filepath.Rel(root, directory)
	if err != nil {
		return request, err
	}
	if subdir == "." {
		subdir = ""
	}
	if status, _ := gitOutput(directory, "status", "--porcelain"); status != "" {
		ui.Warn("you have uncommitted changes, the registry only sees what is pushed")
	}
	var tags []string
	ui.Run("Checking published versions", func(ctx context.Context, _ ui.Reporter) error {
		found, err := recipe.RemoteTags(ctx, repository)
		tags = found
		return err
	})
	if len(semver.SortDescending(tags)) == 0 {
		ui.Warn("no version tags found, users will get the default branch until you push a tag like v0.1.0")
	}
	request = registry.PublishRequest{
		Name:        manifest.Recipe.Name,
		Repository:  repository,
		Subdir:      filepath.ToSlash(subdir),
		Description: manifest.Recipe.Description,
		Keywords:    manifest.Recipe.Keywords,
	}
	return request, nil
}

func submitIssue(title string, body string, url string, forceBrowser bool) error {
	if _, err := exec.LookPath("gh"); err == nil && !forceBrowser {
		if exec.Command("gh", "auth", "status").Run() == nil {
			var created string
			err := ui.Run("Opening registry request", func(context.Context, ui.Reporter) error {
				output, err := exec.Command("gh", "issue", "create", "--repo", registry.Repository, "--title", title, "--body", body).CombinedOutput()
				created = strings.TrimSpace(string(output))
				if err != nil {
					return fmt.Errorf("gh issue create: %s", created)
				}
				return nil
			})
			if err != nil {
				return err
			}
			ui.Success("Request submitted, the registry bot answers on %s", created)
			return nil
		}
	}
	ui.Hint("opening the request form in your browser")
	if err := ui.OpenBrowser(url); err != nil {
		ui.Hint("open this link to submit: %s", url)
		return nil
	}
	ui.Detail("link", url)
	return nil
}

func gitOutput(directory string, arguments ...string) (string, error) {
	output, err := exec.Command("git", append([]string{"-C", directory}, arguments...)...).Output()
	return strings.TrimSpace(string(output)), err
}

func githubURL(remote string) (string, error) {
	cleaned := strings.TrimSuffix(strings.TrimSpace(remote), ".git")
	switch {
	case strings.HasPrefix(cleaned, "git@github.com:"):
		cleaned = "https://github.com/" + strings.TrimPrefix(cleaned, "git@github.com:")
	case strings.HasPrefix(cleaned, "ssh://git@github.com/"):
		cleaned = "https://github.com/" + strings.TrimPrefix(cleaned, "ssh://git@github.com/")
	case strings.HasPrefix(cleaned, "http://github.com/"):
		cleaned = "https://" + strings.TrimPrefix(cleaned, "http://")
	}
	if !strings.HasPrefix(cleaned, "https://github.com/") || strings.Count(strings.TrimPrefix(cleaned, "https://github.com/"), "/") != 1 {
		return "", fmt.Errorf("the registry only accepts public GitHub repositories, origin is %s", remote)
	}
	return cleaned, nil
}
