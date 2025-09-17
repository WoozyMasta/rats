package rats

// Select filters, aggregates, and sorts tags in one call.
// It is equivalent to Sort(Filter(in, opt), opt.Sort, opt.ReleaseOnly).
// When ReleaseOnly is true, Sort will normalize X / X.Y to X.0.0 / X.Y.0 for comparisons.
func Select(in []string, opt Options) []string {
	optNoLimit := opt
	optNoLimit.Limit = 0

	out := Filter(in, optNoLimit)
	if opt.Sort != SortNone {
		out = Sort(out, opt.Sort, opt.ReleaseOnly) // normalize X/X.Y when ReleaseOnly
	}

	return capStrings(out, opt.Limit)
}

// DefaultOptions returns a practical preset for stable releases:
//
//   - FilterSemver: true          // only SemVer-like tags
//   - ReleaseOnly:  true          // no prerelease/build
//   - Format:       FormatAll     // allow X, X.Y, X.Y.Z
//   - Depth:        DepthMinor    // latest per (major, minor)
//   - Sort:         SortDesc      // newest first
//   - Deduplicate:  true          // collapse equivalent forms
//
// Note: OutputCanonical is left false on purpose. Set it to true in your
// own Options if you want canonical "vMAJOR.MINOR.PATCH" output.
func DefaultOptions() Options {
	return Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatAll,
		Depth:        DepthMinor,
		Sort:         SortDesc,
		Deduplicate:  true,
	}
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

// Latest returns a single latest stable release (ReleaseOnly).
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
