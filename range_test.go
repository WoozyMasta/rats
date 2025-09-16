package rats

import "testing"

// helper base options for range tests: SemVer gating on, keep prereleases/builds.
func baseRangeOpt() Options {
	return Options{
		FilterSemver:      true,
		ReleaseOnly:       false, // allow prerelease/build in input
		ExcludeSignatures: true,
		Depth:             DepthPatch, // keep all, preserve input order
	}
}

func TestRange_Min_Shorthand_NoPreAtFloor(t *testing.T) {
	t.Parallel()

	in := []string{
		"1.2.0-alpha", "1.2.0", "1.1.9", "1.3.0",
	}
	opt := baseRangeOpt()
	opt.Range = Range{
		Min: "1.2", // floor = 1.2.0 (inclusive), so 1.2.0-alpha is excluded
	}
	got := Filter(in, opt)
	want := []string{"1.2.0", "1.3.0"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Min shorthand (no pre at floor): got %v; want %v", got, want)
	}
}

func TestRange_Min_Shorthand_IncludePreAtFloor(t *testing.T) {
	t.Parallel()

	in := []string{
		"1.2.0-alpha", "1.2.0", "1.1.9", "1.3.0",
	}
	opt := baseRangeOpt()
	opt.Range = Range{
		Min:                    "1.2",
		IncludePrerelease: true, // floor = 1.2.0-0 (inclusive), so alpha is included
	}
	got := Filter(in, opt)
	want := []string{"1.2.0-alpha", "1.2.0", "1.3.0"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("Min shorthand (+pre at floor): got %v; want %v", got, want)
	}
}

func TestRange_Max_Shorthand_Inclusive(t *testing.T) {
	t.Parallel()

	in := []string{
		"1.1.9", "1.2.0-alpha", "1.2.9", "1.3.0-rc1", "1.3.0",
	}
	opt := baseRangeOpt()
	opt.Range = Range{
		Max: "1.2", // inclusive => < 1.3.0-0, so rc1 and 1.3.0 are excluded
	}
	got := Filter(in, opt)
	want := []string{"1.1.9", "1.2.0-alpha", "1.2.9"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("Max shorthand inclusive: got %v; want %v", got, want)
	}
}

func TestRange_Max_FullSemver_Exclusive(t *testing.T) {
	t.Parallel()

	in := []string{
		"1.2.2", "1.2.3-rc1", "1.2.3", "1.2.4-0",
	}
	opt := baseRangeOpt()
	opt.Range = Range{
		Max:          "1.2.3",
		MaxExclusive: true, // < 1.2.3-0 => only 1.2.2 fits
	}
	got := Filter(in, opt)
	want := []string{"1.2.2"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Max full exclusive: got %v; want %v", got, want)
	}
}

func TestRange_BothBounds_WithPreAtFloor(t *testing.T) {
	t.Parallel()

	in := []string{
		"0.9.9", "1.0.0-alpha", "1.5.0", "2.0.0", "2.0.1-rc1",
	}
	opt := baseRangeOpt()
	opt.Range = Range{
		Min:                    "1",
		IncludePrerelease: true,    // >= 1.0.0-0
		Max:                    "2.0.0", // inclusive => < 2.0.1-0
	}
	got := Filter(in, opt)
	want := []string{"1.0.0-alpha", "1.5.0", "2.0.0"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("Both bounds: got %v; want %v", got, want)
	}
}

func TestRange_Min_FullSemver_Exclusive(t *testing.T) {
	t.Parallel()

	in := []string{
		"1.2.3", "1.2.3+build.1", "1.2.4-0",
	}
	opt := baseRangeOpt()
	opt.Range = Range{
		Min:          "1.2.3",
		MinExclusive: true, // > 1.2.3 => only 1.2.4-0 remains (build is equal to 1.2.3)
	}
	got := Filter(in, opt)
	want := []string{"1.2.4-0"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Min full exclusive: got %v; want %v", got, want)
	}
}
