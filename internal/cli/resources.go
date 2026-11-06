package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DasDarki/nwm/internal/fsutil"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

type resourceCommandSpec struct {
	use     string
	aliases []string
	icon    string
	kind    server.Kind
}

var (
	resourcePackages = resourceCommandSpec{use: "package", aliases: []string{"p", "packages", "pkg"}, icon: "📦", kind: server.PackageKind}
	resourceAssets   = resourceCommandSpec{use: "assets", aliases: []string{"a", "asset"}, icon: "🎨", kind: server.AssetKind}
)

func newResourceCommand(spec resourceCommandSpec) *cobra.Command {
	command := &cobra.Command{
		Use:     spec.use,
		Aliases: spec.aliases,
		Short:   fmt.Sprintf("%s Install, update and list %s from the vault", spec.icon, spec.kind.Plural),
	}
	command.AddCommand(
		newResourceInstallCommand(spec),
		newResourceUpdateCommand(spec),
		newResourceListCommand(spec),
		newResourceRemoveCommand(spec),
		newResourceToggleCommand(spec, true),
		newResourceToggleCommand(spec, false),
	)
	return command
}

func newResourceInstallCommand(spec resourceCommandSpec) *cobra.Command {
	var serverName string
	var enable bool
	command := &cobra.Command{
		Use:     "install <name>...",
		Aliases: []string{"add", "i"},
		Short:   fmt.Sprintf("Install %s and their requirements from the vault", spec.kind.Plural),
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, names []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			title := fmt.Sprintf("Installing %s into %s", strings.Join(names, ", "), target.Name)
			if err := ui.Run(title, func(ctx context.Context, report ui.Reporter) error {
				return server.Vault(ctx, target.Path, "install", spec.kind, names, report)
			}); err != nil {
				return err
			}
			if enable {
				return enableResources(target, spec.kind, names)
			}
			ui.Hint("load them on start with %s", ui.Command(fmt.Sprintf("nwm %s enable %s", spec.use, strings.Join(names, " "))))
			return nil
		},
	}
	addServerFlag(command, &serverName)
	command.Flags().BoolVarP(&enable, "enable", "e", false, "add them to the server's Config.toml")
	return command
}

func newResourceUpdateCommand(spec resourceCommandSpec) *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "update [name]...",
		Aliases: []string{"upgrade"},
		Short:   fmt.Sprintf("Update %s from the vault (all installed if none given)", spec.kind.Plural),
		Args:    cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, names []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				installed, err := server.ListResources(target.Path, spec.kind)
				if err != nil {
					return err
				}
				for _, resource := range installed {
					names = append(names, resource.Name)
				}
			}
			if len(names) == 0 {
				ui.Hint("no %s installed on %s", spec.kind.Plural, target.Name)
				return nil
			}
			title := fmt.Sprintf("Updating %d %s on %s", len(names), spec.kind.Plural, target.Name)
			return ui.Run(title, func(ctx context.Context, report ui.Reporter) error {
				return server.Vault(ctx, target.Path, "update", spec.kind, names, report)
			})
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newResourceListCommand(spec resourceCommandSpec) *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   fmt.Sprintf("List installed %s", spec.kind.Plural),
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			resources, err := server.ListResources(target.Path, spec.kind)
			if err != nil {
				return err
			}
			if len(resources) == 0 {
				ui.Hint("no %s installed on %s", spec.kind.Plural, target.Name)
				return nil
			}
			config, err := server.LoadConfig(target.Path)
			if err != nil {
				return err
			}
			headers := []string{"NAME", "TITLE", "VERSION", "AUTHOR", "ENABLED"}
			if spec.kind == server.PackageKind {
				headers = []string{"NAME", "TITLE", "TYPE", "VERSION", "AUTHOR", "ENABLED"}
			}
			rows := make([][]string, 0, len(resources))
			for _, resource := range resources {
				enabled := ""
				if resourceEnabled(config, spec.kind, resource) {
					enabled = "✓"
				}
				row := []string{resource.Name, resource.Title, resource.Version, resource.Author, enabled}
				if spec.kind == server.PackageKind {
					row = []string{resource.Name, resource.Title, resource.Type, resource.Version, resource.Author, enabled}
				}
				rows = append(rows, row)
			}
			fmt.Println(renderTable(headers, rows))
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func resourceEnabled(config *server.Config, kind server.Kind, resource server.Resource) bool {
	if kind == server.PackageKind && server.SingleSlotType(resource.Type) {
		return config.String("game."+resource.Type) == resource.Name
	}
	value, found := config.Get(kind.ConfigKey)
	if !found {
		return false
	}
	entries, isList := value.([]any)
	if !isList {
		return false
	}
	for _, entry := range entries {
		if fmt.Sprint(entry) == resource.Name {
			return true
		}
	}
	return false
}

func newResourceRemoveCommand(spec resourceCommandSpec) *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "remove <name>...",
		Aliases: []string{"rm", "uninstall"},
		Short:   fmt.Sprintf("Delete %s from a server and its Config.toml", spec.kind.Plural),
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, names []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			config, err := server.LoadConfig(target.Path)
			if err != nil {
				return err
			}
			for _, name := range names {
				resource, err := server.FindResource(target.Path, spec.kind, name)
				if err != nil {
					return fmt.Errorf("%s %q is not installed on %s", spec.kind.Singular, name, target.Name)
				}
				if _, err := server.Disable(config, spec.kind, name); err != nil {
					return err
				}
				if err := fsutil.RemoveAll(resource.Path); err != nil {
					return err
				}
				ui.Success("Removed %s", name)
			}
			if config.Exists() {
				return config.Save()
			}
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newResourceToggleCommand(spec resourceCommandSpec, enable bool) *cobra.Command {
	var serverName string
	use, short := "enable", "Load %s on server start via Config.toml"
	if !enable {
		use, short = "disable", "Stop loading %s on server start"
	}
	command := &cobra.Command{
		Use:   use + " <name>...",
		Short: fmt.Sprintf(short, spec.kind.Plural),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, names []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			if enable {
				return enableResources(target, spec.kind, names)
			}
			return disableResources(target, spec.kind, names)
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func enableResources(target *state.Server, kind server.Kind, names []string) error {
	config, err := server.LoadConfig(target.Path)
	if err != nil {
		return err
	}
	for _, name := range names {
		resource, err := server.FindResource(target.Path, kind, name)
		if err != nil {
			return fmt.Errorf("%s %q is not installed in %s", kind.Singular, name, filepath.Join(target.Path, kind.Directory))
		}
		key, err := server.Enable(config, kind, resource)
		if err != nil {
			return err
		}
		if key == "" {
			ui.Hint("%s is already enabled", name)
			continue
		}
		ui.Success("Enabled %s %s", name, ui.Muted.Render("("+key+")"))
	}
	if err := config.Save(); err != nil {
		return err
	}
	warnBlockedCModules(target, nil)
	return nil
}

func disableResources(target *state.Server, kind server.Kind, names []string) error {
	config, err := server.LoadConfig(target.Path)
	if err != nil {
		return err
	}
	for _, name := range names {
		keys, err := server.Disable(config, kind, name)
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			ui.Hint("%s was not enabled", name)
			continue
		}
		ui.Success("Disabled %s %s", name, ui.Muted.Render("("+strings.Join(keys, ", ")+")"))
	}
	return config.Save()
}
