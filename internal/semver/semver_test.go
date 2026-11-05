package semver

import "testing"

func TestConstraintMatch(t *testing.T) {
	cases := []struct {
		constraint string
		version    string
		expected   bool
	}{
		{"", "v1.2.3", true},
		{"", "v1.3.0-rc.1", false},
		{"^1", "v1.9.0", true},
		{"^1", "v2.0.0", false},
		{"^1.2", "v1.1.0", false},
		{"^0.3", "v0.3.5", true},
		{"^0.3", "v0.4.0", false},
		{"~1.2", "v1.2.9", true},
		{"~1.2", "v1.3.0", false},
		{"1.2.3", "v1.2.3", true},
		{"v1.2.3", "1.2.4", false},
		{"1.2", "v1.2.7", true},
		{">=1.0", "v3.0.0", true},
		{"1.0.0-beta", "v1.0.0-beta", true},
	}
	for _, testCase := range cases {
		constraint, err := ParseConstraint(testCase.constraint)
		if err != nil {
			t.Fatalf("ParseConstraint(%q): %v", testCase.constraint, err)
		}
		version, ok := Parse(testCase.version)
		if !ok {
			t.Fatalf("Parse(%q) failed", testCase.version)
		}
		if actual := constraint.Match(version); actual != testCase.expected {
			t.Errorf("%q matches %q = %v, want %v", testCase.constraint, testCase.version, actual, testCase.expected)
		}
	}
}

func TestHighest(t *testing.T) {
	constraint, _ := ParseConstraint("^1")
	best, ok := Highest([]string{"v0.9.0", "v1.0.0", "v1.4.2", "v1.10.0", "v2.0.0", "main"}, constraint)
	if !ok || best != "v1.10.0" {
		t.Errorf("Highest = %q, want v1.10.0", best)
	}
}
