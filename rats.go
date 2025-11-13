package rats

// DefaultOptions returns a practical preset for stable releases:
//
//   - FilterSemver: true          // only SemVer-like tags
//   - Format:       FormatAll     // allow X, X.Y, X.Y.Z
//   - Depth:        DepthMinor    // latest per (major, minor)
//   - Sort:         SortDesc      // newest first
//   - Deduplicate:  true          // collapse equivalent format
//
// Note: OutputCanonical is left false on purpose. Set it to true in your
// own Options if you want canonical "vMAJOR.MINOR.PATCH" output.
func DefaultOptions() Options {
	return Options{
		FilterSemver: true,
		Format:       FormatAll,
		Depth:        DepthMinor,
		Sort:         SortDesc,
		Deduplicate:  true,
	}
}

// Select filters, aggregates, and sorts tags.
// Simple, readable pipeline:
//  1. cheap raw prefilter (VPrefix/regex/signatures)
//  2. parse all (once)
//  3. if no semver at all -> string-only path (lex sort, limit)
//  4. else -> semver path (Format -> Range -> Dedup -> Depth -> Sort)
//     non-semver are kept only when not gating by semver, and appended after semver.
func Select(in []string, opt Options) []string {
	opt = opt.normalized()

	// 1) raw prefilter
	raw := preFilterRaw(in, opt)
	if len(raw) == 0 {
		return nil
	}

	// 2) parse once
	rs, semCount := parseAll(raw)

	// 3) if there are no semver at all -> string-only pipeline
	if semCount == 0 {
		if opt.FilterSemver {
			return nil
		}

		out := stringOnlyPipeline(raw, opt)
		return capStrings(out, opt.Limit)
	}

	// 4) semver pipeline
	sem, other := splitSemver(rs)

	// SemVer gating: ReleaseOnly / FilterSemver
	if opt.Format != FormatNone {
		sem = filterReleaseOnly(sem, opt.Format)
		// non-semver are dropped in ReleaseOnly mode
		other = nil
	} else if opt.FilterSemver {
		// keep only valid semver
		other = nil
	}

	// Range (only for semver)
	if opt.Range.Enabled() && len(sem) > 0 {
		sem = applyRange(sem, opt.Range)
	}

	// Deduplicate by (X.Y.Z + prerelease), ignoring build
	if opt.Deduplicate && len(sem) > 0 {
		sem = deduplicate(sem)
	}

	// Depth aggregation (for semver only)
	if len(sem) > 0 {
		switch opt.Depth {
		case DepthPatch:

		case DepthMinor:
			sem = aggregateMinor(sem)
		case DepthMajor:
			sem = aggregateMajor(sem)
		case DepthLatest:
			sem = aggregateLatest(sem)
		default: // DepthPatch -> keep all
		}
	}

	// Sort
	switch opt.Sort {
	case SortAsc:
		sortSemver(sem, true)
		sortStrings(other, true)
	case SortDesc:
		sortSemver(sem, false)
		sortStrings(other, false)
	default:
		// keep original order (stable by idx)
	}

	// Join semver first, then non-semver (when kept)
	render := make([]string, 0, len(sem)+len(other))
	if opt.OutputCanonical {
		for _, r := range sem {
			render = append(render, r.ver.Canonical())
		}
	} else if opt.OutputSemVer {
		for _, r := range sem {
			render = append(render, r.ver.SemVer())
		}
	} else {
		for _, r := range sem {
			render = append(render, r.raw)
		}
	}
	render = append(render, other...)

	// Limit
	return capStrings(render, opt.Limit)
}

// Releases runs Select with DefaultOptions.
//
// It keeps only stable SemVer releases (accepts X / X.Y / X.Y.Z),
// aggregates to the latest per (major,minor), sorts in descending
// SemVer order, and deduplicates equivalent tags (e.g. "1.2" vs "v1.2.0").
// Equivalent to Select(in, DefaultOptions()).
func Releases(in []string) []string {
	return Select(in, DefaultOptions())
}

// Latest returns a single latest stable release.
// DepthLatest + SortDesc + Deduplicate=true.
func Latest(in []string) []string {
	opt := DefaultOptions()
	opt.Depth = DepthLatest

	return Select(in, opt)
}

// LatestPerMajor returns the latest stable release for each major series.
// DepthMajor + SortDesc + Deduplicate=true.
func LatestPerMajor(in []string) []string {
	opt := DefaultOptions()
	opt.Depth = DepthMajor

	return Select(in, opt)
}

// ReleasesCanonical is like Releases but returns canonical strings
// ("vMAJOR.MINOR.PATCH") in the output.
func ReleasesCanonical(in []string) []string {
	opt := DefaultOptions()
	opt.OutputCanonical = true

	return Select(in, opt)
}
