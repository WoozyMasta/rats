package rats

import "regexp"

// Options configures filtering and sorting behavior.
type Options struct {
	// FilterSemver enables SemVer gating (X.Y.Z[...]).
	// If false and ReleaseOnly is false, only signature filtering is applied.
	FilterSemver bool

	// ReleaseOnly keeps only release versions (no -prerelease / +build).
	// In this mode, shorthand tags X / X.Y are accepted and normalized
	// to X.0.0 / X.Y.0 for comparison.
	ReleaseOnly bool

	// Deduplicate merges aliases of the same semantic version
	// (MAJOR.MINOR.PATCH + PRERELEASE; build is ignored) after parsing
	// and before Depth* aggregation. Preserves the order of first appearance.
	Deduplicate bool

	// OutputCanonical when true returns canonical version string (vMAJOR.MINOR.PATCH[-PRERELEASE]),
	// build metadata stripped), otherwise returns the original input tag.
	OutputCanonical bool

	// StrictSemver additional (redundant) check with full SemVer regular expression
	// (including optional "v" parameter).
	StrictSemver bool

	// ExcludeSignatures drops signature-like tags: sha256-<64 hex>.sig
	ExcludeSignatures bool

	// Include positive regex filters applied to the raw tag and keep only tags that match.
	Include *regexp.Regexp

	// Exclude negative regex filters applied to the raw tag and drop tags that match.
	Exclude *regexp.Regexp

	// Depth controls aggregation (patch/minor/major/latest).
	Depth Depth

	// Forms restricts allowed release forms in ReleaseOnly mode (X/XY/XYZ).
	// Ignored if ReleaseOnly=false. Default is FormatXYZ.
	Format Format

	// Sort defines final output ordering (none/asc/desc).
	Sort SortMode

	// VPrefix controls whether tags must, may, or must not start with a leading 'v'.
	// This only affects input acceptance. If OutputCanonical=true, the canonical
	// string will use the "vMAJOR.MINOR.PATCH[...]" form per SemVer rules.
	VPrefix VPrefix

	// Range clipping. Applied after parsing and before aggregation.
	Range Range
}

// normalized returns a copy with implicit defaults applied.
func (o Options) normalized() Options {
	out := o

	// ReleaseOnly implies SemVer gating.
	if out.ReleaseOnly && !out.FilterSemver {
		out.FilterSemver = true
	}

	// In ReleaseOnly, zero Format defaults to FormatXYZ.
	if out.ReleaseOnly && out.Format == 0 {
		out.Format = FormatXYZ
	}

	return out
}

// Depth controls aggregation granularity for SemVer-filtered tags.
type Depth int

const (
	// DepthPatch keeps all distinct X.Y.Z* entries (no aggregation).
	DepthPatch Depth = iota
	// DepthMinor keeps the latest per (major, minor).
	DepthMinor
	// DepthMajor keeps the latest per major.
	DepthMajor
	// DepthLatest keeps a single latest tag overall.
	DepthLatest
)

// String returns a stable textual representation for Depth.
func (d Depth) String() string {
	switch d {
	case DepthLatest:
		return "latest"
	case DepthMajor:
		return "major"
	case DepthMinor:
		return "minor"
	default:
		return "patch"
	}
}

// ParseDepth maps free-form tokens to Depth.
// Supported aliases (case-insensitive):
//
//	latest:  "latest","l","head","max","0"
//	major:   "major","maj","x","1"
//	minor:   "minor","min","xy","2"
//	patch:   "patch","pth","xyz","3"
func ParseDepth(s string) Depth {
	switch toTok(s) {
	// single latest
	case "latest", "l", "head", "max", "0":
		return DepthLatest

	// aggregate per major X
	case "major", "maj", "x", "1":
		return DepthMajor

	// aggregate per minor X.Y
	case "minor", "min", "xy", "2":
		return DepthMinor

	// keep all X.Y.Z
	case "patch", "pth", "xyz", "3":
		return DepthPatch

	default:
		return DepthPatch
	}
}

// Format is a bitmask of allowed release forms: X / X.Y / X.Y.Z.
type Format uint8

const (
	// FormatXYZ allows X.Y.Z.
	FormatXYZ Format = 1 << iota
	// FormatXY allows X.Y.
	FormatXY
	// FormatX allows X.
	FormatX
	// FormAll enables all forms (X, X.Y, X.Y.Z).
	FormatAll = FormatXYZ | FormatXY | FormatX
)

// String returns a canonical textual representation like "x-xy-xyz".
func (f Format) String() string {
	if f == FormatAll {
		return "x-xy-xyz"
	}

	out := make([]string, 0, 3)
	if f&FormatX != 0 {
		out = append(out, "x")
	}

	if f&FormatXY != 0 {
		out = append(out, "xy")
	}

	if f&FormatXYZ != 0 {
		out = append(out, "xyz")
	}

	if len(out) == 0 {
		return "xyz"
	}

	return joinDash(out)
}

// ParseFormat accepts combos:
//
//	single: "x", "xy", "xyz", "1|2|3", "major|minor|patch"
//	combos: "x-xy", "x,xyz", "xy|xyz", "x+xy+xyz"
//	any:    "any", "all", "*", "x-xy-xyz"
func ParseFormat(s string) Format {
	s = toTok(s)
	if s == "" {
		return FormatXYZ
	}
	// quick path for any/all
	switch s {
	case "any", "all", "*", "a":
		return FormatAll
	}

	toks := splitTokens(s)
	if len(toks) == 0 {
		return FormatXYZ
	}

	var mask Format
	for _, t := range toks {
		switch t {
		case "x", "1", "major", "maj":
			mask |= FormatX
		case "xy", "2", "minor", "min":
			mask |= FormatXY
		case "xyz", "3", "patch", "pth":
			mask |= FormatXYZ
		}
	}

	if mask == 0 {
		return FormatXYZ
	}

	return mask
}

// SortMode controls the final output ordering.
type SortMode uint8

const (
	// SortNone preserves the existing order.
	SortNone SortMode = iota
	// SortAsc sorts ascending by SemVer (fallback to lexicographic).
	SortAsc
	// SortDesc sorts descending by SemVer (fallback to lexicographic).
	SortDesc
)

// String returns a stable textual representation for SortMode.
func (m SortMode) String() string {
	switch m {
	case SortAsc:
		return "ascending"
	case SortDesc:
		return "descending"
	default:
		return "none"
	}
}

// ParseSort maps strings to SortMode.
// Supported aliases:
//
//	asc:  "asc","ascending","inc","increase","up"
//	desc: "desc","descending","dec","decrease","down"
//	none: "none","default","asis"
func ParseSort(s string) SortMode {
	switch toTok(s) {
	// ascending (low -> high)
	case "asc", "ascending", "inc", "increase", "up":
		return SortAsc

	// descending (high -> low)
	case "desc", "descending", "dec", "decrease", "down":
		return SortDesc

	// as is
	case "none", "default", "asis":
		return SortNone

	default:
		return SortNone
	}
}

// VPrefix controls acceptance of a leading 'v' on input tags.
// It is applied during the cheap pre-filter step before any SemVer parsing.
type VPrefix uint8

const (
	// PrefixAny accepts both forms, with or without a leading 'v'
	// (e.g., "1.2.3" and "v1.2.3").
	PrefixAny VPrefix = iota

	// PrefixV requires a leading 'v' (e.g., "v1.2.3"); tags without 'v'
	// are rejected before SemVer parsing.
	PrefixV

	// PrefixNone forbids a leading 'v' (e.g., "1.2.3"); tags starting with 'v'
	// are rejected before SemVer parsing.
	PrefixNone // запрещать ведущий 'v'
)

// String returns a stable textual representation for VPrefix.
func (m VPrefix) String() string {
	switch m {
	case PrefixV:
		return "v"
	case PrefixNone:
		return "none"
	default:
		return "any"
	}
}

// ParseVPrefix maps free-form strings to VPrefix.
// Supported aliases (case-insensitive):
//
//	any:  "", "any", "*", "auto":
//	v:    "v", "with-v", "require-v", "required":
//	none: "none", "no-v", "without-v", "forbidden":
func ParseVPrefix(s string) VPrefix {
	switch toTok(s) {
	case "", "any", "*", "auto":
		return PrefixAny
	case "v", "with-v", "require-v", "required":
		return PrefixV
	case "none", "no-v", "without-v", "forbidden":
		return PrefixNone
	default:
		return PrefixAny
	}
}

// Range clips versions to [Min, Max] with optional exclusive ends.
// Min/Max accept X, X.Y, X.Y.Z (with optional 'v') or full SemVer (may include -prerelease).
type Range struct {
	Min string // empty => no lower bound
	Max string // empty => no upper bound

	// When true => exclusive bound. Default false => inclusive.
	MinExclusive bool
	MaxExclusive bool

	// When Min is shorthand (X or X.Y), include pre-releases at the floor by using "-0".
	// E.g. Min="1.2" + IncludePrerelease=true => lower floor is "1.2.0-0".
	IncludePrerelease bool
}

func (r Range) Enabled() bool {
	return r.Min != "" || r.Max != ""
}
