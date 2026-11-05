package semver

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Original   string
}

func Parse(text string) (Version, bool) {
	version := Version{Original: text}
	trimmed := strings.TrimPrefix(strings.TrimSpace(text), "v")
	trimmed, _, _ = strings.Cut(trimmed, "+")
	trimmed, version.Prerelease, _ = strings.Cut(trimmed, "-")
	segments := strings.Split(trimmed, ".")
	if trimmed == "" || len(segments) > 3 {
		return version, false
	}
	numbers := [3]int{}
	for index, segment := range segments {
		value, err := strconv.Atoi(segment)
		if err != nil || value < 0 {
			return version, false
		}
		numbers[index] = value
	}
	version.Major, version.Minor, version.Patch = numbers[0], numbers[1], numbers[2]
	return version, true
}

func Compare(a Version, b Version) int {
	for _, difference := range []int{a.Major - b.Major, a.Minor - b.Minor, a.Patch - b.Patch} {
		if difference != 0 {
			return difference
		}
	}
	switch {
	case a.Prerelease == b.Prerelease:
		return 0
	case a.Prerelease == "":
		return 1
	case b.Prerelease == "":
		return -1
	default:
		return strings.Compare(a.Prerelease, b.Prerelease)
	}
}

func (v Version) String() string {
	text := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		text += "-" + v.Prerelease
	}
	return text
}

type operator int

const (
	operatorAny operator = iota
	operatorExact
	operatorCaret
	operatorTilde
	operatorAtLeast
)

type Constraint struct {
	operator operator
	base     Version
	depth    int
}

func ParseConstraint(text string) (Constraint, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || trimmed == "*" || trimmed == "latest" {
		return Constraint{operator: operatorAny}, nil
	}
	constraint := Constraint{operator: operatorExact}
	switch {
	case strings.HasPrefix(trimmed, "^"):
		constraint.operator, trimmed = operatorCaret, trimmed[1:]
	case strings.HasPrefix(trimmed, "~"):
		constraint.operator, trimmed = operatorTilde, trimmed[1:]
	case strings.HasPrefix(trimmed, ">="):
		constraint.operator, trimmed = operatorAtLeast, trimmed[2:]
	}
	base, ok := Parse(trimmed)
	if !ok {
		return constraint, fmt.Errorf("invalid version constraint %q", text)
	}
	constraint.base = base
	core, _, _ := strings.Cut(strings.TrimPrefix(trimmed, "v"), "-")
	constraint.depth = len(strings.Split(core, "."))
	if constraint.operator == operatorExact && constraint.depth < 3 {
		constraint.operator = operatorTilde
	}
	return constraint, nil
}

func (c Constraint) Match(version Version) bool {
	if version.Prerelease != "" && !(c.operator == operatorExact && c.base.Prerelease != "") {
		return false
	}
	switch c.operator {
	case operatorAny:
		return true
	case operatorExact:
		return Compare(version, c.base) == 0
	case operatorAtLeast:
		return Compare(version, c.base) >= 0
	case operatorCaret:
		if Compare(version, c.base) < 0 {
			return false
		}
		if c.base.Major > 0 || c.depth == 1 {
			return version.Major == c.base.Major
		}
		return version.Major == 0 && version.Minor == c.base.Minor
	case operatorTilde:
		if Compare(version, c.base) < 0 || version.Major != c.base.Major {
			return false
		}
		return c.depth == 1 || version.Minor == c.base.Minor
	}
	return false
}

func Highest(candidates []string, constraint Constraint) (string, bool) {
	var best *Version
	for _, candidate := range candidates {
		version, ok := Parse(candidate)
		if !ok || !constraint.Match(version) {
			continue
		}
		if best == nil || Compare(version, *best) > 0 {
			best = &version
		}
	}
	if best == nil {
		return "", false
	}
	return best.Original, true
}

func SortDescending(candidates []string) []string {
	var valid []Version
	for _, candidate := range candidates {
		if version, ok := Parse(candidate); ok {
			valid = append(valid, version)
		}
	}
	slices.SortFunc(valid, func(a, b Version) int { return Compare(b, a) })
	sorted := make([]string, len(valid))
	for index, version := range valid {
		sorted[index] = version.Original
	}
	return sorted
}
