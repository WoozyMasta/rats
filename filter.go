package rats

import (
	"sort"

	"github.com/woozymasta/semver"
)

// Filter applies signature/semver/release/depth rules and returns the filtered list.
func Filter(in []string, opt Options) []string {
	opt = opt.normalized()

	// Fast path: only signature/regex filtering.
	if !opt.FilterSemver && !opt.ReleaseOnly {
		out := make([]string, 0, len(in))
		for _, t := range in {
			if !prefilterTag(t, opt) {
				continue
			}
			out = append(out, t)
		}

		return capStrings(out, opt.Limit)
	}

	// Parse SemVer once and keep structured values.
	vers := make([]semver.Semver, 0, len(in))
	for _, t := range in {
		if !prefilterTag(t, opt) {
			continue
		}

		v, ok := parseCandidate(t, opt)
		if !ok {
			continue
		}
		// keep original for output selection
		v.Original = t
		vers = append(vers, v)
	}

	// Range clipping (on parsed versions).
	if opt.Range.Enabled() {
		vers = clipRange(vers, opt.Range)
	}

	// Deduplicate aliases (e.g., "1.2" vs "v1.2.0") if requested or when canonicalizing.
	if opt.Deduplicate || opt.OutputCanonical {
		vers = deduplicateSemver(vers)
	}

	// Depth aggregation.
	switch opt.Depth {
	case DepthMinor:
		return capStrings(projectOut(latestPerMinor(vers), opt.OutputCanonical), opt.Limit)

	case DepthMajor:
		return capStrings(projectOut(latestPerMajor(vers), opt.OutputCanonical), opt.Limit)

	case DepthLatest:
		if len(vers) == 0 {
			return nil
		}

		best := vers[0]
		for _, x := range vers[1:] {
			if x.Compare(best) > 0 {
				best = x
			}
		}

		return capStrings([]string{pick(best, opt.OutputCanonical)}, opt.Limit)

	default: // DepthPatch
		out := make([]string, 0, len(vers))
		for _, v := range vers {
			out = append(out, pick(v, opt.OutputCanonical))
		}

		return capStrings(out, opt.Limit)
	}
}

// prefilterTag: cheap checks before parsing (user regexes, signatures, v-prefix policy).
func prefilterTag(t string, opt Options) bool {
	switch opt.VPrefix {
	case PrefixV:
		if !hasLeadingV(t) {
			return false
		}
	case PrefixNone:
		if hasLeadingV(t) {
			return false
		}
	}

	if opt.ExcludeSignatures && isSigTag(t) {
		return false
	}

	if opt.Include != nil && !opt.Include.MatchString(t) {
		return false
	}

	if opt.Exclude != nil && opt.Exclude.MatchString(t) {
		return false
	}

	return true
}

func hasLeadingV(s string) bool {
	return len(s) > 0 && (s[0] == 'v' || s[0] == 'V')
}

// parseCandidate parses once and applies ReleaseOnly + format mask using semver Flags.
func parseCandidate(t string, opt Options) (semver.Semver, bool) {
	v, ok := semver.Parse(t)
	if !ok || !v.IsValid() {
		return semver.Semver{}, false
	}

	if !opt.FilterSemver && !opt.ReleaseOnly {
		// (the fast path already returned earlier, so we don't hit this)
		return v, true
	}

	if opt.ReleaseOnly {
		if !v.IsRelease() { // no pre/build
			return semver.Semver{}, false
		}
		if !formatAllowed(v, opt.Format) {
			return semver.Semver{}, false
		}
	}

	// else: FilterSemver==true (accept full semver, pre/build allowed)
	return v, true
}

// formatAllowed maps the release form mask to flags from the parsed version.
// X    => !HasMinor && !HasPatch
// X.Y  => HasMinor && !HasPatch
// X.Y.Z=> HasPatch
func formatAllowed(v semver.Semver, mask Format) bool {
	if v.HasPatch() {
		return (mask & FormatXYZ) != 0
	}

	if v.HasMinor() {
		return (mask & FormatXY) != 0
	}

	return (mask & FormatX) != 0
}

// latestPerMinor / latestPerMajor unchanged
func latestPerMinor(vs []semver.Semver) []semver.Semver { /* same as before */
	type key struct{ maj, min int }
	best := make(map[key]semver.Semver)
	for _, v := range vs {
		k := key{v.Major, v.Minor}
		if cur, ok := best[k]; !ok || v.Compare(cur) > 0 {
			best[k] = v
		}
	}

	acc := make([]semver.Semver, 0, len(best))
	for _, v := range best {
		acc = append(acc, v)
	}

	sort.Slice(acc, func(i, j int) bool { return acc[i].Compare(acc[j]) > 0 })

	return acc
}

func latestPerMajor(vs []semver.Semver) []semver.Semver { /* same as before */
	best := make(map[int]semver.Semver)
	for _, v := range vs {
		if cur, ok := best[v.Major]; !ok || v.Compare(cur) > 0 {
			best[v.Major] = v
		}
	}

	acc := make([]semver.Semver, 0, len(best))
	for _, v := range best {
		acc = append(acc, v)
	}

	sort.Slice(acc, func(i, j int) bool { return acc[i].Compare(acc[j]) > 0 })

	return acc
}

// Output selection
func pick(v semver.Semver, canonical bool) string {
	if canonical {
		return v.Canonical()
	}

	return v.Original
}

// projectOut maps a slice of Semver to strings according to OutputCanonical.
func projectOut(in []semver.Semver, canonical bool) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, pick(v, canonical))
	}

	return out
}

// deduplicateSemver use as final deduplication after all filters
func deduplicateSemver(vs []semver.Semver) []semver.Semver {
	type key struct {
		maj, min, pat int
		pre           string
	}
	seen := make(map[key]struct{}, len(vs))
	keep := vs[:0]

	for _, v := range vs {
		k := key{v.Major, v.Minor, v.Patch, v.Prerelease}
		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		keep = append(keep, v)
	}

	return keep
}
