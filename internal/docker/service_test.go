package docker

import (
	"path/filepath"
	"testing"
)

func TestPublishArgument(t *testing.T) {
	cases := map[string]string{
		"5432":              "127.0.0.1:5432:5432",
		"15432:5432":        "127.0.0.1:15432:5432",
		"0.0.0.0:6379:6379": "0.0.0.0:6379:6379",
		"7777/udp":          "127.0.0.1:7777:7777/udp",
	}
	for input, expected := range cases {
		if actual := publishArgument(input); actual != expected {
			t.Errorf("publishArgument(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestVolumeArgument(t *testing.T) {
	mapped, named := volumeArgument("Dev", "/srv/dev", "data:/var/lib/postgresql/data")
	if mapped != "nwm-dev-data:/var/lib/postgresql/data" || named != "nwm-dev-data" {
		t.Errorf("named volume mapped to %q (%q)", mapped, named)
	}
	mapped, named = volumeArgument("dev", "/srv/dev", "./init:/docker-entrypoint-initdb.d:ro")
	if mapped != filepath.Join("/srv/dev", "init")+":/docker-entrypoint-initdb.d:ro" || named != "" {
		t.Errorf("host path mapped to %q (%q)", mapped, named)
	}
}

func TestHashChangesWithDefinition(t *testing.T) {
	first := Service{Name: "db", Image: "postgres:16"}
	second := Service{Name: "db", Image: "postgres:17"}
	if first.hash() == second.hash() {
		t.Error("hash should change when the image changes")
	}
}
