package rats

import "strings"

// normalizeShorthand converts X / X.Y into a full X.Y.Z string for comparison.
// If the input is already X.Y.Z (with optional leading "v"), it is returned as-is (minus "v").
// For any other form, this function strips a leading "v" and returns the remainder.
func normalizeShorthand(tag string) string {
	t := strings.TrimPrefix(tag, "v")

	switch {
	case relXYZ.MatchString(tag): // already full X.Y.Z
		return t

	case relXY.MatchString(tag): // expand X.Y -> X.Y.0
		return t + ".0"

	case relX.MatchString(tag): // expand X -> X.0.0
		return t + ".0.0"

	default: // fallback: strip leading v only
		return t
	}
}
