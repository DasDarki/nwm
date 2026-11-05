package cli

import (
	"fmt"
	"os"
	"regexp"

	"github.com/DasDarki/nwm/internal/state"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

var serverNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func resolveServer(name string) (*state.State, *state.Server, error) {
	current, err := state.Load()
	if err != nil {
		return nil, nil, err
	}
	if name != "" {
		server, found := current.Find(name)
		if !found {
			return nil, nil, fmt.Errorf("unknown server %q, see %s", name, ui.Command("nwm server list"))
		}
		return current, server, nil
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		if server, found := current.FindByPath(workingDirectory); found {
			return current, server, nil
		}
	}
	switch len(current.Servers) {
	case 0:
		return nil, nil, fmt.Errorf("no servers yet, create one with %s", ui.Command("nwm server create <name>"))
	case 1:
		return current, &current.Servers[0], nil
	default:
		return nil, nil, fmt.Errorf("multiple servers registered, pass a server name")
	}
}

func optionalServerName(arguments []string) string {
	if len(arguments) > 0 {
		return arguments[0]
	}
	return ""
}

func completeServerNames(_ *cobra.Command, arguments []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(arguments) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return serverNames(), cobra.ShellCompDirectiveNoFileComp
}

func completeServerFlag(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return serverNames(), cobra.ShellCompDirectiveNoFileComp
}

func serverNames() []string {
	current, err := state.Load()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(current.Servers))
	for _, server := range current.Servers {
		names = append(names, server.Name)
	}
	return names
}

func addServerFlag(command *cobra.Command, target *string) {
	command.Flags().StringVarP(target, "server", "s", "", "server to operate on (defaults to the server in the current directory or the only one)")
	command.RegisterFlagCompletionFunc("server", completeServerFlag)
}
