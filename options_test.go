package rats

import (
	"reflect"
	"testing"
)

func TestParseDepth(t *testing.T) {
	t.Parallel()

	cases := map[string]Depth{
		"":        DepthPatch, // default
		"latest":  DepthLatest,
		"l":       DepthLatest,
		"head":    DepthLatest,
		"max":     DepthLatest,
		"0":       DepthLatest,
		"major":   DepthMajor,
		"maj":     DepthMajor,
		"x":       DepthMajor,
		"1":       DepthMajor,
		"minor":   DepthMinor,
		"min":     DepthMinor,
		"xy":      DepthMinor,
		"2":       DepthMinor,
		"patch":   DepthPatch,
		"pth":     DepthPatch,
		"xyz":     DepthPatch,
		"3":       DepthPatch,
		"unknown": DepthPatch, // fallback
		"  MiN  ": DepthMinor, // case/space-insensitive
	}

	for in, want := range cases {
		if got := ParseDepth(in); got != want {
			t.Fatalf("ParseDepth(%q) = %v; want %v", in, got, want)
		}
	}
}

func TestDepthString(t *testing.T) {
	t.Parallel()

	cases := map[Depth]string{
		DepthPatch:  "patch",
		DepthMinor:  "minor",
		DepthMajor:  "major",
		DepthLatest: "latest",
	}

	for d, want := range cases {
		if got := d.String(); got != want {
			t.Fatalf("Depth(%v).String() = %q; want %q", d, got, want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want Format
	}{
		{"", FormatXYZ}, // default when empty

		// singles
		{"x", FormatX},
		{"xy", FormatXY},
		{"xyz", FormatXYZ},
		{"1", FormatX},
		{"2", FormatXY},
		{"3", FormatXYZ},
		{"major", FormatX},
		{"minor", FormatXY},
		{"patch", FormatXYZ},
		{"maj", FormatX},
		{"min", FormatXY},
		{"pth", FormatXYZ},

		// any/all
		{"any", FormatAll},
		{"all", FormatAll},
		{"*", FormatAll},
		{"a", FormatAll},

		// combos / mixed separators
		{"x-xy", FormatX | FormatXY},
		{"x,xyz", FormatX | FormatXYZ},
		{"xy|xyz", FormatXY | FormatXYZ},
		{"xyz-xy-x", FormatAll},
		{"x+xy+xyz", FormatAll},
		{"  X  /  XY ", FormatX | FormatXY}, // case/space-insensitive

		// unknown tokens -> fallback to FormatXYZ
		{"foo", FormatXYZ},
	}

	for _, tc := range cases {
		if got := ParseFormat(tc.in); got != tc.want {
			t.Fatalf("ParseFormat(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}
}

func TestFormatString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   Format
		want string
	}{
		{FormatX, "x"},
		{FormatXY, "xy"},
		{FormatXYZ, "xyz"},
		{FormatX | FormatXY, "x-xy"},
		{FormatXY | FormatXYZ, "xy-xyz"},
		{FormatX | FormatXYZ, "x-xyz"},
		{FormatAll, "x-xy-xyz"},
		{0, "xyz"}, // zero-mask -> default to "xyz"
	}

	for _, tc := range cases {
		if got := tc.in.String(); got != tc.want {
			t.Fatalf("Format(%v).String() = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseSort(t *testing.T) {
	t.Parallel()

	cases := map[string]SortMode{
		"":           SortNone, // default
		"asc":        SortAsc,
		"ascending":  SortAsc,
		"inc":        SortAsc,
		"increase":   SortAsc,
		"up":         SortAsc,
		"desc":       SortDesc,
		"descending": SortDesc,
		"dec":        SortDesc,
		"decrease":   SortDesc,
		"down":       SortDesc,
		"none":       SortNone,
		"default":    SortNone,
		"asis":       SortNone,
		"unknown":    SortNone,
		"  DeSc  ":   SortDesc, // case/space-insensitive
	}

	for in, want := range cases {
		if got := ParseSort(in); got != want {
			t.Fatalf("ParseSort(%q) = %v; want %v", in, got, want)
		}
	}
}

func TestSortModeString(t *testing.T) {
	t.Parallel()

	cases := map[SortMode]string{
		SortNone: "none",
		SortAsc:  "ascending",
		SortDesc: "descending",
	}

	for m, want := range cases {
		if got := m.String(); got != want {
			t.Fatalf("SortMode(%v).String() = %q; want %q", m, got, want)
		}
	}
}

func TestVPrefixString(t *testing.T) {
	t.Parallel()
	cases := map[VPrefix]string{
		PrefixAny:  "any",
		PrefixV:    "v",
		PrefixNone: "none",
	}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Fatalf("VPrefix(%v).String() = %q; want %q", in, got, want)
		}
	}
}

func TestParseVPrefix(t *testing.T) {
	t.Parallel()
	cases := map[string]VPrefix{
		"":          PrefixAny,
		"any":       PrefixAny,
		"*":         PrefixAny,
		"auto":      PrefixAny,
		"v":         PrefixV,
		"with-v":    PrefixV,
		"require-v": PrefixV,
		"required":  PrefixV,
		"none":      PrefixNone,
		"no-v":      PrefixNone,
		"without-v": PrefixNone,
		"forbidden": PrefixNone,
		"unknown":   PrefixAny, // default
	}

	for in, want := range cases {
		if got := ParseVPrefix(in); got != want {
			t.Fatalf("ParseVPrefix(%q) = %v; want %v", in, got, want)
		}
	}
}

// Sanity check that Options zero-value behaves as documented defaults.
func TestOptionsZeroValue(t *testing.T) {
	t.Parallel()

	var opt Options
	want := Options{
		// zero-values:
		// FilterSemver: false
		// ExcludeSignatures: false
		// ReleaseOnly: false
		// OutputCanonical: false
		// Depth: 0 -> DepthPatch
		// Format: 0 -> treated by code paths as default FormatXYZ where relevant
		// Sort: 0 -> SortNone
	}
	if !reflect.DeepEqual(opt, want) {
		t.Fatalf("zero Options = %#v; want %#v", opt, want)
	}
}

func TestParseNoCanon_StringEmpty(t *testing.T) {
	v, ok := parseSemver("1.2.3", false)
	if !ok || v.String() != "" || v.Canon() != "" {
		t.Fatalf("String/Canon must be empty after ParseNoCanon")
	}
}

func TestParseCanon_StringEmpty(t *testing.T) {
	v, ok := parseSemver("1.2.3", true)
	if !ok || v.String() == "" || v.Canon() == "" {
		t.Fatalf("String/Canon must be not empty after ParseCanon")
	}
}
