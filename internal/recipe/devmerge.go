package recipe

import (
	"maps"
	"slices"
)

func (m *Manifest) WithDev() *Manifest {
	merged := m.clone()
	merged.Dev = nil
	if m.Dev == nil {
		return merged
	}
	overlay := m.Dev.clone()
	if overlay.Server.Branch != "" {
		merged.Server.Branch = overlay.Server.Branch
	}
	merged.Server.NanosSharp = merged.Server.NanosSharp || overlay.Server.NanosSharp
	merged.Server.Args = append(merged.Server.Args, overlay.Server.Args...)
	if merged.Variables == nil {
		merged.Variables = map[string]any{}
	}
	maps.Copy(merged.Variables, overlay.Variables)
	if merged.Config == nil {
		merged.Config = map[string]any{}
	}
	mergeConfig(merged.Config, overlay.Config)
	if merged.Env == nil {
		merged.Env = map[string]any{}
	}
	maps.Copy(merged.Env, overlay.Env)
	for _, resource := range overlay.Packages {
		merged.Packages = putResource(merged.Packages, resource)
	}
	for _, resource := range overlay.Assets {
		merged.Assets = putResource(merged.Assets, resource)
	}
	for _, module := range overlay.Modules {
		merged.Modules = slices.DeleteFunc(merged.Modules, func(existing Module) bool { return existing.Name == module.Name })
		merged.Modules = append(merged.Modules, module)
	}
	for _, file := range overlay.Files {
		merged.Files = slices.DeleteFunc(merged.Files, func(existing File) bool { return existing.Target == file.Target })
		merged.Files = append(merged.Files, file)
	}
	for _, service := range overlay.Services {
		merged.putService(service)
	}
	for _, secret := range overlay.Secrets {
		merged.Secrets = slices.DeleteFunc(merged.Secrets, func(existing Secret) bool { return existing.Name == secret.Name })
		merged.Secrets = append(merged.Secrets, secret)
	}
	merged.Hooks = append(merged.Hooks, overlay.Hooks...)
	for _, tool := range overlay.Tools {
		if !slices.Contains(merged.Tools, tool) {
			merged.Tools = append(merged.Tools, tool)
		}
	}
	return merged
}

func (m *Manifest) WithoutDev() *Manifest {
	plain := m.clone()
	plain.Dev = nil
	return plain
}

func (m *Manifest) clone() *Manifest {
	parsed, err := ParseDSL(RenderDSL(m))
	if err != nil {
		copied := *m
		return &copied
	}
	return parsed
}

func mergeConfig(target map[string]any, values map[string]any) {
	for key, value := range values {
		incoming, incomingIsTable := value.(map[string]any)
		existing, existingIsTable := target[key].(map[string]any)
		if incomingIsTable && existingIsTable {
			mergeConfig(existing, incoming)
			continue
		}
		target[key] = value
	}
}
