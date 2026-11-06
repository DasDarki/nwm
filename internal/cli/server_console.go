package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func newSuperviseCommand() *cobra.Command {
	return &cobra.Command{
		Use:    server.SupervisorCommand() + " <name> <directory> -- [arguments]",
		Hidden: true,
		Args:   cobra.MinimumNArgs(2),
		RunE: func(command *cobra.Command, arguments []string) error {
			extra := []string{}
			if dash := command.ArgsLenAtDash(); dash >= 0 {
				extra = arguments[dash:]
			}
			err := server.Supervise(arguments[0], arguments[1], extra)
			var exitErr interface{ ExitCode() int }
			if errors.As(err, &exitErr) {
				return exitCodeError{code: exitErr.ExitCode()}
			}
			return err
		},
	}
}

func newServerCmdCommand() *cobra.Command {
	var serverName string
	var wait time.Duration
	command := &cobra.Command{
		Use:   "cmd <console command>...",
		Short: "Send a command to the console of a background server",
		Example: "  nwm s cmd package hotreload my-gamemode\n" +
			"  nwm s cmd -s dev chat \"restart in 5 minutes\"",
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(serverName)
			if err != nil {
				return err
			}
			file, offset := openLogAtEnd(target.Name)
			if err := server.SendCommand(target.Name, strings.Join(arguments, " ")); err != nil {
				return err
			}
			if file == nil {
				return nil
			}
			defer file.Close()
			time.Sleep(wait)
			file.Seek(offset, io.SeekStart)
			io.Copy(os.Stdout, file)
			return nil
		},
	}
	addServerFlag(command, &serverName)
	command.Flags().SetInterspersed(false)
	command.Flags().DurationVar(&wait, "wait", 1500*time.Millisecond, "how long to show the server's response")
	return command
}

func newServerAttachCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "attach [name]",
		Short:             "Follow a background server's console and type commands into it",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, target, err := resolveServer(optionalServerName(arguments))
			if err != nil {
				return err
			}
			if _, running := server.Status(target.Name); !running {
				return fmt.Errorf("server %q is not running", target.Name)
			}
			file, err := os.Open(server.LogFile(target.Name))
			if err != nil {
				return err
			}
			defer file.Close()
			ui.Step("🔌", "Attached to %s %s", ui.Accent.Render(target.Name), ui.Muted.Render("(type console commands, Ctrl+C detaches, the server keeps running)"))
			if err := printTail(file, 20); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()
			go func() {
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" {
						continue
					}
					if err := server.SendCommand(target.Name, line); err != nil {
						ui.Error("%v", err)
					}
				}
			}()
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						if _, running := server.Status(target.Name); !running {
							ui.Warn("server %s stopped", target.Name)
							cancel()
							return
						}
					}
				}
			}()
			err = followFile(ctx, file)
			fmt.Fprintln(os.Stderr)
			ui.Hint("detached from %s", target.Name)
			return err
		},
	}
	return command
}

func openLogAtEnd(name string) (*os.File, int64) {
	file, err := os.Open(server.LogFile(name))
	if err != nil {
		return nil, 0
	}
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		return nil, 0
	}
	return file, offset
}
