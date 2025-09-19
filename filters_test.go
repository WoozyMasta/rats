// filters_test.go
package rats

import (
	"regexp"
	"sort"
	"testing"
)

// * helpers

func parseRecs(t *testing.T, tags []string) []rec {
	t.Helper()
	rs, _ := parseAll(tags)
	return rs
}

func sigTag() string {
	// 64 hex 'a'
	return "sha256-" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + ".sig"
}

func eqStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d, want %d\n got=%v\nwant=%v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("mismatch at %d: got %q, want %q\n got=%v\nwant=%v", i, got[i], want[i], got, want)
		}
	}
}

// * acceptVPrefix

func TestAcceptVPrefix(t *testing.T) {
	cases := []struct {
		s    string
		mode VPrefix
		want bool
	}{
		{"v1.2.3", PrefixAny, true},
		{"1.2.3", PrefixAny, true},
		{"v1.0.0", PrefixV, true},
		{"1.0.0", PrefixV, false},
		{"v2", PrefixNone, false},
		{"2", PrefixNone, true},
	}
	for _, c := range cases {
		if got := acceptVPrefix(c.s, c.mode); got != c.want {
			t.Fatalf("acceptVPrefix(%q,%v)=%v, want %v", c.s, c.mode, got, c.want)
		}
	}
}

// * preFilterRaw

func TestPreFilterRaw(t *testing.T) {
	in := []string{
		"v1.2.3", "1.2.3", "foo", "bar-win", "bar-linux",
		sigTag(), "sha256-bad.sig", // one valid, one invalid
	}
	opt := Options{
		VPrefix:           PrefixAny,
		Include:           regexp.MustCompile(`^[A-Za-z0-9.+_-]+$`),
		Exclude:           regexp.MustCompile(`-win$`),
		ExcludeSignatures: true,
	}
	got := preFilterRaw(in, opt)
	// drops: valid signature, "-win", everything else stays
	want := []string{"v1.2.3", "1.2.3", "foo", "bar-linux", "sha256-bad.sig"}
	eqStrings(t, got, want)
}

// * parseAll / splitSemver

func TestParseAllAndSplit(t *testing.T) {
	in := []string{"1.2.3", "foo", "v2", "1.0.0-alpha+build", "1.2.3.4"}
	rs, semCount := parseAll(in)
	if semCount != 3 {
		t.Fatalf("semCount=%d, want 3", semCount)
	}
	sem, other := splitSemver(rs)
	if len(sem) != 3 || len(other) != 2 {
		t.Fatalf("split: sem=%d other=%d, want 3/2", len(sem), len(other))
	}
	// preserve raw for others
	sort.Strings(other)
	eqStrings(t, other, []string{"1.2.3.4", "foo"})
}

// * stringOnlyPipeline

func TestStringOnlyPipeline_Sort(t *testing.T) {
	in := []string{"b", "a", "c"}
	got := stringOnlyPipeline(append([]string{}, in...), Options{Sort: SortAsc})
	eqStrings(t, got, []string{"a", "b", "c"})

	got = stringOnlyPipeline(append([]string{}, in...), Options{Sort: SortDesc})
	eqStrings(t, got, []string{"c", "b", "a"})

	got = stringOnlyPipeline(append([]string{}, in...), Options{Sort: SortNone})
	eqStrings(t, got, []string{"b", "a", "c"})
}

// * filterReleaseOnly + format

func TestFilterReleaseOnly_FormatMask(t *testing.T) {
	tags := []string{"1", "1.2", "1.2.3", "1.2.3-rc.1", "1.2.3+meta", "v2"}
	rs := parseRecs(t, tags)

	// Allow X, XY, XYZ
	keepAll := filterReleaseOnly(append([]rec{}, rs...), FormatAll)
	got := make([]string, 0, len(keepAll))
	for _, r := range keepAll {
		got = append(got, r.raw)
	}
	eqStrings(t, got, []string{"1", "1.2", "1.2.3", "v2"})

	// Only XYZ
	onlyXYZ := filterReleaseOnly(append([]rec{}, rs...), FormatXYZ)
	got = got[:0]
	for _, r := range onlyXYZ {
		got = append(got, r.raw)
	}
	eqStrings(t, got, []string{"1.2.3"})
}

// * applyRange (incl. IncludePrerelease at Min)

func TestApplyRange_MinMax_WithPrereleaseFloor(t *testing.T) {
	tags := []string{"1.2.0-rc.1", "1.2.0", "1.2.5", "1.3.0", "2.0.0"}
	sem := parseRecs(t, tags)

	// Min="1.2" with IncludePrerelease=true should include "1.2.0-rc.1"
	rr := Range{Min: "1.2", IncludePrerelease: true}
	got := applyRange(append([]rec{}, sem...), rr)
	out := make([]string, 0, len(got))
	for _, r := range got {
		out = append(out, r.raw)
	}
	eqStrings(t, out, []string{"1.2.0-rc.1", "1.2.0", "1.2.5", "1.3.0", "2.0.0"})

	// Clip [1.2, 1.3.0) — exclusive max drops 1.3.0
	rr = Range{Min: "1.2", Max: "1.3.0", MaxExclusive: true}
	got = applyRange(append([]rec{}, sem...), rr)
	out = out[:0]
	for _, r := range got {
		out = append(out, r.raw)
	}
	eqStrings(t, out, []string{"1.2.0", "1.2.5"})
}

// * deduplicate

func TestDeduplicate_CorePlusPrerelease(t *testing.T) {
	// 1.2.3 release seen multiple times (with v and build), and prerelease twice (with build)
	tags := []string{"1.2.3", "v1.2.3", "1.2.3+build5", "1.2.3-rc.1", "1.2.3-rc.1+xyz"}
	sem := parseRecs(t, tags)

	got := deduplicate(append([]rec{}, sem...))
	// Expect first release "1.2.3" and first prerelease "1.2.3-rc.1" kept
	out := make([]string, 0, len(got))
	for _, r := range got {
		out = append(out, r.raw)
	}
	eqStrings(t, out, []string{"1.2.3", "1.2.3-rc.1"})
}

// * aggregation

func TestAggregateMinor(t *testing.T) {
	// Best per (major,minor)
	tags := []string{
		"1.2.1", "1.2.3", // pick 1.2.3
		"1.3.0-rc.1", "1.3.0", // pick 1.3.0
		"2.0.1", // only one
	}
	sem := parseRecs(t, tags)

	got := aggregateMinor(append([]rec{}, sem...))
	out := make([]string, 0, len(got))
	for _, r := range got {
		out = append(out, r.raw)
	}
	// Order of first-seen groups: (1,2), (1,3), (2,0)
	eqStrings(t, out, []string{"1.2.3", "1.3.0", "2.0.1"})
}

func TestAggregateMajor(t *testing.T) {
	// Best per major
	tags := []string{
		"1.2.1", "1.9.9", // pick 1.9.9 for major 1
		"2.0.0-rc.1", "2.0.0",
		"3.1.0",
	}
	sem := parseRecs(t, tags)

	got := aggregateMajor(append([]rec{}, sem...))
	out := make([]string, 0, len(got))
	for _, r := range got {
		out = append(out, r.raw)
	}
	// Order of first-seen majors: 1, 2, 3
	eqStrings(t, out, []string{"1.9.9", "2.0.0", "3.1.0"})
}

func TestAggregateLatest(t *testing.T) {
	tags := []string{"1.2.3", "1.10.0", "2.0.0-rc.1", "2.0.0"}
	sem := parseRecs(t, tags)
	got := aggregateLatest(append([]rec{}, sem...))
	if len(got) != 1 || got[0].raw != "2.0.0" {
		t.Fatalf("latest got=%v", got)
	}
}

// * sortSemver

func TestSortSemver(t *testing.T) {
	tags := []string{"1.0.0", "1.0.0-rc.1", "2.0.0", "1.10.0"}
	sem := parseRecs(t, tags)

	cp := append([]rec{}, sem...)
	sortSemver(cp, true)
	out := make([]string, 0, len(cp))
	for _, r := range cp {
		out = append(out, r.raw)
	}
	eqStrings(t, out, []string{"1.0.0-rc.1", "1.0.0", "1.10.0", "2.0.0"})

	cp = append([]rec{}, sem...)
	sortSemver(cp, false)
	out = out[:0]
	for _, r := range cp {
		out = append(out, r.raw)
	}
	eqStrings(t, out, []string{"2.0.0", "1.10.0", "1.0.0", "1.0.0-rc.1"})
}
