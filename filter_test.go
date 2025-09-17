package rats

import (
	"reflect"
	"testing"
)

func TestMatchFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tag   string
		form  Format
		match bool
	}{
		{"1", FormatX, true},
		{"v1", FormatX, true},
		{"1.2", FormatXY, true},
		{"v1.2", FormatXY, true},
		{"1.2.3", FormatXYZ, true},
		{"v1.2.3", FormatXYZ, true},

		{"1.2.3", FormatX, false},
		{"1.2", FormatX, false},
		{"1", FormatXY, false},

		// ReleaseOnly gate: '-' or '+' should reject
		{"1.2.3-alpha", FormatXYZ, false},
		{"1.2.3+build", FormatXYZ, false},
	}

	for _, tc := range cases {
		got := matchFormat(tc.tag, tc.form)
		if got != tc.match {
			t.Fatalf("matchFormat(%q, %v) = %v; want %v", tc.tag, tc.form, got, tc.match)
		}
	}
}

func TestFilter_SignaturesOnly(t *testing.T) {
	t.Parallel()

	in := []string{
		"sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.sig",
		"1.2.3",
		"sha256-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.sig",
		"foo",
	}

	opt := Options{
		ExcludeSignatures: true,
		FilterSemver:      false,
		ReleaseOnly:       false,
		Depth:             DepthPatch,
	}

	got := Filter(in, opt)
	want := []string{"1.2.3", "foo"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filter signatures-only = %v; want %v", got, want)
	}
}

func TestFilter_ReleaseOnly_ShorthandsAndNoPre(t *testing.T) {
	t.Parallel()

	in := []string{
		"1", "1.2", "1.2.3",
		"1.2.3-alpha", "2.0.0+build.1",
		"v2", "v2.1", "v2.1.0",
		"sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.sig",
	}

	opt := Options{
		ExcludeSignatures: true,
		FilterSemver:      true,
		ReleaseOnly:       true,
		Format:            FormatAll,
		Depth:             DepthPatch,
		OutputCanonical:   false,
	}

	got := Filter(in, opt)
	// Order is input order at DepthPatch; prerelease/build and signatures removed
	want := []string{"1", "1.2", "1.2.3", "v2", "v2.1", "v2.1.0"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filter ReleaseOnly = %v; want %v", got, want)
	}
}

func TestFilter_DepthAggregation(t *testing.T) {
	t.Parallel()

	in := []string{
		"1.0.0", "1.0.1", "1.1.0", "1.1.1",
		"2.0.0", "2.0.1", "2.1.0",
	}

	base := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		Format:            FormatXYZ,
		ExcludeSignatures: true,
	}

	// DepthPatch: keep all (order preserved)
	{
		opt := base
		opt.Depth = DepthPatch
		got := Filter(in, opt)
		if !reflect.DeepEqual(got, in) {
			t.Fatalf("DepthPatch got %v; want %v", got, in)
		}
	}

	// DepthMinor: latest per (major,minor)
	{
		opt := base
		opt.Depth = DepthMinor
		got := Filter(in, opt)
		// (1,0)->1.0.1; (1,1)->1.1.1; (2,0)->2.0.1; (2,1)->2.1.0
		// Implementation sorts globally by SemVer desc, so 2.x before 1.x.
		want := []string{"2.1.0", "2.0.1", "1.1.1", "1.0.1"}
		// Note: result order is descending by SemVer within groups.
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("DepthMinor got %v; want %v", got, want)
		}
	}

	// DepthMajor: latest per major
	{
		opt := base
		opt.Depth = DepthMajor
		got := Filter(in, opt)
		// major 1 -> 1.1.1; major 2 -> 2.1.0
		want := []string{"2.1.0", "1.1.1"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("DepthMajor got %v; want %v", got, want)
		}
	}

	// DepthLatest
	{
		opt := base
		opt.Depth = DepthLatest
		got := Filter(in, opt)
		want := []string{"2.1.0"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("DepthLatest got %v; want %v", got, want)
		}
	}
}

func TestFilter_OutputCanonical(t *testing.T) {
	t.Parallel()

	in := []string{"1", "1.2", "1.2.3"}
	opt := Options{
		FilterSemver:    true,
		ReleaseOnly:     true,
		Format:          FormatAll,
		Depth:           DepthPatch,
		OutputCanonical: true,
	}

	got := Filter(in, opt)
	// Canonical form is vMAJOR.MINOR.PATCH (no build), so expect v1.0.0, v1.2.0, v1.2.3
	want := []string{"v1.0.0", "v1.2.0", "v1.2.3"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("OutputCanonical got %v; want %v", got, want)
	}
}

// Validate deduplication behavior across DepthPatch with various flags.
func TestDeduplicate_DepthPatch_Behavior(t *testing.T) {
	t.Parallel()

	in := []string{"1.2.3", "v1.2.3", "1.2.3", "v1.2.3"}

	base := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatXYZ,
		Depth:        DepthPatch,
		VPrefix:      PrefixAny,
	}

	// 1) No dedup, no canonical: duplicates remain as original order.
	{
		opt := base
		opt.Deduplicate = false
		opt.OutputCanonical = false
		got := Filter(in, opt)
		want := []string{"1.2.3", "v1.2.3", "1.2.3", "v1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("no dedup, no canonical got %v; want %v", got, want)
		}
	}

	// 2) Deduplicate enabled: keep the first semantic version occurrence, preserve order.
	{
		opt := base
		opt.Deduplicate = true
		opt.OutputCanonical = false
		got := Filter(in, opt)
		// first seen semver is "1.2.3"
		want := []string{"1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("dedup only got %v; want %v", got, want)
		}
	}

	// 3) Canonical output implies dedup (by implementation): single canonical entry.
	{
		opt := base
		opt.Deduplicate = false
		opt.OutputCanonical = true
		got := Filter(in, opt)
		// canonical renders to vMAJOR.MINOR.PATCH
		want := []string{"v1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("canonical (implied dedup) got %v; want %v", got, want)
		}
	}
}

// Fast path: no SemVer gating, just prefilter + VPrefix policy.
func TestVPrefix_FastPath(t *testing.T) {
	t.Parallel()

	in := []string{"v1.2.3", "1.2.3", "foo"}

	// PrefixV: only entries starting with 'v' pass.
	{
		opt := Options{
			FilterSemver: false,
			ReleaseOnly:  false,
			VPrefix:      PrefixV,
			Depth:        DepthPatch,
		}
		got := Filter(in, opt)
		want := []string{"v1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("PrefixV fast-path got %v; want %v", got, want)
		}
	}

	// PrefixNone: entries starting with 'v' are rejected.
	{
		opt := Options{
			FilterSemver: false,
			ReleaseOnly:  false,
			VPrefix:      PrefixNone,
			Depth:        DepthPatch,
		}
		got := Filter(in, opt)
		want := []string{"1.2.3", "foo"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("PrefixNone fast-path got %v; want %v", got, want)
		}
	}

	// PrefixAny: accept both forms.
	{
		opt := Options{
			FilterSemver: false,
			ReleaseOnly:  false,
			VPrefix:      PrefixAny,
			Depth:        DepthPatch,
		}
		got := Filter(in, opt)
		want := []string{"v1.2.3", "1.2.3", "foo"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("PrefixAny fast-path got %v; want %v", got, want)
		}
	}
}

// SemVer path (ReleaseOnly): verify VPrefix interacts with parsing.
func TestVPrefix_SemverPath_ReleaseOnly(t *testing.T) {
	t.Parallel()

	in := []string{"v1.2.3", "1.2.3", "v1.2.3-alpha"}

	base := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatXYZ,
		Depth:        DepthPatch,
	}

	// PrefixV => keep only "v1.2.3" (alpha is dropped by ReleaseOnly).
	{
		opt := base
		opt.VPrefix = PrefixV
		got := Filter(in, opt)
		want := []string{"v1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("PrefixV semver-path got %v; want %v", got, want)
		}
	}

	// PrefixNone => keep only "1.2.3".
	{
		opt := base
		opt.VPrefix = PrefixNone
		got := Filter(in, opt)
		want := []string{"1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("PrefixNone semver-path got %v; want %v", got, want)
		}
	}

	// PrefixAny => keep both "v1.2.3" and "1.2.3".
	{
		opt := base
		opt.VPrefix = PrefixAny
		got := Filter(in, opt)
		want := []string{"v1.2.3", "1.2.3"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("PrefixAny semver-path got %v; want %v", got, want)
		}
	}
}

func TestLimit_FilterAsIs(t *testing.T) {
	in := []string{"2.0.0", "1.0.0", "1.0.1", "1.0.2", "1.0.3"}
	opt := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatXYZ,
		Depth:        DepthPatch,
		Limit:        2,
	}

	got := Filter(in, opt)
	want := []string{"2.0.0", "1.0.0"}

	if len(got) != opt.Limit {
		t.Fatalf("got %d items, want %d items", len(got), opt.Limit)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestLimit_FilterSorted(t *testing.T) {
	in := []string{"2.0.0", "1.0.0", "1.0.1", "1.0.2", "1.0.3"}
	opt := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatXYZ,
		Depth:        DepthPatch,
		Sort:         SortDesc,
		Limit:        2,
	}

	got := Select(in, opt)
	want := []string{"2.0.0", "1.0.3"}

	if len(got) != opt.Limit {
		t.Fatalf("got %d items, want %d items", len(got), opt.Limit)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestLimit_Disabled(t *testing.T) {
	in := []string{"2.0.0", "1.0.0", "1.0.1", "1.0.2", "1.0.3"}
	opt := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatXYZ,
		Depth:        DepthPatch,
		Limit:        0,
	}

	got := Filter(in, opt)

	if len(got) != len(in) {
		t.Fatalf("got %d items, want %d items", len(got), len(in))
	}

	if !reflect.DeepEqual(got, in) {
		t.Fatalf("got %v want %v", got, in)
	}
}
