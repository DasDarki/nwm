package recipe

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/DasDarki/nwm/internal/docker"
	"github.com/DasDarki/nwm/internal/server"
)

const DSLFileName = "Recipefile"

type Statement struct {
	Keyword string
	Aliases []string
	Usage   string
	Summary string
	Options []string
	apply   func(m *Manifest, args []token) error
}

var statements = []Statement{
	{Keyword: "recipe", Usage: "recipe <name> [version]", Summary: "name and version of the recipe", apply: applyRecipe},
	{Keyword: "description", Usage: "description <text>", Summary: "short description", apply: func(m *Manifest, args []token) error {
		m.Recipe.Description = joinTokens(args)
		return nil
	}},
	{Keyword: "author", Usage: "author <name>...", Summary: "add authors", apply: func(m *Manifest, args []token) error {
		for _, arg := range args {
			if !slices.Contains(m.Recipe.Authors, arg.Text) {
				m.Recipe.Authors = append(m.Recipe.Authors, arg.Text)
			}
		}
		return requireArgs(args, 1, "author <name>...")
	}},
	{Keyword: "keywords", Usage: "keywords <word>...", Summary: "search keywords", apply: func(m *Manifest, args []token) error {
		for _, arg := range args {
			if !slices.Contains(m.Recipe.Keywords, arg.Text) {
				m.Recipe.Keywords = append(m.Recipe.Keywords, arg.Text)
			}
		}
		return requireArgs(args, 1, "keywords <word>...")
	}},
	{Keyword: "branch", Usage: "branch public|bleeding-edge", Summary: "steam branch of the server", Options: []string{"public", "bleeding-edge"}, apply: func(m *Manifest, args []token) error {
		if err := requireArgs(args, 1, "branch public|bleeding-edge"); err != nil {
			return err
		}
		if args[0].Text != "public" && args[0].Text != "bleeding-edge" {
			return fmt.Errorf("branch must be public or bleeding-edge")
		}
		m.Server.Branch = args[0].Text
		return nil
	}},
	{Keyword: "nanossharp", Usage: "nanossharp [on|off]", Summary: "install NanosSharp", Options: []string{"on", "off"}, apply: func(m *Manifest, args []token) error {
		m.Server.NanosSharp = len(args) == 0 || args[0].Text == "on" || args[0].Text == "true"
		return nil
	}},
	{Keyword: "arg", Usage: "arg <argument>...", Summary: "server launch arguments", apply: func(m *Manifest, args []token) error {
		for _, arg := range args {
			m.Server.Args = append(m.Server.Args, arg.Text)
		}
		return requireArgs(args, 1, "arg <argument>...")
	}},
	{Keyword: "var", Usage: "var <name> <default>", Summary: "variable usable as ${name}", apply: func(m *Manifest, args []token) error {
		if err := requireArgs(args, 2, "var <name> <default>"); err != nil {
			return err
		}
		if m.Variables == nil {
			m.Variables = map[string]any{}
		}
		m.Variables[args[0].Text] = args[1].value()
		return nil
	}},
	{Keyword: "config", Aliases: []string{"cfg"}, Usage: "config <key> <value>", Summary: "set a Config.toml value, e.g. discover.port", apply: func(m *Manifest, args []token) error {
		if err := requireArgs(args, 2, "config <key> <value>"); err != nil {
			return err
		}
		if m.Config == nil {
			m.Config = map[string]any{}
		}
		return setNested(m.Config, args[0].Text, args[1].value())
	}},
	{Keyword: "package", Aliases: []string{"pkg"}, Usage: "package <name> [enable] [git <url>] [ref <ref>] [subdir <dir>] [path <dir>] [local <dir>] [build \"<cmd>\"] [watch \"<cmd>\"] [output <dir>] [reload hot|full]", Summary: "add a package (vault, git or recipe folder)", Options: resourceOptions, apply: func(m *Manifest, args []token) error {
		resource, err := parseResource(args, "package")
		if err != nil {
			return err
		}
		m.Packages = putResource(m.Packages, resource)
		return nil
	}},
	{Keyword: "assets", Aliases: []string{"asset"}, Usage: "assets <name> [enable] [git <url>] [ref <ref>] [subdir <dir>] [path <dir>] [local <dir>]", Summary: "add an asset pack", Options: resourceOptions, apply: func(m *Manifest, args []token) error {
		resource, err := parseResource(args, "assets")
		if err != nil {
			return err
		}
		m.Assets = putResource(m.Assets, resource)
		return nil
	}},
	{Keyword: "module", Aliases: []string{"mod"}, Usage: "module <name> [linux <source>] [windows <source>] [enable]", Summary: "add a C module package from prebuilt .so/.dll files or archives", Options: []string{"linux", "windows", "enable"}, apply: applyModule},
	{Keyword: "file", Usage: "file <source> <target> [template]", Summary: "copy a recipe file into the server", Options: []string{"template"}, apply: func(m *Manifest, args []token) error {
		if err := requireArgs(args, 2, "file <source> <target> [template]"); err != nil {
			return err
		}
		file := File{Source: args[0].Text, Target: args[1].Text}
		for _, option := range args[2:] {
			if option.Text != "template" {
				return fmt.Errorf("unknown option %q for file, expected template", option.Text)
			}
			file.Template = true
		}
		m.Files = slices.DeleteFunc(m.Files, func(existing File) bool { return existing.Target == file.Target })
		m.Files = append(m.Files, file)
		return nil
	}},
	{Keyword: "service", Aliases: []string{"svc"}, Usage: "service <name> <image> [port <p>]... [env K=V]... [volume <v>]... [ready \"<check>\"] [nowait] [command <args>...]", Summary: "add a Docker service", Options: []string{"port", "env", "volume", "ready", "nowait", "command"}, apply: applyService},
	{Keyword: "use", Aliases: []string{"require_service", "requires", "require"}, Usage: "use <image> [as <name>] [port <host port>] [env K=V]...", Summary: "add a service with presets (postgres, mysql, redis, ...), its secrets and .env values", Options: []string{"as", "port", "env"}, apply: applyRequirement},
	{Keyword: "secret", Usage: "secret <name> [random [length]] | [prompt]", Summary: "a value from the server's secret store, used as ${secret.name}", Options: []string{"random", "prompt"}, apply: applySecret},
	{Keyword: "env", Usage: "env <KEY> <value>", Summary: "write a value to the server's .env", apply: func(m *Manifest, args []token) error {
		if err := requireArgs(args, 2, "env <KEY> <value>"); err != nil {
			return err
		}
		if m.Env == nil {
			m.Env = map[string]any{}
		}
		m.Env[args[0].Text] = joinTokens(args[1:])
		return nil
	}},
	{Keyword: "tool", Usage: "tool <name>...", Summary: "a program the recipe needs, e.g. nebra or dotnet", apply: func(m *Manifest, args []token) error {
		for _, arg := range args {
			if !slices.Contains(m.Tools, arg.Text) {
				m.Tools = append(m.Tools, arg.Text)
			}
		}
		return requireArgs(args, 1, "tool <name>...")
	}},
	{Keyword: "dev", Usage: "dev <statement>", Summary: "a statement that only applies in nwm dev, e.g. dev package test-utils path tests/utils"},
	{Keyword: "hook", Usage: "hook [in <dir>] <command>...", Summary: "run a command after applying", apply: func(m *Manifest, args []token) error {
		hook := Hook{}
		if len(args) >= 2 && args[0].Text == "in" && !args[0].Quoted {
			hook.Dir = args[1].Text
			args = args[2:]
		}
		if err := requireArgs(args, 1, "hook [in <dir>] <command>..."); err != nil {
			return err
		}
		for _, arg := range args {
			hook.Run = append(hook.Run, arg.Text)
		}
		m.Hooks = append(m.Hooks, hook)
		return nil
	}},
}

var resourceOptions = []string{"enable", "git", "ref", "subdir", "path", "local", "build", "watch", "output", "reload"}

var devForbidden = []string{"recipe", "description", "author", "keywords", "dev"}

func init() {
	for index := range statements {
		if statements[index].Keyword == "dev" {
			statements[index].apply = applyDev
		}
	}
}

func applyDev(m *Manifest, args []token) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: dev <statement>")
	}
	statement, found := findStatement(args[0].Text)
	if !found {
		return fmt.Errorf("unknown command %q", args[0].Text)
	}
	if slices.Contains(devForbidden, statement.Keyword) {
		return fmt.Errorf("%s cannot be used with dev", statement.Keyword)
	}
	if m.Dev == nil {
		m.Dev = &Manifest{}
	}
	return statement.apply(m.Dev, args[1:])
}

func Statements() []Statement {
	return statements
}

func findStatement(keyword string) (*Statement, bool) {
	for index := range statements {
		if statements[index].Keyword == keyword || slices.Contains(statements[index].Aliases, keyword) {
			return &statements[index], true
		}
	}
	return nil, false
}

func (m *Manifest) ApplyLine(line string) error {
	parsed, err := tokenize(line)
	if err != nil {
		return err
	}
	if len(parsed.tokens) == 0 {
		return nil
	}
	keyword := parsed.tokens[0].Text
	statement, found := findStatement(keyword)
	if !found {
		return fmt.Errorf("unknown command %q", keyword)
	}
	return statement.apply(m, parsed.tokens[1:])
}

func ParseDSL(content string) (*Manifest, error) {
	manifest := &Manifest{}
	var problems []string
	for number, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		if err := manifest.ApplyLine(line); err != nil {
			problems = append(problems, fmt.Sprintf("line %d: %v", number+1, err))
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s:\n  - %s", DSLFileName, strings.Join(problems, "\n  - "))
	}
	return manifest, nil
}

func applyRecipe(m *Manifest, args []token) error {
	if err := requireArgs(args, 1, "recipe <name> [version]"); err != nil {
		return err
	}
	m.Recipe.Name = strings.ToLower(args[0].Text)
	if len(args) > 1 {
		m.Recipe.Version = strings.TrimPrefix(args[1].Text, "v")
	}
	return nil
}

func parseResource(args []token, kind string) (Resource, error) {
	var resource Resource
	if len(args) == 0 {
		return resource, fmt.Errorf("usage: %s <name> [enable] [git <url>] [ref <ref>] [subdir <dir>] [path <dir>]", kind)
	}
	resource.Name = args[0].Text
	for index := 1; index < len(args); index++ {
		option := args[index].Text
		if option == "enable" {
			resource.Enable = true
			continue
		}
		if index+1 >= len(args) {
			return resource, fmt.Errorf("%s: %q needs a value", kind, option)
		}
		value := args[index+1].Text
		index++
		switch option {
		case "git":
			resource.Git = value
		case "ref":
			resource.Ref = value
		case "subdir":
			resource.Subdir = value
		case "path":
			resource.Path = value
		case "local":
			resource.Local = value
		case "build":
			resource.Build = value
		case "watch":
			resource.Watch = value
		case "output":
			resource.Output = value
		case "reload":
			if value != ReloadHot && value != ReloadFull {
				return resource, fmt.Errorf("%s %s: reload must be hot or full", kind, resource.Name)
			}
			resource.Reload = value
		default:
			return resource, fmt.Errorf("unknown option %q for %s, expected %s", option, kind, strings.Join(resourceOptions, ", "))
		}
	}
	if resource.Git != "" && resource.Path != "" {
		return resource, fmt.Errorf("%s %s cannot use git and path together", kind, resource.Name)
	}
	return resource, nil
}

func putResource(resources []Resource, resource Resource) []Resource {
	for index := range resources {
		if resources[index].Name == resource.Name {
			resources[index] = resource
			return resources
		}
	}
	return append(resources, resource)
}

func applyModule(m *Manifest, args []token) error {
	if err := requireArgs(args, 1, "module <name> [linux <source>] [windows <source>] [enable]"); err != nil {
		return err
	}
	module := Module{Name: args[0].Text}
	for index := 1; index < len(args); index += 2 {
		if args[index].Text == "enable" {
			module.Enable = true
			index--
			continue
		}
		if index+1 >= len(args) {
			return fmt.Errorf("module: %q needs a source", args[index].Text)
		}
		switch args[index].Text {
		case "linux":
			module.Linux = args[index+1].Text
		case "windows":
			module.Windows = args[index+1].Text
		default:
			return fmt.Errorf("unknown option %q for module, expected linux, windows or enable", args[index].Text)
		}
	}
	m.Modules = slices.DeleteFunc(m.Modules, func(existing Module) bool { return existing.Name == module.Name })
	m.Modules = append(m.Modules, module)
	return nil
}

func applyService(m *Manifest, args []token) error {
	if err := requireArgs(args, 2, "service <name> <image> [options]"); err != nil {
		return err
	}
	service := docker.Service{Name: args[0].Text, Image: args[1].Text}
	for index := 2; index < len(args); index++ {
		option := args[index].Text
		switch option {
		case "nowait":
			wait := false
			service.Wait = &wait
			continue
		case "command":
			for _, arg := range args[index+1:] {
				service.Command = append(service.Command, arg.Text)
			}
			index = len(args)
			continue
		}
		if index+1 >= len(args) {
			return fmt.Errorf("service: %q needs a value", option)
		}
		value := args[index+1].Text
		index++
		switch option {
		case "port":
			service.Ports = append(service.Ports, value)
		case "volume":
			service.Volumes = append(service.Volumes, value)
		case "ready":
			service.Ready = strings.Fields(value)
		case "env":
			key, envValue, found := strings.Cut(value, "=")
			if !found {
				return fmt.Errorf("service env expects KEY=value, got %q", value)
			}
			if service.Env == nil {
				service.Env = map[string]string{}
			}
			service.Env[key] = envValue
		default:
			return fmt.Errorf("unknown option %q for service, expected port, env, volume, ready, nowait or command", option)
		}
	}
	if err := service.Validate(); err != nil {
		return err
	}
	m.putService(service)
	return nil
}

func applyRequirement(m *Manifest, args []token) error {
	if err := requireArgs(args, 1, "use <image> [as <name>] [port <host port>] [env K=V]..."); err != nil {
		return err
	}
	image := args[0].Text
	var name, hostPort string
	extraEnv := map[string]string{}
	for index := 1; index < len(args); index += 2 {
		if index+1 >= len(args) {
			return fmt.Errorf("use: %q needs a value", args[index].Text)
		}
		value := args[index+1].Text
		switch args[index].Text {
		case "as":
			name = value
		case "port":
			hostPort = value
		case "env":
			key, envValue, found := strings.Cut(value, "=")
			if !found {
				return fmt.Errorf("use env expects KEY=value, got %q", value)
			}
			extraEnv[key] = envValue
		default:
			return fmt.Errorf("unknown option %q for use, expected as, port or env", args[index].Text)
		}
	}
	service, serverEnv, secret := expandRequirement(image, name, hostPort, extraEnv)
	if err := service.Validate(); err != nil {
		return err
	}
	m.putService(service)
	if secret != nil && !slices.ContainsFunc(m.Secrets, func(existing Secret) bool { return existing.Name == secret.Name }) {
		m.Secrets = append(m.Secrets, *secret)
	}
	if len(serverEnv) > 0 && m.Env == nil {
		m.Env = map[string]any{}
	}
	for key, value := range serverEnv {
		if _, exists := m.Env[key]; !exists {
			m.Env[key] = value
		}
	}
	return nil
}

func applySecret(m *Manifest, args []token) error {
	usage := "secret <name> [random [length]] | [prompt]"
	if err := requireArgs(args, 1, usage); err != nil {
		return err
	}
	secret := Secret{Name: args[0].Text}
	rest := args[1:]
	switch {
	case len(rest) == 0:
	case rest[0].Text == "prompt" && len(rest) == 1:
		secret.Prompt = true
	case rest[0].Text == "random" && len(rest) <= 2:
		if len(rest) == 2 {
			length, isNumber := rest[1].value().(int64)
			if !isNumber {
				return fmt.Errorf("secret length must be a number, got %q", rest[1].Text)
			}
			secret.Length = int(length)
		}
	default:
		return fmt.Errorf("usage: %s", usage)
	}
	if err := secret.Validate(); err != nil {
		return err
	}
	m.Secrets = slices.DeleteFunc(m.Secrets, func(existing Secret) bool { return existing.Name == secret.Name })
	m.Secrets = append(m.Secrets, secret)
	return nil
}

func (m *Manifest) putService(service docker.Service) {
	for index := range m.Services {
		if m.Services[index].Name == service.Name {
			m.Services[index] = service
			return
		}
	}
	m.Services = append(m.Services, service)
}

func setNested(table map[string]any, key string, value any) error {
	parts := strings.Split(key, ".")
	for _, part := range parts[:len(parts)-1] {
		next, found := table[part]
		if !found {
			created := map[string]any{}
			table[part] = created
			table = created
			continue
		}
		nested, isTable := next.(map[string]any)
		if !isTable {
			return fmt.Errorf("%s is not a table", part)
		}
		table = nested
	}
	table[parts[len(parts)-1]] = value
	return nil
}

func deleteNested(table map[string]any, key string) bool {
	parts := strings.Split(key, ".")
	for _, part := range parts[:len(parts)-1] {
		nested, isTable := table[part].(map[string]any)
		if !isTable {
			return false
		}
		table = nested
	}
	last := parts[len(parts)-1]
	if _, found := table[last]; !found {
		return false
	}
	delete(table, last)
	return true
}

func flatten(prefix string, table map[string]any, into map[string]any) {
	for key, value := range table {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if nested, isTable := value.(map[string]any); isTable && len(nested) > 0 {
			flatten(path, nested, into)
			continue
		}
		into[path] = value
	}
}

func UnknownConfigKey(line string) (string, bool) {
	parsed, err := tokenize(line)
	if err != nil || len(parsed.tokens) < 2 {
		return "", false
	}
	statement, found := findStatement(parsed.tokens[0].Text)
	if !found || statement.Keyword != "config" {
		return "", false
	}
	key := parsed.tokens[1].Text
	if strings.HasPrefix(key, "custom_settings.") {
		return "", false
	}
	_, known := server.DefaultConfigKeys()[key]
	return key, !known
}

func (m *Manifest) ConfigKeys() map[string]any {
	flat := map[string]any{}
	flatten("", m.Config, flat)
	return flat
}

func requireArgs(args []token, count int, usage string) error {
	if len(args) < count {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
}

func joinTokens(args []token) string {
	parts := make([]string, len(args))
	for index, arg := range args {
		parts[index] = arg.Text
	}
	return strings.Join(parts, " ")
}

func RenderDSL(m *Manifest) string {
	sections := renderSections(m, true)
	if m.Dev != nil {
		for _, section := range renderSections(m.Dev, false) {
			prefixed := make([]string, len(section))
			for index, line := range section {
				prefixed[index] = "dev " + line
			}
			sections = append(sections, prefixed)
		}
	}
	blocks := make([]string, len(sections))
	for index, section := range sections {
		blocks[index] = strings.Join(section, "\n")
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

func renderSections(m *Manifest, withHeader bool) [][]string {
	var sections [][]string
	add := func(lines ...string) {
		var filtered []string
		for _, line := range lines {
			if line != "" {
				filtered = append(filtered, line)
			}
		}
		if len(filtered) > 0 {
			sections = append(sections, filtered)
		}
	}
	join := func(words ...string) string {
		var parts []string
		for _, word := range words {
			if word != "" {
				parts = append(parts, word)
			}
		}
		return strings.Join(parts, " ")
	}

	var header []string
	if withHeader {
		header = append(header, join("recipe", QuoteWord(m.Recipe.Name), m.Recipe.Version))
	}
	if m.Recipe.Description != "" {
		header = append(header, "description "+QuoteWord(m.Recipe.Description))
	}
	if len(m.Recipe.Authors) > 0 {
		header = append(header, "author "+quoteWords(m.Recipe.Authors))
	}
	if len(m.Recipe.Keywords) > 0 {
		header = append(header, "keywords "+quoteWords(m.Recipe.Keywords))
	}
	if len(m.Tools) > 0 {
		header = append(header, "tool "+quoteWords(m.Tools))
	}
	add(header...)

	var serverLines []string
	if m.Server.Branch != "" {
		serverLines = append(serverLines, "branch "+m.Server.Branch)
	}
	if m.Server.NanosSharp {
		serverLines = append(serverLines, "nanossharp")
	}
	if len(m.Server.Args) > 0 {
		serverLines = append(serverLines, "arg "+quoteWords(m.Server.Args))
	}
	add(serverLines...)

	var variableLines []string
	for _, name := range slices.Sorted(maps.Keys(m.Variables)) {
		variableLines = append(variableLines, "var "+name+" "+formatValue(m.Variables[name]))
	}
	add(variableLines...)

	flat := m.ConfigKeys()
	var configLines []string
	for _, key := range slices.Sorted(maps.Keys(flat)) {
		configLines = append(configLines, "config "+key+" "+formatValue(flat[key]))
	}
	add(configLines...)

	add(formatResources("package", m.Packages)...)
	add(formatResources("assets", m.Assets)...)

	var moduleLines []string
	for _, module := range m.Modules {
		line := "module " + QuoteWord(module.Name)
		if module.Linux != "" {
			line += " linux " + QuoteWord(module.Linux)
		}
		if module.Windows != "" {
			line += " windows " + QuoteWord(module.Windows)
		}
		if module.Enable {
			line += " enable"
		}
		moduleLines = append(moduleLines, line)
	}
	add(moduleLines...)

	var fileLines []string
	for _, file := range m.Files {
		line := "file " + QuoteWord(file.Source) + " " + QuoteWord(file.Target)
		if file.Template {
			line += " template"
		}
		fileLines = append(fileLines, line)
	}
	add(fileLines...)

	var serviceLines []string
	for _, service := range m.Services {
		line := "service " + QuoteWord(service.Name) + " " + QuoteWord(service.Image)
		for _, port := range service.Ports {
			line += " port " + QuoteWord(port)
		}
		for _, key := range slices.Sorted(maps.Keys(service.Env)) {
			line += " env " + QuoteWord(key+"="+service.Env[key])
		}
		for _, volume := range service.Volumes {
			line += " volume " + QuoteWord(volume)
		}
		if len(service.Ready) > 0 {
			line += " ready " + QuoteWord(strings.Join(service.Ready, " "))
		}
		if service.Wait != nil && !*service.Wait {
			line += " nowait"
		}
		if len(service.Command) > 0 {
			line += " command " + quoteWords(service.Command)
		}
		serviceLines = append(serviceLines, line)
	}
	add(serviceLines...)

	var secretLines []string
	for _, secret := range m.Secrets {
		line := "secret " + secret.Name
		switch {
		case secret.Prompt:
			line += " prompt"
		case secret.Length > 0:
			line += fmt.Sprintf(" random %d", secret.Length)
		}
		secretLines = append(secretLines, line)
	}
	add(secretLines...)

	var envLines []string
	for _, key := range slices.Sorted(maps.Keys(m.Env)) {
		envLines = append(envLines, "env "+QuoteWord(key)+" "+QuoteWord(fmt.Sprint(m.Env[key])))
	}
	add(envLines...)

	var hookLines []string
	for _, hook := range m.Hooks {
		line := "hook "
		if hook.Dir != "" {
			line += "in " + QuoteWord(hook.Dir) + " "
		}
		hookLines = append(hookLines, line+quoteWords(hook.Run))
	}
	add(hookLines...)

	return sections
}

func formatResources(keyword string, resources []Resource) []string {
	lines := make([]string, 0, len(resources))
	for _, resource := range resources {
		line := keyword + " " + QuoteWord(resource.Name)
		if resource.Enable {
			line += " enable"
		}
		for _, pair := range [][2]string{{"git", resource.Git}, {"ref", resource.Ref}, {"subdir", resource.Subdir}, {"path", resource.Path}, {"local", resource.Local}, {"build", resource.Build}, {"watch", resource.Watch}, {"output", resource.Output}, {"reload", resource.Reload}} {
			if pair[1] != "" {
				line += " " + pair[0] + " " + QuoteWord(pair[1])
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func quoteWords(words []string) string {
	quoted := make([]string, len(words))
	for index, word := range words {
		quoted[index] = QuoteWord(word)
	}
	return strings.Join(quoted, " ")
}

var RemovableKinds = []string{"package", "assets", "module", "file", "service", "secret", "env", "var", "config", "hook", "arg", "author", "keywords", "tool", "dev"}

func canonicalKind(kind string) string {
	if statement, found := findStatement(kind); found {
		return statement.Keyword
	}
	return kind
}

func (m *Manifest) Remove(kind string, name string) error {
	removed := false
	kind = canonicalKind(kind)
	if kind == "dev" {
		innerKind, innerName, found := strings.Cut(name, " ")
		if !found || m.Dev == nil {
			return fmt.Errorf("usage: remove dev <kind> <name>")
		}
		if err := m.Dev.Remove(innerKind, innerName); err != nil {
			return fmt.Errorf("dev %w", err)
		}
		if RenderDSL(m.Dev) == "\n" || strings.TrimSpace(RenderDSL(m.Dev)) == "" {
			m.Dev = nil
		}
		return nil
	}
	switch kind {
	case "tool":
		before := len(m.Tools)
		m.Tools = slices.DeleteFunc(m.Tools, func(tool string) bool { return tool == name })
		removed = len(m.Tools) != before
	case "package":
		before := len(m.Packages)
		m.Packages = slices.DeleteFunc(m.Packages, func(r Resource) bool { return r.Name == name })
		removed = len(m.Packages) != before
	case "assets":
		before := len(m.Assets)
		m.Assets = slices.DeleteFunc(m.Assets, func(r Resource) bool { return r.Name == name })
		removed = len(m.Assets) != before
	case "module":
		before := len(m.Modules)
		m.Modules = slices.DeleteFunc(m.Modules, func(module Module) bool { return module.Name == name })
		removed = len(m.Modules) != before
	case "file":
		before := len(m.Files)
		m.Files = slices.DeleteFunc(m.Files, func(file File) bool { return file.Target == name || file.Source == name })
		removed = len(m.Files) != before
	case "service":
		before := len(m.Services)
		m.Services = slices.DeleteFunc(m.Services, func(service docker.Service) bool { return service.Name == name })
		removed = len(m.Services) != before
	case "secret":
		before := len(m.Secrets)
		m.Secrets = slices.DeleteFunc(m.Secrets, func(secret Secret) bool { return secret.Name == name })
		removed = len(m.Secrets) != before
	case "env":
		_, removed = m.Env[name]
		delete(m.Env, name)
	case "var":
		_, removed = m.Variables[name]
		delete(m.Variables, name)
	case "config":
		removed = deleteNested(m.Config, name)
	case "hook":
		before := len(m.Hooks)
		m.Hooks = slices.DeleteFunc(m.Hooks, func(hook Hook) bool { return strings.Join(hook.Run, " ") == name })
		removed = len(m.Hooks) != before
	case "arg":
		before := len(m.Server.Args)
		m.Server.Args = slices.DeleteFunc(m.Server.Args, func(arg string) bool { return arg == name })
		removed = len(m.Server.Args) != before
	case "author":
		before := len(m.Recipe.Authors)
		m.Recipe.Authors = slices.DeleteFunc(m.Recipe.Authors, func(author string) bool { return author == name })
		removed = len(m.Recipe.Authors) != before
	case "keywords":
		before := len(m.Recipe.Keywords)
		m.Recipe.Keywords = slices.DeleteFunc(m.Recipe.Keywords, func(keyword string) bool { return keyword == name })
		removed = len(m.Recipe.Keywords) != before
	default:
		return fmt.Errorf("cannot remove %q, expected one of %s", kind, strings.Join(RemovableKinds, ", "))
	}
	if !removed {
		return fmt.Errorf("%s %q not found", kind, name)
	}
	return nil
}

func (m *Manifest) Names(kind string) []string {
	var names []string
	switch canonicalKind(kind) {
	case "tool":
		names = m.Tools
	case "dev":
		if m.Dev != nil {
			for _, inner := range RemovableKinds {
				if inner != "dev" && len(m.Dev.Names(inner)) > 0 {
					names = append(names, inner)
				}
			}
		}
	case "package":
		for _, resource := range m.Packages {
			names = append(names, resource.Name)
		}
	case "assets":
		for _, resource := range m.Assets {
			names = append(names, resource.Name)
		}
	case "module":
		for _, module := range m.Modules {
			names = append(names, module.Name)
		}
	case "file":
		for _, file := range m.Files {
			names = append(names, file.Target)
		}
	case "service":
		for _, service := range m.Services {
			names = append(names, service.Name)
		}
	case "secret":
		for _, secret := range m.Secrets {
			names = append(names, secret.Name)
		}
	case "env":
		names = slices.Sorted(maps.Keys(m.Env))
	case "var":
		names = slices.Sorted(maps.Keys(m.Variables))
	case "config":
		names = slices.Sorted(maps.Keys(m.ConfigKeys()))
	case "hook":
		for _, hook := range m.Hooks {
			names = append(names, strings.Join(hook.Run, " "))
		}
	case "arg":
		names = m.Server.Args
	case "author":
		names = m.Recipe.Authors
	case "keywords":
		names = m.Recipe.Keywords
	}
	return names
}
