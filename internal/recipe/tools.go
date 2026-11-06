package recipe

import (
	"fmt"
	"os/exec"
	"strings"
)

var toolHints = map[string]string{
	"nebra":  "https://nebra-lang.github.io/docs",
	"git":    "https://git-scm.com/downloads",
	"docker": "https://docs.docker.com/get-docker/",
	"dotnet": "https://dotnet.microsoft.com/download",
	"node":   "https://nodejs.org",
	"npm":    "https://nodejs.org",
}

func MissingTools(tools []string) []string {
	var missing []string
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	return missing
}

func MissingToolsError(missing []string) error {
	lines := make([]string, len(missing))
	for index, tool := range missing {
		lines[index] = tool
		if hint, known := toolHints[tool]; known {
			lines[index] += " (" + hint + ")"
		}
	}
	return fmt.Errorf("this recipe needs tools that are not installed: %s", strings.Join(lines, ", "))
}

func KnownTools() []string {
	tools := make([]string, 0, len(toolHints))
	for tool := range toolHints {
		tools = append(tools, tool)
	}
	return tools
}
