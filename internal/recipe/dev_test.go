package recipe

import (
	"reflect"
	"strings"
	"testing"
)

const devRecipefile = `recipe @me/gamemode 0.1.0
tool nebra

package my-gamemode path . enable build "nebra build" watch "nebra watch --debounce 150" output out reload full
package my-ui git https://github.com/me/my-ui.git ref v1.0.0 local ../my-ui enable
package sandbox

config general.max_players 8

dev package test-utils path tests enable
dev config debug.log_level 3
dev arg --log_level 3
dev use redis
dev tool dotnet
`

func TestDevRecipefileRoundTrip(t *testing.T) {
	manifest, err := ParseDSL(devRecipefile)
	if err != nil {
		t.Fatal(err)
	}
	game := manifest.Packages[0]
	if game.Build != "nebra build" || game.Watch != "nebra watch --debounce 150" || game.Output != "out" || game.Reload != ReloadFull {
		t.Errorf("package options: %+v", game)
	}
	if manifest.Packages[1].Local != "../my-ui" {
		t.Errorf("local option: %+v", manifest.Packages[1])
	}
	if manifest.Dev == nil || len(manifest.Dev.Packages) != 1 || len(manifest.Dev.Services) != 1 {
		t.Fatalf("dev overlay: %+v", manifest.Dev)
	}
	rendered := RenderDSL(manifest)
	reparsed, err := ParseDSL(rendered)
	if err != nil {
		t.Fatalf("rendered recipe does not parse: %v\n%s", err, rendered)
	}
	if !reflect.DeepEqual(manifest, reparsed) {
		t.Errorf("round trip changed the recipe:\n%s", rendered)
	}
	if !strings.Contains(rendered, "dev package test-utils enable path tests") {
		t.Errorf("dev lines missing:\n%s", rendered)
	}
}

func TestWithDevMergesOverlay(t *testing.T) {
	manifest, err := ParseDSL(devRecipefile)
	if err != nil {
		t.Fatal(err)
	}
	merged := manifest.WithDev()
	if merged.Dev != nil {
		t.Error("merged manifest should have no overlay left")
	}
	if len(merged.Packages) != 4 || merged.Packages[3].Name != "test-utils" {
		t.Errorf("packages: %+v", merged.Names("package"))
	}
	if merged.ConfigKeys()["general.max_players"] != int64(8) || merged.ConfigKeys()["debug.log_level"] != int64(3) {
		t.Errorf("config: %v", merged.ConfigKeys())
	}
	if strings.Join(merged.Server.Args, " ") != "--log_level 3" || len(merged.Services) != 1 || len(merged.Tools) != 2 {
		t.Errorf("args, services or tools: %+v %d %v", merged.Server.Args, len(merged.Services), merged.Tools)
	}
	plain := manifest.WithoutDev()
	if plain.Dev != nil || len(plain.Packages) != 3 || len(plain.Services) != 0 {
		t.Errorf("production view should ignore dev lines: %+v", plain.Names("package"))
	}
}

func TestDevRejectsMetadataAndRemoves(t *testing.T) {
	if _, err := ParseDSL("recipe x\ndev recipe y\n"); err == nil {
		t.Error("dev recipe should be rejected")
	}
	manifest, _ := ParseDSL(devRecipefile)
	if err := manifest.Remove("dev", "package test-utils"); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Dev.Packages) != 0 {
		t.Error("dev package was not removed")
	}
}
