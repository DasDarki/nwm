package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func guardUnsafeLibs(current *state.State, target *state.Server, recipeArguments []string, extra []string, allow bool) ([]string, error) {
	if !slices.Contains(recipeArguments, server.UnsafeLibsFlag) || slices.Contains(extra, server.UnsafeLibsFlag) || current.AllowUnsafeLibs {
		return recipeArguments, nil
	}
	modules := server.EnabledCModules(target.Path)
	if target.UnsafeLibs.Covers(modules) {
		return recipeArguments, nil
	}
	choice := "y"
	if !allow {
		describeUnsafeLibs(target, modules)
		choice = ui.Choose("Allow "+server.UnsafeLibsFlag+" for "+target.Name+"?", "[y] yes, this server · [a] always, all servers · [N] no", "n")
	}
	switch choice {
	case "y":
		target.UnsafeLibs = &state.UnsafeLibsApproval{ApprovedAt: time.Now(), Modules: modules}
	case "a":
		current.AllowUnsafeLibs = true
	default:
		ui.Warn("starting %s without %s, C modules and unsafe Lua libraries stay disabled", target.Name, server.UnsafeLibsFlag)
		ui.Hint("approve later with %s or %s", ui.Command("nwm s trust "+target.Name), ui.Command("nwm s start "+target.Name+" --allow-unsafe-libs"))
		return slices.DeleteFunc(slices.Clone(recipeArguments), func(argument string) bool { return argument == server.UnsafeLibsFlag }), nil
	}
	if err := current.Save(); err != nil {
		return nil, err
	}
	return recipeArguments, nil
}

func describeUnsafeLibs(target *state.Server, modules []string) {
	fmt.Fprintln(os.Stderr)
	ui.Warn("the recipe of %s starts the server with %s", target.Name, server.UnsafeLibsFlag)
	fmt.Fprintln(os.Stderr, ui.Muted.Render("   This lets packages run native code (C modules) and use unsafe Lua libraries"))
	fmt.Fprintln(os.Stderr, ui.Muted.Render("   such as io and os, with full access to this machine. Only allow it for code you trust."))
	if len(modules) > 0 {
		fmt.Fprintln(os.Stderr, "   "+ui.Bold.Render("C modules: ")+strings.Join(modules, ", "))
	}
}

func newServerTrustCommand() *cobra.Command {
	var revoke bool
	var global bool
	command := &cobra.Command{
		Use:   "trust [name]",
		Short: "Show, approve or revoke " + server.UnsafeLibsFlag + " for a server or globally",
		Example: "  nwm s trust dev\n" +
			"  nwm s trust dev --revoke\n" +
			"  nwm s trust --global",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			if global {
				current, err := state.Load()
				if err != nil {
					return err
				}
				current.AllowUnsafeLibs = !revoke
				if err := current.Save(); err != nil {
					return err
				}
				if revoke {
					ui.Success("%s is no longer allowed globally, servers ask again", server.UnsafeLibsFlag)
				} else {
					ui.Warn("%s is now allowed for all servers without asking", server.UnsafeLibsFlag)
				}
				return nil
			}
			current, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			modules := server.EnabledCModules(target.Path)
			switch {
			case revoke:
				target.UnsafeLibs = nil
				if err := current.Save(); err != nil {
					return err
				}
				ui.Success("Revoked %s for %s", server.UnsafeLibsFlag, target.Name)
			case current.AllowUnsafeLibs || target.UnsafeLibs.Covers(modules):
				printTrust(current, target, modules)
			default:
				describeUnsafeLibs(target, modules)
				if !ui.Confirm("Allow " + server.UnsafeLibsFlag + " for " + target.Name + "?") {
					ui.Hint("nothing changed")
					return nil
				}
				target.UnsafeLibs = &state.UnsafeLibsApproval{ApprovedAt: time.Now(), Modules: modules}
				if err := current.Save(); err != nil {
					return err
				}
				ui.Success("Allowed %s for %s", server.UnsafeLibsFlag, target.Name)
			}
			return nil
		},
	}
	command.Flags().BoolVar(&revoke, "revoke", false, "withdraw the approval")
	command.Flags().BoolVar(&global, "global", false, "allow (or with --revoke disallow) it for all servers")
	return command
}

func printTrust(current *state.State, target *state.Server, modules []string) {
	ui.Step("🔓", "%s", ui.Accent.Render(target.Name))
	if current.AllowUnsafeLibs {
		ui.Detail("global", "allowed for all servers")
	}
	if target.UnsafeLibs != nil {
		ui.Detail("approved", target.UnsafeLibs.ApprovedAt.Format(time.DateTime))
		ui.Detail("approved modules", strings.Join(target.UnsafeLibs.Modules, ", "))
	}
	ui.Detail("enabled C modules", strings.Join(modules, ", "))
}
