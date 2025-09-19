/*
Package rats (Release App Tag Selector) provides filtering, aggregation,
and sorting utilities for container image tags.

The package is network-agnostic: it operates purely on a slice of tag strings.
Typical flow:

 1. Fetch raw tags elsewhere (e.g., via crane.ListTags).
 2. Call Select with desired Options (filters, depth, sort, range).
 3. Use the resulting list.

SemVer notes:
  - A leading "v" is accepted on input.
  - In ReleaseOnly mode, shorthand tags X and X.Y are accepted and normalized
    to X.0.0 and X.Y.0 for comparison.
  - You can choose to output the original tag or the canonical form
    (vMAJOR.MINOR.PATCH[-PRERELEASE]); build metadata is stripped in canonical.

Additional filters:
  - Include / Exclude: optional regex prefilters on raw tag strings (before SemVer parsing).
  - Range: clip by lower/upper bounds (X / X.Y / X.Y.Z or full SemVer), with inclusive/exclusive ends.

Usage example:

	raw := []string{
		"v1.2.2", "v1.2.3", "1.2.4", "1.2", "1", "1.3.0-alpha.1", "sha256-xxx.sig",
		"v2.0.0+build.1", "2.0", "v2", "someval", "001.100.01", "1.2.3.4.5", "1.1.2",
	}

	exclude, _ := regexp.Compile(`4$`)

	res := rats.Select(raw, rats.Options{
		// FilterSemver is implied by ReleaseOnly; setting true is explicit but optional here.
		FilterSemver:      true,            // enable SemVer gating
		ReleaseOnly:       true,            // only releases; allow X / X.Y / X.Y.Z
		OutputCanonical:   true,            // output in canonical format vX.Y.Z-rc
		ExcludeSignatures: true,            // drop sha256-<64 hex>.sig tags early
		Include:           nil,             // positive regexp match
		Exclude:           exclude,         // negative regexp match
		Format:            rats.FormatAll,  // permit X, X.Y, X.Y.Z
		Depth:             rats.DepthMinor, // latest per (major,minor)
		Sort:              rats.SortDesc,   // descending
		// Range:          (optional) clip by min/max bounds
	})

	fmt.Println(res) // [v2.0.0 v1.2.3 v1.1.2 v1.0.0]
*/
package rats
