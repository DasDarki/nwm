package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/process"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/DasDarki/nwm/internal/updater"
	"github.com/spf13/cobra"
)

type exitCodeError struct {
	code int
}

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exited with code %d", e.code)
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "nwm",
		Short:         "🌍 nanos world manager",
		Long:          banner() + "\n\nCreate, update and run nanos world servers, set them up from recipes and manage packages,\nassets, NanosSharp and Docker services.",
		Version:       buildinfo.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			updater.CleanupPrevious()
		},
		PersistentPostRun: func(command *cobra.Command, _ []string) {
			if command.Name() == "self-update" || command.Hidden || !ui.Interactive() {
				return
			}
			if latest := updater.PendingNotice(context.Background()); latest != "" {
				fmt.Fprintln(os.Stderr)
				ui.Hint("nwm %s is available, run %s", latest, ui.Command("nwm self-update"))
			}
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(
		newServerCommand(),
		newResourceCommand(resourcePackages),
		newResourceCommand(resourceAssets),
		newNanosSharpCommand(),
		newRecipeCommand(),
		newServiceCommand(),
		newDevCommand(),
		newSecretCommand(),
		newSelfUpdateCommand(),
		newVersionCommand(),
		newSuperviseCommand(),
	)
	return root
}

func banner() string {
	return ui.Accent.Render("🌍 nwm") + ui.Muted.Render(" · nanos world manager · "+buildinfo.Version)
}

func Execute() int {
	err := newRootCommand().Execute()
	if err == nil {
		return 0
	}
	var exitCode exitCodeError
	if errors.As(err, &exitCode) {
		return exitCode.code
	}
	var commandErr *process.CommandError
	if errors.As(err, &commandErr) && len(commandErr.Tail) > 0 {
		fmt.Fprintln(os.Stderr, ui.Muted.Render("── last output of "+commandErr.Command+" ──"))
		for _, line := range commandErr.Tail {
			fmt.Fprintln(os.Stderr, ui.Muted.Render("  "+line))
		}
	}
	var exitErr *exec.ExitError
	if errors.Is(err, context.Canceled) {
		return 130
	}
	ui.Error("%v", err)
	if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
		return exitErr.ExitCode()
	}
	return 1
}
