package recipe

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/DasDarki/nwm/internal/server"
)

func resolveRecipe(t *testing.T, content string, secrets map[string]string) (*Manifest, error) {
	t.Helper()
	manifest, err := ParseDSL(content)
	if err != nil {
		t.Fatal(err)
	}
	return Resolve(&Fetched{Manifest: manifest}, Target{Name: "dev", Dir: "/srv/dev"}, nil, secrets)
}

func TestEnvReferencesOtherEnvInAnyOrder(t *testing.T) {
	resolved, err := resolveRecipe(t, `recipe x
var host db.local
env FULL ${env.BASE}?sslmode=disable
env BASE postgres://${env.USER}:${secret.pw}@${host}/game
env USER nanos
config custom_settings.database_url ${env.FULL}
hook echo ${env.USER}
`, map[string]string{"pw": "hunter2"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://nanos:hunter2@db.local/game?sslmode=disable"
	if resolved.Env["FULL"] != expected {
		t.Errorf("FULL = %v", resolved.Env["FULL"])
	}
	if resolved.ConfigKeys()["custom_settings.database_url"] != expected {
		t.Errorf("config = %v", resolved.ConfigKeys()["custom_settings.database_url"])
	}
	if resolved.Hooks[0].Run[1] != "nanos" {
		t.Errorf("hook = %v", resolved.Hooks[0].Run)
	}
}

func TestEnvReferencesKeepSecretsOutOfServices(t *testing.T) {
	resolved, err := resolveRecipe(t, `recipe x
env PASSWORD ${secret.pw}
service db postgres:17 env POSTGRES_PASSWORD=${env.PASSWORD} command run ${env.PASSWORD}
`, map[string]string{"pw": "hunter2"})
	if err != nil {
		t.Fatal(err)
	}
	service := resolved.Services[0]
	if service.Env["POSTGRES_PASSWORD"] != "${secret.pw}" {
		t.Errorf("the service definition must keep the placeholder, got %q", service.Env["POSTGRES_PASSWORD"])
	}
	final, err := ResolveServiceSecrets(service, map[string]string{"pw": "hunter2"})
	if err != nil {
		t.Fatal(err)
	}
	if final.Env["POSTGRES_PASSWORD"] != "hunter2" || final.Command[1] != "hunter2" {
		t.Errorf("container values were not resolved: %+v", final)
	}
}

func TestExposeUsesPresetValues(t *testing.T) {
	manifest, err := ParseDSL("recipe x\nuse postgres port 15432\nexpose DATABASE_URL\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.Validate(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(RenderDSL(manifest), "config custom_settings.DATABASE_URL ${env.DATABASE_URL}") {
		t.Errorf("expose should be stored as config:\n%s", RenderDSL(manifest))
	}
	resolved, err := Resolve(&Fetched{Manifest: manifest}, Target{Name: "dev"}, nil, map[string]string{"postgres_password": "pw"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ConfigKeys()["custom_settings.DATABASE_URL"] != "postgres://nanos:pw@127.0.0.1:15432/nanos" {
		t.Errorf("exposed value = %v", resolved.ConfigKeys()["custom_settings.DATABASE_URL"])
	}
}

func TestEnvReferenceErrors(t *testing.T) {
	cases := map[string]string{
		"recipe x\nenv A ${env.B}\nenv B ${env.A}\n":                      "circle",
		"recipe x\nenv A ${env.A}\n":                                      "circle",
		"recipe x\nenv A ${env.MISSING}\n":                                "env.MISSING",
		"recipe x\nexpose MISSING\n":                                      "env.MISSING",
		"recipe x\nconfig game.map ${env.ONLY_DEV}\ndev env ONLY_DEV x\n": "env.ONLY_DEV",
	}
	for content, expected := range cases {
		manifest, err := ParseDSL(content)
		if err != nil {
			t.Fatal(err)
		}
		err = manifest.Validate(t.TempDir())
		if err == nil || !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate(%q) = %v, want %q", content, err, expected)
		}
	}
	manifest, err := ParseDSL("recipe x\ndev env A 1\ndev expose A\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.Validate(t.TempDir()); err != nil {
		t.Errorf("dev statements may use dev env values: %v", err)
	}
}

func TestConfigWithSecretsIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows has no unix file modes")
	}
	for value, expected := range map[string]os.FileMode{"hunter2": 0o600, "public": 0o644} {
		directory := t.TempDir()
		manifest := &Manifest{Config: map[string]any{"custom_settings": map[string]any{"value": value}}}
		if err := writeConfiguration(directory, manifest, map[string]string{"pw": "hunter2"}); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Join(directory, server.ConfigFileName))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != expected {
			t.Errorf("value %q: mode %v, want %v", value, info.Mode().Perm(), expected)
		}
	}
}
