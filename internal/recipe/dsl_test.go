package recipe

import (
	"reflect"
	"strings"
	"testing"
)

const sampleRecipefile = `# sample
recipe @dasdarki/roleplay 1.0.0
description "Roleplay dev server"
author DasDarki
keywords roleplay postgres

branch public
nanossharp
arg --log_level 2

var port 7777
var label "7777"
var ids [1, 2, 3]

cfg discover.name "RP ${server.name}"
config discover.port ${port}
config custom_settings.spawn_cars true

pkg sandbox enable
package rp-core git https://github.com/me/rp-core.git ref v2.1.0 enable
package dev-tools path packages/dev-tools
asset nanos-world

mod mysql linux https://example.com/mysql.so windows "C:\\mods\\mysql.dll" enable
file files/NanosSharp.toml NanosSharp.toml template

use postgres
requires redis port 16379
svc search getmeili/meilisearch:v1 port 7700 env MEILI_ENV=development volume search-data:/meili_data ready "curl -f localhost:7700/health" nowait command meilisearch --no-analytics

secret api_key random 48
secret steam_token prompt
env API_KEY "abc 123"
env QUOTE 'it"s'
hook in Packages/rp-core dotnet build
hook echo done
`

func TestParseDSL(t *testing.T) {
	manifest, err := ParseDSL(sampleRecipefile)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Recipe.Name != "@dasdarki/roleplay" || manifest.Recipe.Version != "1.0.0" {
		t.Errorf("recipe header: %+v", manifest.Recipe)
	}
	if manifest.Variables["port"] != int64(7777) || manifest.Variables["label"] != "7777" {
		t.Errorf("variable types: %#v", manifest.Variables)
	}
	if manifest.ConfigKeys()["discover.port"] != "${port}" || manifest.ConfigKeys()["custom_settings.spawn_cars"] != true {
		t.Errorf("config: %#v", manifest.ConfigKeys())
	}
	if len(manifest.Packages) != 3 || !manifest.Packages[1].Enable || manifest.Packages[1].Ref != "v2.1.0" {
		t.Errorf("packages: %+v", manifest.Packages)
	}
	if manifest.Modules[0].Windows != `C:\mods\mysql.dll` || !manifest.Modules[0].Enable {
		t.Errorf("module escape: %q", manifest.Modules[0].Windows)
	}
	if len(manifest.Services) != 3 {
		t.Fatalf("services: %+v", manifest.Services)
	}
	postgres := manifest.Services[0]
	if postgres.Image != "postgres:17" || postgres.Env["POSTGRES_PASSWORD"] != "${secret.postgres_password}" {
		t.Errorf("postgres preset: %+v", postgres)
	}
	if manifest.Env["DATABASE_URL"] != "postgres://nanos:${secret.postgres_password}@127.0.0.1:5432/nanos" {
		t.Errorf("database url: %v", manifest.Env["DATABASE_URL"])
	}
	if manifest.Services[1].Ports[0] != "16379:6379" || manifest.Env["REDIS_URL"] != "redis://127.0.0.1:16379" {
		t.Errorf("redis preset: %+v %v", manifest.Services[1], manifest.Env["REDIS_URL"])
	}
	search := manifest.Services[2]
	if strings.Join(search.Ready, " ") != "curl -f localhost:7700/health" || len(postgres.Ready) == 0 {
		t.Errorf("ready checks: %v / %v", search.Ready, postgres.Ready)
	}
	if search.Wait == nil || *search.Wait || strings.Join(search.Command, " ") != "meilisearch --no-analytics" {
		t.Errorf("service options: %+v", search)
	}
	secretNames := manifest.Names("secret")
	if strings.Join(secretNames, ",") != "postgres_password,api_key,steam_token" {
		t.Errorf("secrets: %v", secretNames)
	}
	if manifest.Env["QUOTE"] != `it"s` || manifest.Hooks[0].Dir != "Packages/rp-core" {
		t.Errorf("env or hooks: %v %+v", manifest.Env, manifest.Hooks)
	}
}

func TestRenderRoundTrip(t *testing.T) {
	manifest, err := ParseDSL(sampleRecipefile)
	if err != nil {
		t.Fatal(err)
	}
	rendered := RenderDSL(manifest)
	reparsed, err := ParseDSL(rendered)
	if err != nil {
		t.Fatalf("rendered recipe does not parse: %v\n%s", err, rendered)
	}
	if !reflect.DeepEqual(manifest, reparsed) {
		t.Errorf("round trip changed the recipe:\n%s", rendered)
	}
	if RenderDSL(reparsed) != rendered {
		t.Error("rendering is not stable")
	}
}

func TestParseErrorsCarryLineNumbers(t *testing.T) {
	_, err := ParseDSL("recipe x\npackage\nfrobnicate y\nbranch nightly\n")
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, expected := range []string{"line 2", "line 3: unknown command", "line 4"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error should mention %q:\n%v", expected, err)
		}
	}
}

func TestRemoveAndAliases(t *testing.T) {
	manifest, err := ParseDSL(sampleRecipefile)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.Remove("pkg", "sandbox"); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Remove("config", "discover.port"); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Remove("package", "missing"); err == nil {
		t.Error("removing an unknown package should fail")
	}
	if len(manifest.Packages) != 2 || manifest.ConfigKeys()["discover.port"] != nil {
		t.Errorf("remove failed: %+v %v", manifest.Packages, manifest.ConfigKeys())
	}
}
