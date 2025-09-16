package rats

// normalizeShorthand converts X / X.Y into a full X.Y.Z string for comparison.
// If the input is already X.Y.Z (with optional leading "v"), it is returned as-is (minus "v").
// For any other form, this function strips a leading "v" and returns the remainder.
func normalizeShorthand(tag string) string {
	t := trimLeadingV(tag)

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

// trimLeadingV removes a single leading 'v' or 'V' if present.
func trimLeadingV(s string) string {
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		return s[1:]
	}

	return s
}
