package rats

import (
	"strings"
	"unicode"
)

// toTok normalizes a free-form string into a lowercased token.
func toTok(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// splitTokens splits by common separators: comma, pipe, plus, slash, dash, space.
func splitTokens(s string) []string {
	f := func(r rune) bool {
		// keep only letters and digits; anything else is a separator
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	}
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), f)

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
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
