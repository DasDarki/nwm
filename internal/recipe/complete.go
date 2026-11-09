package recipe

import (
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/DasDarki/nwm/internal/server"
	"github.com/DasDarki/nwm/internal/store"
)

const maxSuggestions = 50

type Suggestion struct {
	Text   string
	Detail string
}

type EditorCommand struct {
	Name    string
	Usage   string
	Summary string
}

var EditorCommands = []EditorCommand{
	{Name: "remove", Usage: "remove <kind> <name>", Summary: "remove an entry, e.g. remove package sandbox"},
	{Name: "undo", Usage: "undo", Summary: "revert the last change"},
	{Name: "validate", Usage: "validate", Summary: "check the recipe for mistakes"},
	{Name: "save", Usage: "save [toml|dsl]", Summary: "write the recipe, optionally converting it"},
	{Name: "help", Usage: "help [command]", Summary: "show commands"},
	{Name: "quit", Usage: "quit | quit!", Summary: "leave the editor (quit! discards changes)"},
}

type Completer struct {
	Manifest *Manifest
	Dir      string
	Store    *store.Index
	Editor   bool
}

func (c *Completer) Complete(line string) (int, []Suggestion) {
	start := currentTokenStart(line)
	prefix := strings.Trim(line[start:], `"'`)
	parsed, _ := tokenize(line[:start])
	previous := parsed.tokens
	return start, filterSuggestions(c.candidates(previous), prefix)
}

func currentTokenStart(line string) int {
	inQuote := rune(0)
	start := 0
	for index, character := range line {
		switch {
		case inQuote != 0:
			if character == inQuote {
				inQuote = 0
			}
		case character == '"' || character == '\'':
			inQuote = character
		case unicode.IsSpace(character):
			start = index + 1
		}
	}
	return start
}

func (c *Completer) candidates(previous []token) []Suggestion {
	if len(previous) == 0 {
		return c.keywords()
	}
	keyword := previous[0].Text
	if keyword == "dev" {
		if len(previous) == 1 {
			var result []Suggestion
			for _, suggestion := range c.keywords() {
				if !slices.Contains(devForbidden, suggestion.Text) && !isEditorCommand(suggestion.Text) {
					result = append(result, suggestion)
				}
			}
			return result
		}
		return c.candidates(previous[1:])
	}
	position := len(previous)
	last := previous[len(previous)-1].Text
	if c.Editor {
		switch keyword {
		case "remove", "rm":
			if position == 1 {
				return plain(RemovableKinds)
			}
			if position == 2 && c.Manifest != nil {
				return plain(c.Manifest.Names(previous[1].Text))
			}
			return nil
		case "save":
			if position == 1 {
				return []Suggestion{{Text: "toml", Detail: FileName}, {Text: "dsl", Detail: DSLFileName}}
			}
			return nil
		case "help":
			if position == 1 {
				return c.keywords()
			}
			return nil
		}
	}
	statement, found := findStatement(keyword)
	if !found {
		return nil
	}
	used := map[string]bool{}
	for _, item := range previous[1:] {
		used[item.Text] = true
	}
	switch statement.Keyword {
	case "package", "assets":
		if position == 1 {
			return c.storeItems(statement.Keyword == "assets")
		}
		switch last {
		case "path", "subdir", "local", "output", "web":
			return c.files(true)
		case "git", "ref":
			return nil
		case "build":
			return []Suggestion{{Text: "nebra build", Detail: "detected automatically when the package has a nebra.toml"}}
		case "watch":
			return []Suggestion{{Text: "nebra watch", Detail: "detected automatically when the package has a nebra.toml"}}
		case "reload":
			return []Suggestion{{Text: ReloadHot, Detail: "package hotreload, keeps the Lua state (default)"}, {Text: ReloadFull, Detail: "package reload, starts the package fresh"}}
		}
		return unusedOptions(statement.Options, used)
	case "use":
		if position == 1 {
			return presetSuggestions()
		}
		if last == "as" || last == "port" || last == "env" {
			return nil
		}
		return unusedOptions([]string{"as", "port"}, used, "env")
	case "service":
		if position == 2 {
			var images []Suggestion
			for _, name := range PresetNames() {
				preset := servicePresets[name]
				images = append(images, Suggestion{Text: preset.Image, Detail: preset.Description})
			}
			return images
		}
		if position > 2 && !slices.Contains([]string{"port", "env", "volume", "ready"}, last) && !used["command"] {
			return unusedOptions([]string{"ready", "nowait", "command"}, used, "port", "env", "volume")
		}
		return nil
	case "config":
		if position == 1 {
			return configKeySuggestions(c.Manifest)
		}
		if position == 2 {
			if value, known := server.DefaultConfigKeys()[previous[1].Text]; known && value != nil {
				return []Suggestion{{Text: formatValue(value), Detail: "default"}}
			}
		}
		return nil
	case "expose":
		if c.Manifest == nil {
			return nil
		}
		names := c.Manifest.Names("env")
		if c.Manifest.Dev != nil {
			names = append(names, c.Manifest.Dev.Names("env")...)
		}
		return unusedOptions(names, used)
	case "tool":
		var tools []Suggestion
		for _, tool := range KnownTools() {
			tools = append(tools, Suggestion{Text: tool, Detail: toolHints[tool]})
		}
		return tools
	case "branch", "nanossharp":
		if position == 1 {
			return plain(statement.Options)
		}
		return nil
	case "secret":
		if position == 2 {
			return []Suggestion{{Text: "random", Detail: "generate a random value (default)"}, {Text: "prompt", Detail: "ask for the value when applying"}}
		}
		return nil
	case "file":
		switch position {
		case 1:
			return c.files(false)
		case 3:
			return unusedOptions([]string{"template"}, used)
		}
		return nil
	case "module":
		if position == 1 {
			return nil
		}
		if last == "linux" || last == "windows" {
			return c.files(false)
		}
		return unusedOptions(statement.Options, used)
	case "env", "var":
		if position == 1 && c.Manifest != nil {
			return plain(c.Manifest.Names(statement.Keyword))
		}
		return nil
	}
	return nil
}

func isEditorCommand(text string) bool {
	for _, command := range EditorCommands {
		if command.Name == text {
			return true
		}
	}
	return false
}

func (c *Completer) keywords() []Suggestion {
	var result []Suggestion
	for _, statement := range statements {
		detail := statement.Summary
		if len(statement.Aliases) > 0 {
			detail += " (alias: " + strings.Join(statement.Aliases, ", ") + ")"
		}
		result = append(result, Suggestion{Text: statement.Keyword, Detail: detail})
		for _, alias := range statement.Aliases {
			result = append(result, Suggestion{Text: alias, Detail: "alias of " + statement.Keyword})
		}
	}
	if c.Editor {
		for _, command := range EditorCommands {
			result = append(result, Suggestion{Text: command.Name, Detail: command.Summary})
		}
	}
	return result
}

func (c *Completer) storeItems(assets bool) []Suggestion {
	if c.Store == nil {
		return nil
	}
	items := c.Store.Packages
	if assets {
		items = c.Store.Assets
	}
	result := make([]Suggestion, 0, len(items))
	for _, item := range items {
		detail := item.Title
		if !assets && item.Type != "" {
			detail += " [" + item.Type + "]"
		}
		if item.ShortDescription != "" {
			detail += " · " + item.ShortDescription
		}
		result = append(result, Suggestion{Text: item.Name, Detail: detail})
	}
	return result
}

func (c *Completer) files(directoriesOnly bool) []Suggestion {
	if c.Dir == "" {
		return nil
	}
	var result []Suggestion
	filepath.WalkDir(c.Dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || len(result) >= 200 {
			return filepath.SkipAll
		}
		relative, _ := filepath.Rel(c.Dir, path)
		if relative == "." {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == FileName || entry.Name() == DSLFileName {
			return nil
		}
		if entry.IsDir() || !directoriesOnly {
			result = append(result, Suggestion{Text: filepath.ToSlash(relative)})
		}
		return nil
	})
	return result
}

func presetSuggestions() []Suggestion {
	var result []Suggestion
	for _, name := range PresetNames() {
		preset := servicePresets[name]
		result = append(result, Suggestion{Text: name, Detail: preset.Description + " · " + preset.Image})
	}
	return result
}

func configKeySuggestions(manifest *Manifest) []Suggestion {
	defaults := server.DefaultConfigKeys()
	keys := map[string]string{}
	for key, value := range defaults {
		detail := ""
		if value != nil {
			detail = "default " + formatValue(value)
		}
		keys[key] = detail
	}
	if manifest != nil {
		for key, value := range manifest.ConfigKeys() {
			keys[key] = "current " + formatValue(value)
		}
	}
	result := make([]Suggestion, 0, len(keys))
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		result = append(result, Suggestion{Text: key, Detail: keys[key]})
	}
	return result
}

func unusedOptions(options []string, used map[string]bool, repeatable ...string) []Suggestion {
	var result []Suggestion
	for _, option := range options {
		if !used[option] {
			result = append(result, Suggestion{Text: option})
		}
	}
	return append(result, plain(repeatable)...)
}

func plain(values []string) []Suggestion {
	result := make([]Suggestion, len(values))
	for index, value := range values {
		result[index] = Suggestion{Text: value}
	}
	return result
}

func filterSuggestions(candidates []Suggestion, prefix string) []Suggestion {
	lowered := strings.ToLower(prefix)
	var starts, contains []Suggestion
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if seen[candidate.Text] {
			continue
		}
		text := strings.ToLower(candidate.Text)
		switch {
		case strings.HasPrefix(text, lowered):
			starts = append(starts, candidate)
		case lowered != "" && (strings.Contains(text, lowered) || strings.Contains(strings.ToLower(candidate.Detail), lowered)):
			contains = append(contains, candidate)
		default:
			continue
		}
		seen[candidate.Text] = true
	}
	result := append(starts, contains...)
	if len(result) > maxSuggestions {
		result = result[:maxSuggestions]
	}
	return result
}
