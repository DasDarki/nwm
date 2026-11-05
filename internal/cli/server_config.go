package cli

import (
	"fmt"

	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func newServerConfigCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:   "config",
		Short: "Show or change a server's Config.toml",
		Example: "  nwm server config -s dev\n" +
			"  nwm server config get discover.port -s dev\n" +
			"  nwm server config set discover.name '\"My Server\"' -s dev\n" +
			"  nwm server config set game.packages '[\"sandbox\"]' -s dev",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			config, err := loadServerConfig(serverName)
			if err != nil {
				return err
			}
			if !config.Exists() {
				ui.Hint("%s does not exist yet, it is created on the first server start", config.Path())
				return nil
			}
			rendered, err := config.Render()
			if err != nil {
				return err
			}
			fmt.Print(rendered)
			return nil
		},
	}
	addServerFlag(command, &serverName)

	get := &cobra.Command{
		Use:   "get <key>",
		Short: "Print a single value, e.g. discover.port",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			config, err := loadServerConfig(serverName)
			if err != nil {
				return err
			}
			value, found := config.Get(arguments[0])
			if !found {
				return fmt.Errorf("%s is not set", arguments[0])
			}
			fmt.Println(server.FormatValue(value))
			return nil
		},
	}
	addServerFlag(get, &serverName)

	set := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a value, given as TOML (plain words are treated as strings)",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, arguments []string) error {
			config, err := loadServerConfig(serverName)
			if err != nil {
				return err
			}
			value := server.ParseValue(arguments[1])
			if err := config.Set(arguments[0], value); err != nil {
				return err
			}
			if err := config.Save(); err != nil {
				return err
			}
			ui.Success("%s = %s", arguments[0], server.FormatValue(value))
			return nil
		},
	}
	addServerFlag(set, &serverName)

	command.AddCommand(get, set)
	return command
}

func loadServerConfig(serverName string) (*server.Config, error) {
	_, target, err := resolveServer(serverName)
	if err != nil {
		return nil, err
	}
	return server.LoadConfig(target.Path)
}
