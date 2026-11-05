package cli

import (
	"context"
	"fmt"
	"runtime"

	"github.com/DasDarki/nwm/internal/buildinfo"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/DasDarki/nwm/internal/updater"
	"github.com/spf13/cobra"
)

func newSelfUpdateCommand() *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:     "self-update",
		Aliases: []string{"su", "upgrade"},
		Short:   "⚡ Update nwm to the latest release",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if buildinfo.IsDevelopment() && !force {
				return fmt.Errorf("this is a development build, use --force to replace it with the latest release")
			}
			var release *updater.Release
			if err := ui.Run("Checking for updates", func(ctx context.Context, _ ui.Reporter) error {
				latest, err := updater.Latest(ctx)
				release = latest
				return err
			}); err != nil {
				return err
			}
			if !force && !updater.IsNewer(release.Tag, buildinfo.Version) {
				ui.Success("nwm %s is up to date", buildinfo.Version)
				return nil
			}
			if err := ui.Run("Updating nwm to "+release.Tag, func(ctx context.Context, report ui.Reporter) error {
				return updater.Apply(ctx, release, report)
			}); err != nil {
				return err
			}
			ui.Step("🎉", "nwm is now on %s", ui.Accent.Render(release.Tag))
			ui.Hint("release notes: %s", release.URL)
			return nil
		},
	}
	command.Flags().BoolVar(&force, "force", false, "reinstall even if already up to date")
	return command
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the nwm version",
		Args:  cobra.NoArgs,
		Run: func(*cobra.Command, []string) {
			fmt.Println(banner())
			fmt.Println(ui.Muted.Render(fmt.Sprintf("%s/%s · %s", runtime.GOOS, runtime.GOARCH, runtime.Version())))
		},
	}
}
