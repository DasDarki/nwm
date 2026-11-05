package recipe

import "testing"

func TestParseSource(t *testing.T) {
	cases := []struct {
		spec    string
		kind    SourceKind
		name    string
		url     string
		subdir  string
		version string
	}{
		{"@dasdarki/rp@^1", SourceRegistry, "@dasdarki/rp", "", "", "^1"},
		{"@dasdarki/rp", SourceRegistry, "@dasdarki/rp", "", "", ""},
		{"rp@1.2.0", SourceRegistry, "rp", "", "", "1.2.0"},
		{"github:DasDarki/recipes/rp/dev@v2", SourceGit, "", "https://github.com/DasDarki/recipes.git", "rp/dev", "v2"},
		{"https://example.com/r.git#v1.0.0", SourceGit, "", "https://example.com/r.git", "", "v1.0.0"},
		{"git+https://example.com/r.git//sub/dir#^1", SourceGit, "", "https://example.com/r.git", "sub/dir", "^1"},
		{"git+file:///srv/recipes//rp", SourceGit, "", "file:///srv/recipes", "rp", ""},
		{"git@github.com:me/r.git#main", SourceGit, "", "git@github.com:me/r.git", "", "main"},
	}
	for _, testCase := range cases {
		source, err := ParseSource(testCase.spec)
		if err != nil {
			t.Fatalf("ParseSource(%q): %v", testCase.spec, err)
		}
		if source.Kind != testCase.kind || source.Name != testCase.name || source.URL != testCase.url || source.Subdir != testCase.subdir || source.Version != testCase.version {
			t.Errorf("ParseSource(%q) = %+v", testCase.spec, source)
		}
	}
}

func TestExpanderKeepsTypes(t *testing.T) {
	expander := NewExpander(map[string]any{"port": int64(7777)}, map[string]any{"port": int64(7790)}, map[string]string{"server.name": "dev"}, nil)
	if value := expander.Value("${port}"); value != int64(7790) {
		t.Errorf("expected typed override, got %#v", value)
	}
	if value := expander.Value("srv-${server.name}:${port}"); value != "srv-dev:7790" {
		t.Errorf("unexpected interpolation %#v", value)
	}
	expander.String("${missing}")
	if expander.Err() == nil {
		t.Error("expected an error for unknown variables")
	}
}
