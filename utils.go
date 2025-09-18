package rats

import (
	"strings"
)

// toTok normalizes a free-form string into a lowercased token.
func toTok(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// splitTokens splits by common separators: comma, pipe, plus, slash, dash, space.
func splitTokens(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	out := make([]string, 0, 8)

	start := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		isAlnum := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		if isAlnum {
			if start < 0 {
				start = i
			}
			continue
		}

		if start >= 0 {
			out = append(out, s[start:i])
			start = -1
		}
	}

	if start >= 0 {
		out = append(out, s[start:])
	}

	return out
}

// joinDash joins parts with a dash. Extracted for testability.
func joinDash(parts []string) string {
	switch len(parts) {
	case 0:
		return ""

	case 1:
		return parts[0]

	default:
		var b strings.Builder
		for i, p := range parts {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteString(p)
		}

		return b.String()
	}
}

// capStrings returns out[:min(limit, len(out))] if limit>0; otherwise out.
func capStrings(out []string, limit int) []string {
	if limit > 0 && limit < len(out) {
		return out[:limit]
	}

	return out
}

// isSigTag reports whether s matches "sha256-<64 anycase hex>.sig".
func isSigTag(s string) bool {
	// "sha256-" (7) + 64 hex + ".sig" (4) = 75
	if len(s) != 75 || s[:7] != "sha256-" || s[71:] != ".sig" {
		return false
	}

	// check 64 anycase hex chars
	for i := 7; i < 71; i++ {
		c := s[i]
		if (c < '0' || c > '9') &&
			(c < 'a' || c > 'f') &&
			(c < 'A' || c > 'F') {
			return false
		}
	}

	return true
}
