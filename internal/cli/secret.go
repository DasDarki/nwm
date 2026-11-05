package cli

import (
	"fmt"
	"strings"

	"github.com/DasDarki/nwm/internal/secrets"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func newSecretCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "secret",
		Aliases: []string{"secrets"},
		Short:   "🔑 Manage a server's secrets (passwords, tokens)",
		Long: "Secrets are stored per server in ~/.nwm/secrets/<server>.toml (readable only by you)\n" +
			"and referenced in recipes as ${secret.name}. Recipes create them on apply.",
	}
	command.AddCommand(
		newSecretListCommand(),
		newSecretGetCommand(),
		newSecretSetCommand(),
		newSecretRotateCommand(),
		newSecretRemoveCommand(),
	)
	return command
}

func openSecrets(serverName string) (string, *secrets.Store, error) {
	_, target, err := resolveServer(serverName)
	if err != nil {
		return "", nil, err
	}
	store, err := secrets.Open(target.Name)
	return target.Name, store, err
}

func completeSecretNames(command *cobra.Command, arguments []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(arguments) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverName, _ := command.Flags().GetString("server")
	_, store, err := openSecrets(serverName)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return store.Names(), cobra.ShellCompDirectiveNoFileComp
}

func newSecretListCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List secret names (values stay hidden)",
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			name, store, err := openSecrets(serverName)
			if err != nil {
				return err
			}
			names := store.Names()
			if len(names) == 0 {
				ui.Hint("%s has no secrets", name)
				return nil
			}
			rows := make([][]string, 0, len(names))
			for _, secretName := range names {
				value, _ := store.Get(secretName)
				rows = append(rows, []string{secretName, strings.Repeat("•", min(len(value), 12)), fmt.Sprintf("%d chars", len(value))})
			}
			fmt.Println(renderTable([]string{"NAME", "VALUE", "LENGTH"}, rows))
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newSecretGetCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:               "get <name>",
		Short:             "Print a secret's value",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSecretNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			_, store, err := openSecrets(serverName)
			if err != nil {
				return err
			}
			value, found := store.Get(arguments[0])
			if !found {
				return fmt.Errorf("secret %q not found", arguments[0])
			}
			fmt.Println(value)
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newSecretSetCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:   "set <name> [value]",
		Short: "Set a secret, asking for the value if it is not given",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, arguments []string) error {
			name, store, err := openSecrets(serverName)
			if err != nil {
				return err
			}
			var value string
			if len(arguments) == 2 {
				value = arguments[1]
			} else {
				if !ui.StdinInteractive() {
					return fmt.Errorf("pass the value as an argument when not running in a terminal")
				}
				if value, err = ui.PromptSecret("Value for " + arguments[0]); err != nil {
					return err
				}
			}
			if value == "" {
				return fmt.Errorf("secret values cannot be empty")
			}
			store.Set(arguments[0], value)
			if err := store.Save(); err != nil {
				return err
			}
			ui.Success("Stored %s for %s", arguments[0], name)
			ui.Hint("restart services or re-apply the recipe to use the new value")
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}

func newSecretRotateCommand() *cobra.Command {
	var serverName string
	var length int
	command := &cobra.Command{
		Use:               "rotate <name>",
		Short:             "Replace a secret with a new random value",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSecretNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			name, store, err := openSecrets(serverName)
			if err != nil {
				return err
			}
			value, err := secrets.Random(length)
			if err != nil {
				return err
			}
			store.Set(arguments[0], value)
			if err := store.Save(); err != nil {
				return err
			}
			ui.Success("Rotated %s for %s", arguments[0], name)
			ui.Warn("databases keep their old password in existing volumes, change it there too or recreate the volume")
			return nil
		},
	}
	addServerFlag(command, &serverName)
	command.Flags().IntVar(&length, "length", secrets.DefaultLength, "length of the new value")
	return command
}

func newSecretRemoveCommand() *cobra.Command {
	var serverName string
	command := &cobra.Command{
		Use:               "remove <name>",
		Aliases:           []string{"rm"},
		Short:             "Delete a secret",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSecretNames,
		RunE: func(_ *cobra.Command, arguments []string) error {
			name, store, err := openSecrets(serverName)
			if err != nil {
				return err
			}
			if !store.Delete(arguments[0]) {
				return fmt.Errorf("secret %q not found", arguments[0])
			}
			if err := store.Save(); err != nil {
				return err
			}
			ui.Success("Removed %s from %s", arguments[0], name)
			return nil
		},
	}
	addServerFlag(command, &serverName)
	return command
}
