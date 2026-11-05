package cli

import "testing"

func TestGithubURL(t *testing.T) {
	valid := map[string]string{
		"git@github.com:DasDarki/recipes.git":       "https://github.com/DasDarki/recipes",
		"https://github.com/DasDarki/recipes.git":   "https://github.com/DasDarki/recipes",
		"ssh://git@github.com/DasDarki/recipes.git": "https://github.com/DasDarki/recipes",
		"http://github.com/DasDarki/recipes":        "https://github.com/DasDarki/recipes",
	}
	for remote, expected := range valid {
		actual, err := githubURL(remote)
		if err != nil || actual != expected {
			t.Errorf("githubURL(%q) = %q, %v", remote, actual, err)
		}
	}
	for _, remote := range []string{"https://gitlab.com/a/b.git", "https://github.com/a/b/c"} {
		if _, err := githubURL(remote); err == nil {
			t.Errorf("githubURL(%q) should fail", remote)
		}
	}
}
