package updater

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		candidate string
		current   string
		expected  bool
	}{
		{"v1.2.0", "1.1.9", true},
		{"v1.2.0", "v1.2.0", false},
		{"1.10.0", "1.9.0", true},
		{"v2.0.0-rc.1", "1.9.9", true},
		{"v1.0.0", "1.0.1", false},
		{"garbage", "1.0.0", false},
		{"v1.0.0", "dev", false},
	}
	for _, testCase := range cases {
		if actual := IsNewer(testCase.candidate, testCase.current); actual != testCase.expected {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", testCase.candidate, testCase.current, actual, testCase.expected)
		}
	}
}
