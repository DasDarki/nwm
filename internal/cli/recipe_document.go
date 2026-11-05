package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/store"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/spf13/cobra"
)

func addRecipeDirFlag(command *cobra.Command, target *string) {
	command.Flags().StringVarP(target, "dir", "C", ".", "recipe directory or file, e.g. staging.Recipefile")
}

func openRecipeDocument(location string) (*recipe.Document, error) {
	absolute, err := filepath.Abs(location)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(absolute); err == nil && !info.IsDir() {
		return recipe.OpenDocumentFile(absolute)
	}
	return recipe.OpenDocument(absolute)
}

func joinStatement(arguments []string) string {
	parts := make([]string, len(arguments))
	for index, argument := range arguments {
		if strings.HasPrefix(argument, "[") || strings.HasPrefix(argument, "{") {
			parts[index] = argument
			continue
		}
		parts[index] = recipe.QuoteWord(argument)
	}
	return strings.Join(parts, " ")
}

func shellCompleter(command *cobra.Command, editor bool) *recipe.Completer {
	location, _ := command.Flags().GetString("dir")
	completer := &recipe.Completer{Dir: location, Editor: editor}
	if document, err := openRecipeDocument(location); err == nil {
		completer.Manifest = document.Manifest
		completer.Dir = document.Dir
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if index, err := store.Load(ctx, false); err == nil {
		completer.Store = index
	}
	return completer
}

func completeStatement(prefixWords []string, editor bool) cobra.CompletionFunc {
	return func(command *cobra.Command, arguments []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		words := append(append([]string{}, prefixWords...), arguments...)
		line := joinStatement(words)
		if line != "" {
			line += " "
		}
		_, suggestions := shellCompleter(command, editor).Complete(line + toComplete)
		completions := make([]cobra.Completion, 0, len(suggestions))
		for _, suggestion := range suggestions {
			completions = append(completions, cobra.CompletionWithDesc(suggestion.Text, suggestion.Detail))
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func newRecipeEditCommand() *cobra.Command {
	var useTOML bool
	var name string
	command := &cobra.Command{
		Use:   "edit [directory | file]",
		Short: "Edit a recipe interactively with completion",
		Long: "Opens a full screen editor. Every recipe statement is a command, e.g.\n" +
			"  package sandbox enable\n  use postgres\n  config discover.port 7790\n" +
			"Tab completes commands, store packages, presets, config keys and files.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			if !ui.Interactive() || !ui.StdinInteractive() {
				return fmt.Errorf("the editor needs a terminal, use %s instead", ui.Command("nwm recipe add"))
			}
			directory := "."
			if len(arguments) > 0 {
				directory = arguments[0]
			}
			directory, err := filepath.Abs(directory)
			if err != nil {
				return err
			}
			if info, statErr := os.Stat(directory); statErr == nil && !info.IsDir() {
				document, err := recipe.OpenDocumentFile(directory)
				if err != nil {
					return err
				}
				return runEditor(document, false)
			}
			if recipe.IsRecipeFileName(filepath.Base(directory)) {
				if name == "" {
					name = strings.ToLower(filepath.Base(filepath.Dir(directory)))
				}
				return runEditor(recipe.NewDocumentFile(directory, name), true)
			}
			_, found, err := recipe.Detect(directory)
			if err != nil {
				return err
			}
			if found {
				document, err := recipe.OpenDocument(directory)
				if err != nil {
					return err
				}
				return runEditor(document, false)
			}
			format := recipe.FormatDSL
			if useTOML {
				format = recipe.FormatTOML
			}
			if name == "" {
				name = strings.ToLower(filepath.Base(directory))
			}
			return runEditor(recipe.NewDocument(directory, format, name), true)
		},
	}
	command.Flags().BoolVar(&useTOML, "toml", false, "create recipe.toml instead of a Recipefile for new recipes")
	command.Flags().StringVar(&name, "name", "", "name for a new recipe, e.g. @you/my-recipe")
	return command
}

func newRecipeAddCommand() *cobra.Command {
	var directory string
	command := &cobra.Command{
		Use:   "add <statement>...",
		Short: "Add a statement to the recipe in the current directory",
		Example: "  nwm recipe add package sandbox enable\n" +
			"  nwm recipe add use postgres\n" +
			"  nwm recipe add config discover.port 7790\n" +
			"  nwm recipe add secret steam_token prompt",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeStatement(nil, false),
		RunE: func(_ *cobra.Command, arguments []string) error {
			document, err := openRecipeDocument(directory)
			if err != nil {
				return err
			}
			line := joinStatement(arguments)
			if err := document.Manifest.ApplyLine(line); err != nil {
				return err
			}
			if err := document.Save(); err != nil {
				return err
			}
			ui.Success("%s %s", line, ui.Muted.Render("-> "+document.Path()))
			if key, unknown := recipe.UnknownConfigKey(line); unknown {
				ui.Warn("%s is not a known Config.toml key, check for typos (custom values belong in custom_settings.*)", key)
			}
			return nil
		},
	}
	addRecipeDirFlag(command, &directory)
	return command
}

func newRecipeRemoveCommand() *cobra.Command {
	var directory string
	command := &cobra.Command{
		Use:     "rm <kind> <name>",
		Aliases: []string{"remove"},
		Short:   "Remove an entry from the recipe, e.g. rm package sandbox",
		Args:    cobra.MinimumNArgs(2),
		ValidArgsFunction: func(command *cobra.Command, arguments []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return completeStatement([]string{"remove"}, true)(command, arguments, toComplete)
		},
		RunE: func(_ *cobra.Command, arguments []string) error {
			document, err := openRecipeDocument(directory)
			if err != nil {
				return err
			}
			name := strings.Join(arguments[1:], " ")
			if err := document.Manifest.Remove(arguments[0], name); err != nil {
				return err
			}
			if err := document.Save(); err != nil {
				return err
			}
			ui.Success("Removed %s %s", arguments[0], name)
			return nil
		},
	}
	addRecipeDirFlag(command, &directory)
	return command
}

func newRecipeShowCommand() *cobra.Command {
	var formatName string
	command := &cobra.Command{
		Use:   "show [directory | file]",
		Short: "Print a recipe as Recipefile or TOML",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			directory := "."
			if len(arguments) > 0 {
				directory = arguments[0]
			}
			document, err := openRecipeDocument(directory)
			if err != nil {
				return err
			}
			format := document.Format
			if formatName != "" {
				if format, err = recipe.ParseFormat(formatName); err != nil {
					return err
				}
			}
			rendered, err := recipe.Render(document.Manifest, format)
			if err != nil {
				return err
			}
			fmt.Print(rendered)
			return nil
		},
	}
	command.Flags().StringVar(&formatName, "format", "", "output format: dsl or toml (default: the file's format)")
	return command
}

func newRecipeConvertCommand() *cobra.Command {
	var formatName string
	command := &cobra.Command{
		Use:   "convert [directory | file] --to <dsl|toml>",
		Short: "Convert between Recipefile and recipe.toml",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, arguments []string) error {
			directory := "."
			if len(arguments) > 0 {
				directory = arguments[0]
			}
			format, err := recipe.ParseFormat(formatName)
			if err != nil {
				return err
			}
			document, err := openRecipeDocument(directory)
			if err != nil {
				return err
			}
			previous := document.Path()
			if document.Format == format {
				ui.Hint("%s is already in that format", previous)
				return nil
			}
			if err := document.Convert(format); err != nil {
				return err
			}
			ui.Success("Converted %s -> %s", filepath.Base(previous), filepath.Base(document.Path()))
			return nil
		},
	}
	command.Flags().StringVar(&formatName, "to", "", "target format: dsl or toml")
	command.MarkFlagRequired("to")
	return command
}

func writeNewRecipe(directory string, fileName string, name string, format recipe.Format) (string, error) {
	if fileName != "" {
		format = recipe.FormatForFile(fileName)
		if _, err := os.Stat(filepath.Join(directory, fileName)); err == nil {
			return "", fmt.Errorf("%s already exists", filepath.Join(directory, fileName))
		}
	} else if _, found, err := recipe.Detect(directory); err != nil || found {
		if err == nil {
			err = fmt.Errorf("%s already contains a recipe", directory)
		}
		return "", err
	}
	manifest, err := recipe.ParseDSL(recipe.ScaffoldDSL(name))
	if err != nil {
		return "", err
	}
	document := &recipe.Document{Dir: directory, FileName: fileName, Format: format, Manifest: manifest}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	if format == recipe.FormatDSL {
		return document.Path(), os.WriteFile(document.Path(), []byte(recipe.ScaffoldDSL(name)), 0o644)
	}
	return document.Path(), document.Save()
}
