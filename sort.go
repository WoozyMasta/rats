package rats

import (
	"sort"

	"github.com/woozymasta/semver"
)

// Sort orders tags using SemVer precedence when possible, otherwise falls back
// to lexicographic sort.
//
// Note: ties are deterministically broken by the underlying semver.Compare,
// which considers Original string if versions are otherwise equal.
func Sort(in []string, mode SortMode) []string {
	if mode == SortNone || len(in) < 2 {
		return in
	}

	type item struct {
		v    semver.Semver
		orig string
	}

	arr := make([]item, 0, len(in))
	for _, t := range in {
		v, ok := semver.Parse(t)
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

// SortN sorts and then returns at most N items.
func SortN(in []string, mode SortMode, n int) []string {
	return capStrings(Sort(in, mode), n)
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
