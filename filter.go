package rats

import (
	"sort"

	"github.com/woozymasta/semver"
)

// Filter applies signature/semver/release/depth rules and returns the filtered list.
//
// Behavior summary:
//   - If both FilterSemver=false and ReleaseOnly=false => only signature and regex filtering.
//   - If ReleaseOnly=true => accepts X / X.Y / X.Y.Z (optional leading "v"),
//     rejects -prerelease and +build, and normalizes shorthands for comparison.
//   - Else if FilterSemver=true => accepts valid SemVer X.Y.Z[...], including
//     prereleases and build metadata.
//   - Depth controls aggregation (patch/minor/major/latest).
//   - Output form (Original/Canonical) is chosen later by pick().
func Filter(in []string, opt Options) []string {
	opt = opt.normalized()

	// Fast path: only signature filtering.
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

	// Parse SemVer and retain structured values while preserving Original.
	vers := make([]semver.Semver, 0, len(in))
	for _, t := range in {
		if !prefilterTag(t, opt) {
			continue
		}

		v, ok := parseCandidate(t, opt)
		if !ok {
			continue
		}

		v.Original = t
		vers = append(vers, v)
	}

	if opt.Range.Enabled() {
		vers = clipRange(vers, opt.Range)
	}

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

// prefilterTag applies cheap, non-SemVer checks to a raw tag:
// - signature filtering (e.g., "sha256-...sig") if ExcludeSignatures is set;
// - Include regex (must match if provided);
// - Exclude regex (must NOT match if provided).
// It runs before any SemVer parsing and is used on both the fast path
// (no SemVer gating) and the SemVer path.
//
// It returns true if the tag passes these prefilters and should be
// considered for further processing.
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

	if opt.StrictSemver && !strictSemver.MatchString(t) {
		return false
	}

	if opt.ExcludeSignatures && sigRe.MatchString(t) {
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

// hasLeadingV reports whether s starts with 'v' or 'V'.
func hasLeadingV(s string) bool {
	return len(s) > 0 && (s[0] == 'v' || s[0] == 'V')
}

// parseCandidate parses a tag according to the current Options.
// In ReleaseOnly mode it:
//   - requires that the tag matches one of the allowed release forms (X / X.Y / X.Y.Z);
//   - normalizes shorthands (X -> X.0.0, X.Y -> X.Y.0) for comparison;
//   - rejects any prerelease/build metadata (must be a plain release).
//
// When ReleaseOnly is false but FilterSemver is true, it accepts the full
// SemVer grammar, including prerelease and build metadata.
//
// The parser used (with or without canonicalization) is chosen via parseSemver
// based on OutputCanonical.
//
// The returned Semver has Valid=true on success. Note: the caller is responsible
// for filling v.Original with the raw tag string.
func parseCandidate(t string, opt Options) (semver.Semver, bool) {
	if opt.ReleaseOnly {
		if !matchFormat(t, opt.Format) {
			return semver.Semver{}, false
		}

		v, ok := parseSemver(normalizeShorthand(t), opt.OutputCanonical)
		if !ok || !v.IsValid() || v.Prerelease != "" || v.Build != "" {
			return semver.Semver{}, false
		}

		return v, true
	}

	// SemVer-only (allow pre/build)
	return parseSemver(t, opt.OutputCanonical)
}

// matchFormat ensures the tag is a release (no '-' or '+') and matches any allowed
// shorthand form X / X.Y / X.Y.Z. It does not parse the version; that's done later.
func matchFormat(tag string, forms Format) bool {
	// Quick reject: release-only means no prerelease/build metadata.
	for i := 0; i < len(tag); i++ {
		switch tag[i] {
		case '-', '+':
			return false
		}
	}

	if forms&FormatXYZ != 0 && relXYZ.MatchString(tag) {
		return true
	}

	if forms&FormatXY != 0 && relXY.MatchString(tag) {
		return true
	}

	if forms&FormatX != 0 && relX.MatchString(tag) {
		return true
	}

	return false
}

// latestPerMinor returns the latest per (major, minor).
func latestPerMinor(vs []semver.Semver) []semver.Semver {
	type key struct{ maj, min int }
	best := make(map[key]semver.Semver, len(vs))
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

// latestPerMajor returns the latest per major.
func latestPerMajor(vs []semver.Semver) []semver.Semver {
	best := make(map[int]semver.Semver, len(vs))
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

// pick selects which string to expose outward (Original vs Canonical).
func pick(v semver.Semver, canonical bool) string {
	if canonical && v.Canonical != "" {
		return v.Canonical
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
