package rats

import (
	"sort"

	"github.com/woozymasta/semver"
)

// rec is an internal record carrying raw tag, input index, and parsed semver (if valid).
type rec struct {
	raw string        // raw input string
	ver semver.Semver // semver
	idx int           // position
}

// * raw prefilter (cheap, string-only)

// preFilterRaw applies VPrefix / Include / Exclude / signature drop (when requested).
func preFilterRaw(in []string, opt Options) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		// V prefix gate
		if !acceptVPrefix(s, opt.VPrefix) {
			continue
		}

		// regex gates
		if opt.Include != nil && !opt.Include.MatchString(s) {
			continue
		}

		if opt.Exclude != nil && opt.Exclude.MatchString(s) {
			continue
		}

		// signatures drop (useful only when not strictly gating by semver, but cheap anyway)
		if opt.ExcludeSignatures && isSigTag(s) {
			continue
		}

		out = append(out, s)
	}

	return out
}

// * parsing & classification

// parseAll parses every tag. Returns all records and number of valid semver.
func parseAll(in []string) ([]rec, int) {
	rs := make([]rec, 0, len(in))
	semCount := 0

	for idx, s := range in {
		r := rec{raw: s, idx: idx}
		if v, ok := semver.Parse(s); ok && v.Valid {
			r.ver = v
			semCount++
		}

		rs = append(rs, r)
	}

	return rs, semCount
}

// splitSemver separates valid semver recs and non-semver raw strings.
func splitSemver(rs []rec) (sem []rec, other []string) {
	for _, r := range rs {
		if r.ver.Valid {
			sem = append(sem, r)
		} else {
			other = append(other, r.raw)
		}
	}

	return
}

// * string-only pipeline

func stringOnlyPipeline(in []string, opt Options) []string {
	// Sorting: lexicographic only
	switch opt.Sort {
	case SortAsc:
		sortStrings(in, true)
	case SortDesc:
		sortStrings(in, false)
	default:
		// as-is
	}

	// Nothing else applies (Range/Dedup/Depth are semver-only)
	return in
}

func sortStrings(in []string, asc bool) {
	if len(in) < 2 {
		return
	}

	sort.SliceStable(in, func(i, j int) bool {
		if asc {
			return in[i] < in[j]
		}

		return in[i] > in[j]
	})
}

// * semver gating

// filterReleaseOnly keeps only releases (no prerelease/build) and checks X/XY/XYZ form mask.
func filterReleaseOnly(in []rec, fm Format) []rec {
	out := in[:0]
	for _, r := range in {
		v := r.ver
		if has(v.Flags, semver.FlagHasPre) || has(v.Flags, semver.FlagHasBuild) {
			continue
		}

		if fm != 0 {
			if (formFromFlags(v.Flags) & fm) == 0 {
				continue
			}
		}

		out = append(out, r)
	}

	return out
}

func has(f semver.Flags, bit semver.Flags) bool {
	return (f & bit) != 0
}

func formFromFlags(f semver.Flags) Format {
	switch {
	case has(f, semver.FlagHasMajor) && has(f, semver.FlagHasMinor) && has(f, semver.FlagHasPatch):
		return FormatXYZ
	case has(f, semver.FlagHasMajor) && has(f, semver.FlagHasMinor) && !has(f, semver.FlagHasPatch):
		return FormatXY
	case has(f, semver.FlagHasMajor) && !has(f, semver.FlagHasMinor) && !has(f, semver.FlagHasPatch):
		return FormatX
	default:
		return FormatAll
	}
}

// * range

func applyRange(in []rec, r Range) []rec {
	if len(in) == 0 {
		return in
	}
	minV, hasMin := parseBound(r.Min, r.IncludePrerelease, false)
	maxV, hasMax := parseBound(r.Max, r.IncludePrerelease, true)

	out := in[:0]
	for _, it := range in {
		v := it.ver
		if hasMin {
			c := v.Compare(minV)
			if c < 0 || (c == 0 && r.MinExclusive) {
				continue
			}
		}

		if hasMax {
			c := v.Compare(maxV)
			if c > 0 || (c == 0 && r.MaxExclusive) {
				continue
			}
		}

		out = append(out, it)
	}

	return out
}

func parseBound(s string, includePre bool, isMax bool) (semver.Semver, bool) {
	if s == "" {
		return semver.Semver{}, false
	}

	v, ok := semver.Parse(s)
	if !ok || !v.Valid {
		return semver.Semver{}, false
	}

	// For Min with shorthand and IncludePrerelease => floor to "-0"
	if includePre && !isMax {
		f := v.Flags
		if !has(f, semver.FlagHasMinor) || !has(f, semver.FlagHasPatch) {
			if vv, ok2 := v.WithPre("0"); ok2 {
				v = vv
			}
		}
	}

	return v, true
}

// * dedup

type dkey struct {
	pre           string
	maj, min, pat int
}

func deduplicate(in []rec) []rec {
	seen := make(map[dkey]struct{}, len(in))
	out := in[:0]

	for _, r := range in {
		v := r.ver
		k := dkey{maj: v.Major, min: v.Minor, pat: v.Patch, pre: v.Prerelease}
		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, r)
	}

	return out
}

// * aggregation (Depth)

func aggregateMinor(in []rec) []rec {
	type best struct{ r rec }
	by := make(map[uint64]best, len(in))
	order := make([]uint64, 0, 64)

	pack := func(maj, minV int) uint64 {
		if maj < 0 || minV < 0 {
			return 0 // semver never gives negative, just a guard
		}
		// #nosec G115 -- semver major/minor are bounded, safe to cast
		return (uint64(maj) << 32) | uint64(minV&0xffffffff)
	}

	for _, r := range in {
		v := r.ver
		k := pack(v.Major, v.Minor)

		if b, ok := by[k]; ok {
			c := v.Compare(b.r.ver)
			if c > 0 || (c == 0 && r.idx < b.r.idx) {
				by[k] = best{r: r}
			}
		} else {
			by[k] = best{r: r}
			order = append(order, k)
		}
	}

	out := make([]rec, 0, len(by))
	for _, k := range order {
		out = append(out, by[k].r)
	}

	return out
}

func aggregateMajor(in []rec) []rec {
	type best struct{ r rec }
	by := make(map[int]best, len(in))
	order := make([]int, 0, 64)

	for _, r := range in {
		v := r.ver
		k := v.Major
		if b, ok := by[k]; ok {
			c := v.Compare(b.r.ver)
			if c > 0 || (c == 0 && r.idx < b.r.idx) {
				by[k] = best{r: r}
			}
		} else {
			by[k] = best{r: r}
			order = append(order, k)
		}
	}

	out := make([]rec, 0, len(by))
	for _, k := range order {
		out = append(out, by[k].r)
	}

	return out
}

func aggregateLatest(in []rec) []rec {
	if len(in) == 0 {
		return in
	}

	best := in[0]
	for i := 1; i < len(in); i++ {
		v := in[i].ver
		c := v.Compare(best.ver)
		if c > 0 || (c == 0 && in[i].idx < best.idx) {
			best = in[i]
		}
	}

	return []rec{best}
}

// * Sorting

func sortSemver(in []rec, asc bool) {
	if len(in) < 2 {
		return
	}

	sort.SliceStable(in, func(i, j int) bool {
		a, b := in[i], in[j]
		c := a.ver.Compare(b.ver)
		if c == 0 {
			// deterministic tie-breaker: lex raw, then by input order
			if a.raw != b.raw {
				if asc {
					return a.raw < b.raw
				}
				return a.raw > b.raw
			}
			return a.idx < b.idx
		}

		if asc {
			return c < 0
		}

		return c > 0
	})
}

// * V prefix

// acceptVPrefix checks input acceptance rules for leading 'v'/'V'.
func acceptVPrefix(s string, mode VPrefix) bool {
	if len(s) == 0 {
		return mode != PrefixV
	}

	hasV := s[0] == 'v' || s[0] == 'V'
	switch mode {
	case PrefixV:
		return hasV
	case PrefixNone:
		return !hasV
	default:
		return true
	}
}
