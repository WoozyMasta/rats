# RATS

RATS — Release App Tag Selector.  
A small library for engineers who need to
quickly squeeze exactly what they want out of a list of container image
tags: SemVer releases, depth-based aggregation, stable sorting.

Input is just a `[]string` of tags; output is a filtered and sorted list.

## Key features

* **SemVer gating** – optionally allow only valid `X.Y.Z[-pre][+build]`.
* **ReleaseOnly mode** – excludes prerelease/build; accepts shorthands `X` /
  `X.Y` (compares as `X.0.0` / `X.Y.0`).
* **VPrefix policy** – require, forbid, or allow the leading `v` (`PrefixV`
  / `PrefixNone` / `PrefixAny`).
* **Release forms mask** – permit exactly `X`, `X.Y`, `X.Y.Z`, or any combo
  (`FormatX` | `FormatXY` | `FormatXYZ` | `FormatAll`).
* **Depth aggregation** – `Patch` (all), `Minor` (latest per major/minor),
  `Major` (latest per major), `Latest` (single best).
* **Range clipping** – min/max bounds using shorthand (`1`, `1.2`, `1.2.3`)
  or full semver; inclusive/exclusive ends; optional prerelease-at-floor
  (`>= X.Y.0-0`).
* **Regex filters** – `Include`/`Exclude` applied to raw tags before parsing
  (e.g., drop `-alpine`, `-rc`, platform suffixes).
* **Deduplicate** – merges aliases of the same version (MAJOR.MINOR.PATCH +
  PRERELEASE; build ignored). Useful with `DepthPatch` or `OutputCanonical`.
* **Sorting** – SemVer-first (`Asc`/`Desc`), with shorthand normalization in
  ReleaseOnly; falls back to lexicographic if a tag isn’t SemVer.
* **Output modes** – original tag or canonical
  `vMAJOR.MINOR.PATCH[-PRERELEASE]` (build stripped). With
  `OutputCanonical=false`, parsing avoids building canon strings to reduce
  allocations.
* **Deterministic order** – semver ties are stabilized by the original input
  string.
* **Signature filtering** – drop `sha256-<64 hex>.sig` noise from
  registries.
* **Helpers** – convenient shortcuts:
  * `DefaultOptions()` (sensible defaults: `SemVer + ReleaseOnly`,
    `FormatAll`, `DepthMinor`, `SortDesc`, `Deduplicate`),
  * `Releases(in)`,
  * `ReleasesCanonical(in)`,
  * `Latest(in)`,
  * `LatestPerMajor(in)`.

## Integration

* Fetch tags any way you like (e.g., `crane.ListTags`), then pass to
  `rats.Select`.
* Performance: `O(n)` parsing + `O(k log k)` sorting (k = size after
  filters), minimal allocations; regexes are precompiled.

## Limitations

* Shorthands are accepted only in ReleaseOnly mode.
* Build metadata is always stripped in Canonical (per [SemVer]).

## Installation

```shell
go get github.com/woozymasta/semver
```

## CLI

```shell
go install github.com/woozymasta/rats/cmd/rats@latest
```

```txt
Usage:
  rats [OPTIONS]

RATS — Release App Tag Selector.
A CLI tool for selecting versions from tag lists:
supports SemVer and Go canonical (v-prefixed), can filter prereleases, drop build metadata, sort and aggregate results.

SemVer and releases:
  -s, --semver                                  Keep only SemVer tags (X.Y.Z[-pre][+build]) (default: true)
  -r, --release-only                            Keep only releases (no -pre/+build); allow X / X.Y / X.Y.Z (default: true)
  -d, --deduplicate                             Collapse aliases of the same version (MAJOR.MINOR.PATCH+PRERELEASE) (default: true)

Output:
  -c, --canonical-out                           Print canonical vMAJOR.MINOR.PATCH[-PRERELEASE] (drop +BUILD)
  -v, --semver-out                              Print SemVer MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]

Aggregation and sort:
  -n, --limit=                                  Max number of output tags (<=0 = unlimited) (default: 0)
  -D, --depth=[patch|minor|major|latest]        Aggregation depth (default: minor)
  -S, --sort=[none|asc|desc]                    Sort output tags (default: desc)
  -f, --format=[x|xy|xyz|x-xy|x-xyz|xy-xyz|any] Allowed release forms (when --release-only) (default: any)

Input filters:
  -V, --v-prefix=[any|v|none]                   Policy for leading 'v' in tags (default: any)
  -i, --include=                                Regexp to keep tags (applied before parsing)
  -e, --exclude=                                Regexp to drop tags (applied before parsing)
  -E, --exclude-sigs                            Drop sha256-<64>.sig tags

Range:
  -m, --min=                                    Lower bound (X / X.Y / X.Y.Z or full SemVer)
  -x, --max=                                    Upper bound (X / X.Y / X.Y.Z or full SemVer)
  -M, --min-exclusive                           Exclude lower bound itself
  -X, --max-exclusive                           Exclude upper bound itself
  -p, --include-prerelease                      When min is shorthand, include prereleases at the floor (>= X.Y.0-0)

Help Options:
  -h, --help                                    Show this help message
```

The utility accepts `true|false` or `1|0` for boolean flags and allows you to disable them.

For example, you can disable all processing.

```shell
some-tool-print-tags | rats -r=0 -s=0 -d=0 -c=0 -v=0 -D=none -S=none -f=none -V=any -E=0
```

## Example

Basic example of use

```go
raw := []string{
  "3.0",
  "2.0",
  "2.0.3",
  "2.0.2",
  "1.3.0-rc1",
  "1.3.0001",
  "1.3.0",
  "1.2.4-beta.1",
  "1.2.3",
  "1.2.2",
  "1.2.1",
  "1.1.2",
  "1.0.2",
}

fmt.Println(rats.Releases(raw)) // [3.0 2.0.3 1.3.0 1.2.3 1.1.2 1.0.2]
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
