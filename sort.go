package rats

import (
	"sort"

	"github.com/woozymasta/semver"
)

// Sort orders tags using SemVer precedence when possible, otherwise falls back
// to lexicographic sort. When normalizeShorthand=true, X and X.Y are first
// expanded to X.0.0 and X.Y.0 respectively for comparison.
//
// Note: ties are deterministically broken by the underlying semver.Compare,
// which considers Original string if versions are otherwise equal.
func Sort(in []string, mode SortMode, normalizeShorthand bool) []string {
	if mode == SortNone || len(in) < 2 {
		return in
	}

	type item struct {
		v    semver.Semver
		orig string
	}

	arr := make([]item, 0, len(in))
	for _, t := range in {
		s := t
		if normalizeShorthand {
			s = normalizeShorthandFn(t) // indirection for test injection if needed
		}
		v, ok := semver.ParseNoCanon(s)
		if !ok || !v.IsValid() {
			// Fallback: lexicographic sort if any tag is not a valid SemVer.
			return sortLex(in, mode)
		}
		arr = append(arr, item{v: v, orig: t})
	}

	sort.Slice(arr, func(i, j int) bool {
		cmp := arr[i].v.Compare(arr[j].v)
		if mode == SortAsc {
			return cmp < 0
		}
		return cmp > 0 // SortDesc
	})

	out := make([]string, len(arr))
	for i, it := range arr {
		out[i] = it.orig
	}
	return out
}

// sortLex does a plain lexicographic sort as a fallback.
func sortLex(in []string, mode SortMode) []string {
	out := append([]string(nil), in...)
	if mode == SortAsc {
		sort.Strings(out)
	} else { // SortDesc
		sort.Sort(sort.Reverse(sort.StringSlice(out)))
	}
	return out
}

// normalizeShorthandFn allows overriding in tests; default points to normalizeShorthand.
var normalizeShorthandFn = normalizeShorthand
