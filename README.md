# RATS

RATS — Release App Tag Selector.  
A small library for engineers who need to
quickly squeeze exactly what they want out of a list of container image
tags: SemVer releases, depth-based aggregation, stable sorting.

Input is just a `[]string` of tags; output is a filtered and sorted list.

## Key features

* SemVer gating: optionally allow only valid `X.Y.Z[-pre][+build]`.
* ReleaseOnly mode: excludes prerelease and build; understands shorthands
  `X` and `X.Y` (normalizes to `X.0.0` and `X.Y.0`).
* Release forms: mask `X` / `X.Y` / `X.Y.Z` (`FormatX` | `FormatXY` |
  `FormatXYZ` | `FormatAll`).
* Depth: `Patch` (everything), `Minor` (latest per major/minor pair),
  `Major` (latest per major), `Latest` (single top-most).
* Sorting: SemVer-first (`Asc`/`Desc`); in ReleaseOnly it normalizes
  shorthands; if a non-SemVer tag appears, falls back to lexicographic.
* Signatures: can drop `sha256-<64>.sig` (handy with raw registry tag sets).
* Output: Original or Canonical (`vMAJOR.MINOR.PATCH[-PRERELEASE]`, build
  metadata removed).
* Range clipping: min/max SemVer bounds, shorthand bounds (`1`, `1.2`,
  `1.2.3`), inclusive/exclusive ends, `IncludePrerelease` (≥ `X.Y.0-0`).
* Regex filters: `Include` / `Exclude` run on raw tags before parsing (e.g.,
  cut out `-alpine`, `-rc`, platform suffixes).
* Deterministic ordering: when SemVer precedence ties, order is stabilized
  by the original input string.
* Allocation-aware: with `OutputCanonical=false`, parsing avoids
  constructing the canonical string.
* Aggregation order: for `DepthMinor`/`DepthMajor`, results are emitted in
  global SemVer order (newest → oldest), not grouped arbitrarily.

## Integration

* Fetch tags any way you like (e.g., `crane.ListTags`), then pass to
  `rats.Select`.
* Performance: `O(n)` parsing + `O(k log k)` sorting (k = size after
  filters), minimal allocations; regexes are precompiled.

## Limitations

* Shorthands are accepted only in ReleaseOnly mode.
* Build metadata is always stripped in Canonical (per [SemVer]).

## Example

Basic example of use

```go
raw := []string{
  "v1.2.2",
  "v1.2.3",
  "1.2.4",
  "1.2",
  "1",
  "1.3.0-alpha.1",
  "sha256-xxx.sig",
  "v2.0.0+build.1",
  "2.0",
  "v2",
  "someval",
  "001.100.01",
  "1.2.3.4.5",
  "1.1.2",
}

exclude, _ := regexp.Compile(`4$`)

res := rats.Select(raw, rats.Options{
  // FilterSemver is implied by ReleaseOnly; setting true is explicit but optional here.
  FilterSemver:      true,            // (optional here) enable SemVer gating
  ReleaseOnly:       true,            // only releases; allow X / X.Y / X.Y.Z
  OutputCanonical:   true,            // output in canonical format vX.Y.Z
  ExcludeSignatures: true,            // drop sha256-<64 hex>.sig tags early
  Include:           nil,             // positive regexp match
  Exclude:           exclude,         // negative regexp match
  Format:            rats.FormatAll,  // permit X, X.Y, X.Y.Z
  Depth:             rats.DepthMinor, // latest per (major,minor)
  Sort:              rats.SortDesc,   // descending
  // Range:          (optional) clip by min/max bounds
})

fmt.Println(res) // [v2.0.0 v1.2.3 v1.1.2 v1.0.0]
```

### Range clipping

Keep prerelease, range: ≥1.10.0-0 and ≤3.x

```go
raw := []string{
  "0.9.9",
  "1.9.9",
  "1.10.0-alpha.1",
  "1.10.0",
  "2.0.0-rc.1",
  "2.1.0+build.5",
  "3.0.0",
  "3.1.0-alpha",
  "3.1.0",
  "4.0.0",
}

res := rats.Select(raw, rats.Options{
  FilterSemver: true,
  Range: rats.Range{
    Min:               "1.10",
    IncludePrerelease: true, // >= 1.10.0-0
    Max:               "3",  // <= 3.x
  },
})

fmt.Println(res) // [1.10.0-alpha.1 1.10.0 2.0.0-rc.1 2.1.0+build.5 3.0.0 3.1.0-alpha 3.1.0]
```

### Regex Include + Exclude

Only `1.*` and no alpha/beta/rc

```go
raw := []string{
  "v1",
  "1.0.0",
  "1.1.0-alpha",
  "v1.2.3",
  "2.0.0",
  "1.3.0-rc.1",
  "1.3.0",
  "latest",
}

inc := regexp.MustCompile(`^v?1(\.|$)`)         // 1 and 1.*
exc := regexp.MustCompile(`-(alpha|beta|rc)\b`) // cutting out pre-releases

res := rats.Select(raw, rats.Options{
  FilterSemver: true,
  Include:      inc,
  Exclude:      exc,
})

fmt.Println(res) // [v1 1.0.0 v1.2.3 1.3.0]
```

### Latest per major

Stable releases only, canonical output

```go
raw := []string{
  "v1.2.0",
  "1.5.1",
  "1.6.0-rc.1",
  "2.0.0-alpha",
  "2",
  "2.1",
  "2.1.3",
  "3.0.0",
}

res := rats.Select(raw, rats.Options{
  ReleaseOnly:     true,            // X/X.Y shorthands are normalized
  Format:          rats.FormatAll,  // apply X, X.Y, X.Y.Z
  Depth:           rats.DepthMajor, // one for each major
  OutputCanonical: true,            // vMAJOR.MINOR.PATCH
})

fmt.Println(res) // [v3.0.0 v2.1.3 v1.5.1]
```

[SemVer]: https://semver.org/
