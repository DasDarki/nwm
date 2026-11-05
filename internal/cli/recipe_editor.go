package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DasDarki/nwm/internal/recipe"
	"github.com/DasDarki/nwm/internal/store"
	"github.com/DasDarki/nwm/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const visibleSuggestions = 6

var (
	editorKeyword     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Bold(true)
	editorValue       = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	editorMuted       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	editorSelected    = lipgloss.NewStyle().Background(lipgloss.Color("#8B5CF6")).Foreground(lipgloss.Color("#FFFFFF"))
	editorError       = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	editorSuccess     = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	editorPreview     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#4B5563")).Padding(0, 1)
	editorPlaceholder = "type a command, e.g. package sandbox enable (tab completes)"
)

type storeLoadedMessage struct {
	index *store.Index
	err   error
}

type editorModel struct {
	document       *recipe.Document
	completer      *recipe.Completer
	input          textinput.Model
	suggestions    []recipe.Suggestion
	selected       int
	completionBase string
	completionAt   int
	history        []string
	historyIndex   int
	snapshots      []string
	dirty          bool
	confirmQuit    bool
	message        string
	messageIsError bool
	helpText       string
	scroll         int
	width          int
	height         int
}

func newEditorModel(document *recipe.Document, dirty bool) editorModel {
	input := textinput.New()
	input.Prompt = editorKeyword.Render("› ")
	input.Placeholder = editorPlaceholder
	input.Focus()
	return editorModel{
		document:  document,
		completer: &recipe.Completer{Manifest: document.Manifest, Dir: document.Dir, Editor: true},
		input:     input,
		selected:  -1,
		dirty:     dirty,
		width:     100,
		height:    30,
		message:   "loading store catalog for completion...",
	}
}

func (m editorModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		index, err := store.Load(ctx, false)
		return storeLoadedMessage{index: index, err: err}
	})
}

func (m editorModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		m.input.Width = max(20, typed.Width-6)
		return m, nil
	case storeLoadedMessage:
		if typed.err != nil {
			m.setMessage("store catalog unavailable, package names will not complete", true)
		} else {
			m.completer.Store = typed.index
			m.setMessage(fmt.Sprintf("ready · %d packages and %d asset packs from the store", len(typed.index.Packages), len(typed.index.Assets)), false)
		}
		m.refreshSuggestions()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(typed)
	}
	var command tea.Cmd
	m.input, command = m.input.Update(message)
	return m, command
}

func (m editorModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		if m.dirty && !m.confirmQuit {
			m.confirmQuit = true
			m.setMessage("unsaved changes, press Ctrl+C again to discard them or type save", true)
			return m, nil
		}
		return m, tea.Quit
	case tea.KeyTab, tea.KeyShiftTab:
		m.cycleCompletion(key.Type == tea.KeyShiftTab)
		return m, nil
	case tea.KeyUp, tea.KeyDown:
		m.browseHistory(key.Type == tea.KeyUp)
		return m, nil
	case tea.KeyPgUp:
		m.scroll = max(0, m.scroll-5)
		return m, nil
	case tea.KeyPgDown:
		m.scroll += 5
		return m, nil
	case tea.KeyEsc:
		m.input.SetValue("")
		m.refreshSuggestions()
		return m, nil
	case tea.KeyEnter:
		line := strings.TrimSpace(m.input.Value())
		m.input.SetValue("")
		m.historyIndex = len(m.history)
		if line == "" {
			m.refreshSuggestions()
			return m, nil
		}
		if len(m.history) == 0 || m.history[len(m.history)-1] != line {
			m.history = append(m.history, line)
		}
		m.historyIndex = len(m.history)
		quit := m.execute(line)
		m.refreshSuggestions()
		if quit {
			return m, tea.Quit
		}
		return m, nil
	}
	m.confirmQuit = false
	var command tea.Cmd
	m.input, command = m.input.Update(key)
	m.refreshSuggestions()
	return m, command
}

func (m *editorModel) refreshSuggestions() {
	m.selected = -1
	m.completionBase = m.input.Value()
	if strings.TrimSpace(m.completionBase) == "" {
		m.suggestions = nil
		return
	}
	m.completionAt, m.suggestions = m.completer.Complete(m.completionBase)
}

func (m *editorModel) cycleCompletion(backwards bool) {
	if strings.TrimSpace(m.input.Value()) == "" && len(m.suggestions) == 0 {
		m.completionBase = ""
		m.completionAt, m.suggestions = m.completer.Complete("")
	}
	if len(m.suggestions) == 0 {
		return
	}
	switch {
	case m.selected < 0 && backwards:
		m.selected = len(m.suggestions) - 1
	case m.selected < 0:
		m.selected = 0
	case backwards:
		m.selected = (m.selected - 1 + len(m.suggestions)) % len(m.suggestions)
	default:
		m.selected = (m.selected + 1) % len(m.suggestions)
	}
	completed := m.completionBase[:m.completionAt] + recipe.QuoteWord(m.suggestions[m.selected].Text)
	if len(m.suggestions) == 1 {
		completed += " "
	}
	m.input.SetValue(completed)
	m.input.CursorEnd()
	if len(m.suggestions) == 1 {
		m.refreshSuggestions()
	}
}

func (m *editorModel) browseHistory(older bool) {
	if len(m.history) == 0 {
		return
	}
	if older && m.historyIndex > 0 {
		m.historyIndex--
	} else if !older && m.historyIndex < len(m.history) {
		m.historyIndex++
	}
	if m.historyIndex == len(m.history) {
		m.input.SetValue("")
	} else {
		m.input.SetValue(m.history[m.historyIndex])
	}
	m.input.CursorEnd()
	m.refreshSuggestions()
}

func (m *editorModel) setMessage(text string, isError bool) {
	m.message, m.messageIsError = text, isError
}

func (m *editorModel) snapshot() {
	m.snapshots = append(m.snapshots, recipe.RenderDSL(m.document.Manifest))
}

func (m *editorModel) restore(rendered string) error {
	manifest, err := recipe.ParseDSL(rendered)
	if err != nil {
		return err
	}
	*m.document.Manifest = *manifest
	return nil
}

func (m *editorModel) execute(line string) bool {
	m.helpText = ""
	m.confirmQuit = false
	fields := strings.Fields(line)
	switch fields[0] {
	case "quit", "exit", "q":
		if m.dirty {
			m.setMessage("unsaved changes, type save first or quit! to discard them", true)
			return false
		}
		return true
	case "quit!", "exit!", "q!":
		return true
	case "save", "w":
		return m.save(fields[1:])
	case "wq":
		m.save(nil)
		return !m.dirty
	case "undo", "u":
		if len(m.snapshots) == 0 {
			m.setMessage("nothing to undo", true)
			return false
		}
		previous := m.snapshots[len(m.snapshots)-1]
		m.snapshots = m.snapshots[:len(m.snapshots)-1]
		if err := m.restore(previous); err != nil {
			m.setMessage(err.Error(), true)
			return false
		}
		m.dirty = true
		m.setMessage("undone", false)
		return false
	case "validate":
		if err := m.document.Manifest.Validate(m.document.Dir); err != nil {
			m.setMessage(strings.ReplaceAll(err.Error(), "\n", " "), true)
			return false
		}
		m.setMessage("the recipe is valid", false)
		return false
	case "help", "?":
		m.helpText = editorHelp(fields[1:])
		m.setMessage("", false)
		return false
	case "remove", "rm":
		if len(fields) < 3 {
			m.setMessage("usage: remove <kind> <name>", true)
			return false
		}
		m.snapshot()
		if err := m.document.Manifest.Remove(fields[1], strings.Join(fields[2:], " ")); err != nil {
			m.snapshots = m.snapshots[:len(m.snapshots)-1]
			m.setMessage(err.Error(), true)
			return false
		}
		m.dirty = true
		m.setMessage("removed "+fields[1]+" "+strings.Join(fields[2:], " "), false)
		return false
	}
	m.snapshot()
	if err := m.document.Manifest.ApplyLine(line); err != nil {
		previous := m.snapshots[len(m.snapshots)-1]
		m.snapshots = m.snapshots[:len(m.snapshots)-1]
		m.restore(previous)
		m.setMessage(err.Error(), true)
		return false
	}
	m.dirty = true
	m.setMessage("✓ "+line, false)
	if key, unknown := recipe.UnknownConfigKey(line); unknown {
		m.setMessage("✓ "+line+" (warning: "+key+" is not a known Config.toml key)", true)
	}
	return false
}

func (m *editorModel) save(arguments []string) bool {
	if len(arguments) > 0 {
		format, err := recipe.ParseFormat(arguments[0])
		if err != nil {
			m.setMessage(err.Error(), true)
			return false
		}
		if err := m.document.Convert(format); err != nil {
			m.setMessage(err.Error(), true)
			return false
		}
	} else if err := m.document.Save(); err != nil {
		m.setMessage(err.Error(), true)
		return false
	}
	m.dirty = false
	m.setMessage("saved "+m.document.Path(), false)
	return false
}

func editorHelp(arguments []string) string {
	var builder strings.Builder
	for _, statement := range recipe.Statements() {
		if len(arguments) > 0 && statement.Keyword != arguments[0] && !containsString(statement.Aliases, arguments[0]) {
			continue
		}
		builder.WriteString(editorKeyword.Render(statement.Usage) + "\n")
		detail := "  " + statement.Summary
		if len(statement.Aliases) > 0 {
			detail += " · alias " + strings.Join(statement.Aliases, ", ")
		}
		builder.WriteString(editorMuted.Render(detail) + "\n")
	}
	if len(arguments) == 0 {
		builder.WriteString("\n")
		for _, command := range recipe.EditorCommands {
			builder.WriteString(editorKeyword.Render(command.Usage) + editorMuted.Render("  "+command.Summary) + "\n")
		}
	}
	if builder.Len() == 0 {
		return editorError.Render("unknown command " + arguments[0])
	}
	return strings.TrimRight(builder.String(), "\n")
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func highlightLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return ""
	}
	keyword, rest, _ := strings.Cut(line, " ")
	return editorKeyword.Render(keyword) + " " + editorValue.Render(rest)
}

func (m editorModel) View() string {
	var builder strings.Builder
	title := ui.Accent.Render("📜 nwm recipe editor") + "  " + editorValue.Render(m.document.Manifest.Recipe.Name)
	status := editorMuted.Render("  " + m.document.Path())
	if m.dirty {
		status += editorError.Render("  ● modified")
	}
	builder.WriteString(title + status + "\n")

	suggestionLines := m.renderSuggestions()
	reserved := 6 + len(suggestionLines)
	available := max(3, m.height-reserved)
	var body []string
	if m.helpText != "" {
		body = strings.Split(m.helpText, "\n")
	} else {
		for _, line := range strings.Split(strings.TrimRight(recipe.RenderDSL(m.document.Manifest), "\n"), "\n") {
			body = append(body, highlightLine(line))
		}
	}
	if len(body) > available {
		maxScroll := len(body) - available + 1
		offset := min(m.scroll, maxScroll)
		visible := append([]string{}, body[offset:offset+available-1]...)
		hidden := len(body) - offset - len(visible)
		if hidden > 0 {
			visible = append(visible, editorMuted.Render(fmt.Sprintf("… %d more lines (PgDn)", hidden)))
		} else {
			visible = append(visible, editorMuted.Render(fmt.Sprintf("… %d lines above (PgUp)", offset)))
		}
		body = visible
	}
	contentWidth := max(20, m.width-4)
	for index, line := range body {
		body[index] = ansi.Truncate(line, contentWidth, "…")
	}
	builder.WriteString(editorPreview.Width(max(24, m.width-2)).Render(strings.Join(body, "\n")) + "\n")
	builder.WriteString(m.input.View() + "\n")
	for _, line := range suggestionLines {
		builder.WriteString(line + "\n")
	}
	if m.message != "" {
		style := editorSuccess
		if m.messageIsError {
			style = editorError
		}
		builder.WriteString(style.Render(ansi.Truncate(m.message, m.width-2, "…")) + "\n")
	}
	builder.WriteString(editorMuted.Render("tab complete · ↑↓ history · PgUp/PgDn scroll · enter run · help · undo · save [toml|dsl] · quit"))
	return builder.String()
}

func (m editorModel) renderSuggestions() []string {
	if len(m.suggestions) == 0 {
		return nil
	}
	first := 0
	if m.selected >= visibleSuggestions {
		first = m.selected - visibleSuggestions + 1
	}
	last := min(len(m.suggestions), first+visibleSuggestions)
	var lines []string
	for index := first; index < last; index++ {
		suggestion := m.suggestions[index]
		text := "  " + suggestion.Text
		if index == m.selected {
			text = editorSelected.Render(text + " ")
		} else {
			text = editorValue.Render(text)
		}
		if suggestion.Detail != "" {
			text += editorMuted.Render("  " + suggestion.Detail)
		}
		lines = append(lines, ansi.Truncate(text, m.width-1, "…"))
	}
	if remaining := len(m.suggestions) - last; remaining > 0 {
		lines = append(lines, editorMuted.Render(fmt.Sprintf("    +%d more", remaining)))
	}
	return lines
}

func runEditor(document *recipe.Document, dirty bool) error {
	program := tea.NewProgram(newEditorModel(document, dirty), tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		return err
	}
	if model, ok := final.(editorModel); ok && model.dirty {
		ui.Warn("left the editor without saving")
		return nil
	}
	ui.Success("%s saved", document.Path())
	return nil
}
