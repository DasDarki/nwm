package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/home"
	"github.com/DasDarki/nwm/internal/nanossharp"
	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/secrets"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/steamcmd"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "server",
		Aliases: []string{"s", "servers", "srv"},
		Short:   "🖥️  Create, update and run nanos world servers",
	}
	command.AddCommand(
		newServerCreateCommand(),
		newServerImportCommand(),
		newServerUpdateCommand(),
		newServerListCommand(),
		newServerInfoCommand(),
		newServerStartCommand(),
		newServerStopCommand(),
		newServerRestartCommand(),
		newServerLogsCommand(),
		newServerRemoveCommand(),
		newServerConfigCommand(),
		newServerTrustCommand(),
	)
	return command
}

type createOptions struct {
	directory   string
	branch      string
	force       bool
	withSharp   bool
	displayName string
	port        int
	queryPort   int
	maxPlayers  int
	gameMode    string
	mapName     string
	recipe      string
	recipeFlags recipeFlags
}

func newServerCreateCommand() *cobra.Command {
	options := createOptions{}
	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Install a new nanos world server via SteamCMD",
		Example: "  nwm server create dev\n" +
			"  nwm server create staging --branch bleeding-edge --port 7787 --nanossharp\n" +
			"  nwm server create rp --recipe @dasdarki/roleplay --var port=7790",
		Args: cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			return createServer(command, arguments[0], options)
		},
	}
	flags := command.Flags()
	flags.StringVarP(&options.directory, "dir", "d", "", "installation directory (default: ~/.nwm/servers/<name>)")
	flags.StringVarP(&options.branch, "branch", "b", steamcmd.BranchPublic, "steam branch: public or bleeding-edge")
	flags.BoolVar(&options.force, "force", false, "install into a non-empty directory")
	flags.BoolVar(&options.withSharp, "nanossharp", false, "install NanosSharp right after the server")
	flags.StringVar(&options.displayName, "display-name", "", "server name shown in the server list")
	flags.IntVar(&options.port, "port", 0, "game port")
	flags.IntVar(&options.queryPort, "query-port", 0, "query port")
	flags.IntVar(&options.maxPlayers, "max-players", 0, "maximum players")
	flags.StringVar(&options.gameMode, "game-mode", "", "game mode package")
	flags.StringVar(&options.mapName, "map", "", "map package")
	flags.StringVarP(&options.recipe, "recipe", "r", "", "set the server up from a recipe (registry name, github:owner/repo, git URL, gist:, file URL or path)")
	options.recipeFlags.register(command)
	command.RegisterFlagCompletionFunc("branch", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{steamcmd.BranchPublic, steamcmd.BranchBleedingEdge}, cobra.ShellCompDirectiveNoFileComp
	})
	return command
}

func createServer(command *cobra.Command, name string, options createOptions) error {
	if !serverNamePattern.MatchString(name) {
		return fmt.Errorf("invalid server name %q, use letters, digits, '-' and '_'", name)
	}
	if err := validateBranch(options.branch); err != nil {
		return err
	}
	current, err := state.Load()
	if err != nil {
		return err
	}
	if _, exists := current.Find(name); exists {
		return fmt.Errorf("a server named %q already exists", name)
	}
	directory := options.directory
	if directory == "" {
		directory = filepath.Join(home.ServersDir(), name)
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return err
	}
	if !options.force && !directoryEmpty(directory) {
		return fmt.Errorf("%s is not empty, use --force or %s", directory, ui.Command("nwm server import"))
	}
	variables, err := options.recipeFlags.parsedVariables()
	if err != nil {
		return err
	}
	var fetched *recipe.Fetched
	var prepared *preparedRecipe
	target := recipe.Target{Name: name, Dir: directory}
	if options.recipe != "" {
		if fetched, err = fetchRecipe(options.recipe, options.recipeFlags.refresh); err != nil {
			return err
		}
		if prepared, err = prepareRecipe(fetched, target, options.recipeFlags, variables); err != nil {
			return err
		}
		if !command.Flags().Changed("branch") && prepared.manifest.Server.Branch != "" {
			options.branch = prepared.manifest.Server.Branch
		}
		if err := validateBranch(options.branch); err != nil {
			return err
		}
	}

	if err := installServer(directory, options.branch, true); err != nil {
		return err
	}

	now := time.Now()
	if err := current.Add(state.Server{Name: name, Path: directory, Branch: options.branch, CreatedAt: now, UpdatedAt: now}); err != nil {
		return err
	}
	if err := current.Save(); err != nil {
		return err
	}

	if fetched != nil {
		if err := applyPrepared(fetched, prepared, target, options.recipeFlags); err != nil {
			ui.Hint("the server is installed, fix the problem and run %s", ui.Command("nwm recipe apply "+options.recipe+" -s "+name))
			return err
		}
	}

	config, err := server.LoadConfig(directory)
	if err != nil {
		return err
	}
	overrides := map[string]any{}
	if options.displayName != "" {
		overrides["discover.name"] = options.displayName
	}
	if command.Flags().Changed("port") {
		overrides["discover.port"] = int64(options.port)
	}
	if command.Flags().Changed("query-port") {
		overrides["discover.query_port"] = int64(options.queryPort)
	}
	if command.Flags().Changed("max-players") {
		overrides["general.max_players"] = int64(options.maxPlayers)
	}
	if options.gameMode != "" {
		overrides["game.game_mode"] = options.gameMode
	}
	if options.mapName != "" {
		overrides["game.map"] = options.mapName
	}
	if len(overrides) > 0 {
		for key, value := range overrides {
			if err := config.Set(key, value); err != nil {
				return err
			}
		}
		if err := config.Save(); err != nil {
			return err
		}
	}

	if options.withSharp && !nanossharp.Installed(directory) {
		if err := ui.Run("Installing NanosSharp", func(ctx context.Context, report ui.Reporter) error {
			return nanossharp.Install(ctx, directory, report)
		}); err != nil {
			return err
		}
	}

	fmt.Fprintln(os.Stderr)
	ui.Step("🎉", "Server %s is ready", ui.Accent.Render(name))
	ui.Detail("path", directory)
	ui.Detail("branch", options.branch)
	ui.Hint("start it with %s", ui.Command("nwm server start "+name))
	return nil
}

func installServer(directory string, branch string, validate bool) error {
	var executable string
	if err := ui.Run("Preparing SteamCMD", func(ctx context.Context, report ui.Reporter) error {
		located, err := steamcmd.Ensure(ctx, report)
		executable = located
		return err
	}); err != nil {
		return err
	}
	title := fmt.Sprintf("Installing nanos world server (%s)", branch)
	if server.Installed(directory) {
		title = fmt.Sprintf("Updating nanos world server (%s)", branch)
	}
	return ui.Run(title, func(ctx context.Context, report ui.Reporter) error {
		return steamcmd.AppUpdate(ctx, executable, steamcmd.UpdateOptions{Directory: directory, Branch: branch, Validate: validate}, report)
	})
}

func validateBranch(branch string) error {
	if branch != steamcmd.BranchPublic && branch != steamcmd.BranchBleedingEdge {
		return fmt.Errorf("unknown branch %q, use %s or %s", branch, steamcmd.BranchPublic, steamcmd.BranchBleedingEdge)
	}
	return nil
}

func directoryEmpty(directory string) bool {
	entries, err := os.ReadDir(directory)
	return err != nil || len(entries) == 0
}

func newServerImportCommand() *cobra.Command {
	var branch string
	command := &cobra.Command{
		Use:   "import <name> <directory>",
		Short: "Register an existing server installation",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, arguments []string) error {
			name := arguments[0]
			if !serverNamePattern.MatchString(name) {
				return fmt.Errorf("invalid server name %q", name)
			}
			if err := validateBranch(branch); err != nil {
				return err
			}
			directory, err := filepath.Abs(arguments[1])
			if err != nil {
				return err
			}
			if !server.Installed(directory) {
				return fmt.Errorf("no nanos world server found in %s", directory)
			}
			current, err := state.Load()
			if err != nil {
				return err
			}
			now := time.Now()
			if err := current.Add(state.Server{Name: name, Path: directory, Branch: branch, CreatedAt: now, UpdatedAt: now}); err != nil {
				return err
			}
			if err := current.Save(); err != nil {
				return err
			}
			ui.Success("Imported %s from %s", ui.Accent.Render(name), directory)
			return nil
		},
	}
	command.Flags().StringVarP(&branch, "branch", "b", steamcmd.BranchPublic, "steam branch the installation uses")
	return command
}

func newServerUpdateCommand() *cobra.Command {
	var branch string
	var skipValidation bool
	command := &cobra.Command{
		Use:               "update [name]",
		Short:             "Update a server to the latest build of its branch",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(command *cobra.Command, arguments []string) error {
			current, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			if info, running := server.Status(target.Name); running {
				return fmt.Errorf("server %q is running (pid %d), stop it first", target.Name, info.PID)
			}
			if command.Flags().Changed("branch") {
				if err := validateBranch(branch); err != nil {
					return err
				}
				target.Branch = branch
			}
			if target.Branch == "" {
				target.Branch = steamcmd.BranchPublic
			}
			if err := installServer(target.Path, target.Branch, !skipValidation); err != nil {
				return err
			}
			if nanossharp.Installed(target.Path) {
				ui.Hint("NanosSharp is installed, run %s if the server version changed", ui.Command("nwm nanossharp update "+target.Name))
			}
			target.UpdatedAt = time.Now()
			return current.Save()
		},
	}
	command.Flags().StringVarP(&branch, "branch", "b", "", "switch to another steam branch")
	command.Flags().BoolVar(&skipValidation, "no-validate", false, "skip file validation")
	return command
}

func newServerListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List registered servers",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			current, err := state.Load()
			if err != nil {
				return err
			}
			if len(current.Servers) == 0 {
				ui.Hint("no servers yet, create one with %s", ui.Command("nwm server create <name>"))
				return nil
			}
			rows := make([][]string, 0, len(current.Servers))
			for _, entry := range current.Servers {
				sharp := ""
				if nanossharp.Installed(entry.Path) {
					sharp = "✓"
				}
				rows = append(rows, []string{entry.Name, statusLabel(entry.Name), entry.Branch, sharp, entry.Path})
			}
			fmt.Println(renderTable([]string{"NAME", "STATUS", "BRANCH", "C#", "PATH"}, rows))
			return nil
		},
	}
}

func statusLabel(name string) string {
	info, running := server.Status(name)
	if !running {
		return "⚫ stopped"
	}
	return fmt.Sprintf("🟢 running (%d)", info.PID)
}

func renderTable(headers []string, rows [][]string) string {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(ui.Muted).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row int, _ int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true).Foreground(lipgloss.Color("#8B5CF6"))
			}
			return style
		}).
		String()
}

func newServerInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "info [name]",
		Short:             "Show details about a server",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			config, err := server.LoadConfig(target.Path)
			if err != nil {
				return err
			}
			packages, _ := server.ListResources(target.Path, server.PackageKind)
			assets, _ := server.ListResources(target.Path, server.AssetKind)
			ui.Step("🖥️ ", "%s", ui.Accent.Render(target.Name))
			ui.Detail("status", statusLabel(target.Name))
			ui.Detail("path", target.Path)
			ui.Detail("branch", target.Branch)
			ui.Detail("nanossharp", strconv.FormatBool(nanossharp.Installed(target.Path)))
			ui.Detail("updated", target.UpdatedAt.Format(time.DateTime))
			if config.Exists() {
				ui.Detail("display name", config.String("discover.name"))
				ui.Detail("port", config.String("discover.port"))
				ui.Detail("query port", config.String("discover.query_port"))
				ui.Detail("game mode", config.String("game.game_mode"))
				ui.Detail("map", config.String("game.map"))
			}
			ui.Detail("packages", strconv.Itoa(len(packages))+" installed")
			ui.Detail("asset packs", strconv.Itoa(len(assets))+" installed")
			ui.Detail("log", server.LogFile(target.Name))
			return nil
		},
	}
}

func newServerStartCommand() *cobra.Command {
	var detach bool
	var skipServices bool
	var allowUnsafe bool
	command := &cobra.Command{
		Use:   "start [name] [-- server arguments]",
		Short: "Start a server in the foreground or in the background",
		Example: "  nwm server start dev\n" +
			"  nwm server start dev --detach\n" +
			"  nwm server start dev -- --log_level 2",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeServerNames,
		RunE: func(command *cobra.Command, arguments []string) error {
			name, extra := splitServerArguments(command, arguments)
			current, target, err := resolveServer(name)
			if err != nil {
				return err
			}
			return startServer(current, target, extra, startOptions{detach: detach, skipServices: skipServices, allowUnsafe: allowUnsafe})
		},
	}
	command.Flags().BoolVarP(&detach, "detach", "d", false, "run in the background and write output to the log file")
	command.Flags().BoolVar(&skipServices, "no-services", false, "do not start the server's Docker services")
	command.Flags().BoolVar(&allowUnsafe, "allow-unsafe-libs", false, "approve "+server.UnsafeLibsFlag+" from the recipe without asking")
	return command
}

func splitServerArguments(command *cobra.Command, arguments []string) (string, []string) {
	dash := command.ArgsLenAtDash()
	if dash < 0 {
		return optionalServerName(arguments), nil
	}
	return optionalServerName(arguments[:dash]), arguments[dash:]
}

func warnBlockedCModules(target *state.Server, extra []string) {
	arguments := extra
	if local, err := server.LoadLocal(target.Path); err == nil {
		arguments = append(append([]string{}, local.Args...), extra...)
	}
	blocked := server.BlockedCModules(target.Path, arguments)
	if len(blocked) == 0 {
		return
	}
	ui.Warn("C modules (%s) only load with %s, which allows native code and unsafe Lua libraries", strings.Join(blocked, ", "), server.UnsafeLibsFlag)
	ui.Hint("if you trust them, add %s to the recipe (nwm asks once on start) or run %s", ui.Command("arg "+server.UnsafeLibsFlag), ui.Command("nwm s start "+target.Name+" -- "+server.UnsafeLibsFlag))
}

type startOptions struct {
	detach       bool
	skipServices bool
	allowUnsafe  bool
}

func startServer(current *state.State, target *state.Server, extra []string, options startOptions) error {
	detach := options.detach
	local, err := server.LoadLocal(target.Path)
	if err != nil {
		return err
	}
	recipeArguments, err := guardUnsafeLibs(current, target, local.Args, extra, options.allowUnsafe)
	if err != nil {
		return err
	}
	if !options.skipServices {
		if err := upServices(target, local.Services); err != nil {
			return err
		}
	}
	arguments := append(append([]string{}, recipeArguments...), extra...)
	warnBlockedCModules(target, arguments)
	environment, err := server.EnvList(target.Path)
	if err != nil {
		return err
	}
	if detach {
		var info *server.RunInfo
		err := ui.Run("Starting "+target.Name, func(context.Context, ui.Reporter) error {
			started, err := server.StartDetached(target.Name, target.Path, arguments, environment)
			info = started
			return err
		})
		if err != nil {
			return err
		}
		ui.Detail("pid", strconv.Itoa(info.PID))
		ui.Hint("follow the output with %s", ui.Command("nwm server logs "+target.Name+" -f"))
		return nil
	}
	ui.Step("🚀", "Starting %s %s", ui.Accent.Render(target.Name), ui.Muted.Render("(Ctrl+C to stop)"))
	err = server.RunForeground(target.Name, target.Path, arguments, environment)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		ui.Warn("server exited with code %d", exitErr.ExitCode())
		return exitCodeError{code: exitErr.ExitCode()}
	}
	if err == nil {
		ui.Step("🛑", "Server %s stopped", target.Name)
	}
	return err
}

func newServerStopCommand() *cobra.Command {
	var timeout time.Duration
	var withServices bool
	command := &cobra.Command{
		Use:               "stop [name]",
		Short:             "Stop a running server",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			if _, running := server.Status(target.Name); running || !withServices {
				if err := ui.Run("Stopping "+target.Name, func(context.Context, ui.Reporter) error {
					return server.Stop(target.Name, timeout)
				}); err != nil {
					return err
				}
			}
			if !withServices {
				return nil
			}
			local, err := server.LoadLocal(target.Path)
			if err != nil {
				return err
			}
			return downServices(target, local.Services)
		},
	}
	command.Flags().DurationVar(&timeout, "timeout", 15*time.Second, "time to wait before killing the server")
	command.Flags().BoolVar(&withServices, "services", false, "also stop the server's Docker services")
	return command
}

func newServerRestartCommand() *cobra.Command {
	var timeout time.Duration
	var allowUnsafe bool
	command := &cobra.Command{
		Use:               "restart [name] [-- server arguments]",
		Short:             "Restart a server in the background",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeServerNames,
		RunE: func(command *cobra.Command, arguments []string) error {
			name, extra := splitServerArguments(command, arguments)
			current, target, err := resolveServer(name)
			if err != nil {
				return err
			}
			if _, running := server.Status(target.Name); running {
				if err := ui.Run("Stopping "+target.Name, func(context.Context, ui.Reporter) error {
					return server.Stop(target.Name, timeout)
				}); err != nil {
					return err
				}
			}
			return startServer(current, target, extra, startOptions{detach: true, allowUnsafe: allowUnsafe})
		},
	}
	command.Flags().DurationVar(&timeout, "timeout", 15*time.Second, "time to wait before killing the server")
	command.Flags().BoolVar(&allowUnsafe, "allow-unsafe-libs", false, "approve "+server.UnsafeLibsFlag+" from the recipe without asking")
	return command
}

func newServerRemoveCommand() *cobra.Command {
	var deleteFiles bool
	command := &cobra.Command{
		Use:               "remove <name>",
		Aliases:           []string{"rm"},
		Short:             "Unregister a server and optionally delete its files",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			current, target, err := resolveServer(arguments[0])
			if err != nil {
				return err
			}
			if _, running := server.Status(target.Name); running {
				return fmt.Errorf("server %q is running, stop it first", target.Name)
			}
			name, path := target.Name, target.Path
			if local, err := server.LoadLocal(path); err == nil && len(local.Services) > 0 {
				if err := removeContainers(target, local.Services, deleteFiles); err != nil {
					ui.Warn("could not remove service containers: %v", err)
				}
			}
			if deleteFiles {
				if err := secrets.Remove(name); err != nil {
					ui.Warn("could not delete secrets: %v", err)
				}
				if err := ui.Run("Deleting "+path, func(context.Context, ui.Reporter) error {
					return os.RemoveAll(path)
				}); err != nil {
					return err
				}
			}
			current.Remove(name)
			if err := current.Save(); err != nil {
				return err
			}
			os.Remove(server.LogFile(name))
			ui.Success("Removed %s", ui.Accent.Render(name))
			if !deleteFiles {
				ui.Hint("files were kept in %s", path)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&deleteFiles, "delete-files", false, "also delete the installation directory")
	return command
}
