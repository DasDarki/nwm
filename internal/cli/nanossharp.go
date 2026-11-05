package cli

import (
	"context"
	"fmt"

	"github.com/DasDarki/nwm/internal/nanossharp"
	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func newNanosSharpCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "nanossharp",
		Aliases: []string{"sharp", "cs"},
		Short:   "🧩 Install, update and remove NanosSharp (C# scripting)",
	}
	command.AddCommand(
		newNanosSharpActionCommand("install", "Install NanosSharp into a server", "Installing NanosSharp", nanossharp.Install),
		newNanosSharpActionCommand("update", "Update NanosSharp to the latest release", "Updating NanosSharp", nanossharp.Install),
		newNanosSharpActionCommand("uninstall", "Remove NanosSharp, keeping NanosSharp.toml", "Removing NanosSharp", nanossharp.Uninstall),
	)
	return command
}

func newNanosSharpActionCommand(use string, short string, title string, action func(context.Context, string, ui.Reporter) error) *cobra.Command {
	return &cobra.Command{
		Use:               use + " [server]",
		Short:             short,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			if err := ensureStopped(target); err != nil {
				return err
			}
			return ui.Run(fmt.Sprintf("%s on %s", title, target.Name), func(ctx context.Context, report ui.Reporter) error {
				return action(ctx, target.Path, report)
			})
		},
	}
}

func ensureStopped(target *state.Server) error {
	if info, running := server.Status(target.Name); running {
		return fmt.Errorf("server %q is running (pid %d), stop it first", target.Name, info.PID)
	}
	return nil
}
