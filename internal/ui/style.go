package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var renderer = lipgloss.NewRenderer(os.Stderr)

var (
	colorAccent  = lipgloss.Color("#8B5CF6")
	colorSuccess = lipgloss.Color("#22C55E")
	colorError   = lipgloss.Color("#EF4444")
	colorWarning = lipgloss.Color("#F59E0B")
	colorMuted   = lipgloss.Color("#6B7280")
)

var (
	Accent       = renderer.NewStyle().Foreground(colorAccent).Bold(true)
	Muted        = renderer.NewStyle().Foreground(colorMuted)
	Bold         = renderer.NewStyle().Bold(true)
	successStyle = renderer.NewStyle().Foreground(colorSuccess)
	errorStyle   = renderer.NewStyle().Foreground(colorError).Bold(true)
	warningStyle = renderer.NewStyle().Foreground(colorWarning)
)

func Interactive() bool {
	descriptor := os.Stderr.Fd()
	return isatty.IsTerminal(descriptor) || isatty.IsCygwinTerminal(descriptor)
}

func Success(format string, args ...any) {
	print("✅", successStyle, format, args...)
}

func Error(format string, args ...any) {
	print("❌", errorStyle, format, args...)
}

func Warn(format string, args ...any) {
	print("⚠️ ", warningStyle, format, args...)
}

func Hint(format string, args ...any) {
	print("💡", Muted, format, args...)
}

func Step(icon string, format string, args ...any) {
	print(icon, Bold, format, args...)
}

func Detail(label string, value string) {
	if value == "" {
		value = Muted.Render("-")
	}
	fmt.Fprintf(os.Stderr, "   %s %s\n", Muted.Render(label+":"), value)
}

func Command(command string) string {
	return Accent.Render(command)
}

func print(icon string, style lipgloss.Style, format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, style.Render(fmt.Sprintf(format, args...)))
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	divisor, exponent := int64(unit), 0
	for value := bytes / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(divisor), "KMGTPE"[exponent])
}
