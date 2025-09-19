package rats

import (
	"strings"
	"testing"
)

// * toTok

func TestToToken(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  Foo  ", "foo"},
		{"\tBaR\n", "bar"},
		{"MiXeD Case", "mixed case"},
		{"", ""},
		{"   ", ""},
	}

	for _, c := range cases {
		if got := toToken(c.in); got != c.want {
			t.Fatalf("toTok(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

// * splitTokens

func TestSplitTokens_Blank(t *testing.T) {
	// Mix of common separators: comma, pipe, plus, slash, dash, space.
	in := "    "
	got := splitTokens(in)

	assertEqSlice(t, got, nil)
}

func TestSplitTokens_Basic(t *testing.T) {
	// Mix of common separators: comma, pipe, plus, slash, dash, space.
	in := " x,xy|xyz/1-2 3+patch "
	got := splitTokens(in)
	want := []string{"x", "xy", "xyz", "1", "2", "3", "patch"}

	assertEqSlice(t, got, want)
}

func TestSplitTokens_ConsecutiveAndEdges(t *testing.T) {
	in := ",,|//--  ++  "
	got := splitTokens(in)

	if len(got) != 0 {
		t.Fatalf("expected empty tokens, got %v", got)
	}
}

func TestSplitTokens_MixedCaseAndDigits(t *testing.T) {
	// toLower inside splitTokens should normalize case
	in := "X,Xy,XYZ 10|20/30-min+Pth"
	got := splitTokens(in)
	want := []string{"x", "xy", "xyz", "10", "20", "30", "min", "pth"}

	assertEqSlice(t, got, want)
}

func TestSplitTokens_NonASCIISeparators(t *testing.T) {
	// EN dash and EM dash act as separators (non [a-z0-9])
	in := "foo–bar—baz"
	got := splitTokens(in)
	want := []string{"foo", "bar", "baz"}

	assertEqSlice(t, got, want)
}

// Optional micro-fuzz for splitTokens (quick sanity):
func TestSplitTokens_SanityRandomSeparators(t *testing.T) {
	seps := []rune{',', '|', '+', '/', '-', ' '}
	words := []string{"alpha", "BETA", "rc1", "v2", "x", "XY", "xyz", "123"}
	// Build an input string like "alpha<sep>beta<sep>...".
	var b strings.Builder
	for i, w := range words {
		if i > 0 {
			b.WriteRune(seps[i%len(seps)])
		}
		b.WriteString(w)
	}

	got := splitTokens(b.String())
	// Ensure all lowercased words were captured in order
	want := make([]string, 0, len(words))
	for _, w := range words {
		want = append(want, strings.ToLower(w))
	}

	assertEqSlice(t, got, want)
}

// * joinDash

func TestJoinDash(t *testing.T) {
	if got := joinDash(nil); got != "" {
		t.Fatalf("joinDash(nil)=%q, want empty", got)
	}

	if got := joinDash([]string{}); got != "" {
		t.Fatalf("joinDash([])=%q, want empty", got)
	}

	if got := joinDash([]string{"x"}); got != "x" {
		t.Fatalf(`joinDash(["x"])=%q, want "x"`, got)
	}

	got := joinDash([]string{"x", "xy", "xyz"})
	if got != "x-xy-xyz" {
		t.Fatalf(`joinDash(["x","xy","xyz"])=%q, want "x-xy-xyz"`, got)
	}
}

// * capStrings

func TestCapStrings(t *testing.T) {
	orig := []string{"a", "b", "c", "d"}

	// limit <= 0 => unchanged
	if got := capStrings(orig, 0); !equalStrings(got, orig) {
		t.Fatalf("capStrings limit=0 changed slice: got %v want %v", got, orig)
	}
	if got := capStrings(orig, -5); !equalStrings(got, orig) {
		t.Fatalf("capStrings limit<0 changed slice: got %v want %v", got, orig)
	}

	// limit == len => unchanged
	if got := capStrings(orig, len(orig)); !equalStrings(got, orig) {
		t.Fatalf("capStrings limit=len changed slice: got %v want %v", got, orig)
	}

	// limit < len => truncated
	got := capStrings(orig, 2)
	want := []string{"a", "b"}
	assertEqSlice(t, got, want)

	// limit > len => unchanged
	if got := capStrings(orig, 10); !equalStrings(got, orig) {
		t.Fatalf("capStrings limit>len changed slice: got %v want %v", got, orig)
	}
}

func TestIsSigTag(t *testing.T) {
	ok := []string{
		"sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.sig",
		"sha256-0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF.sig",
	}
	bad := []string{
		"",               // empty
		"sha256-xyz.sig", // length
		"sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.sig", // length
		"sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg.sig",                 // 'g'
		"sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.SIG",                 // suffix
		"sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.sic",                 // suffix
		"sha257-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.sig",                 // prefix
	}

	for _, s := range ok {
		if !isSigTag(s) {
			t.Fatalf("want true for %q", s)
		}
	}

	for _, s := range bad {
		if isSigTag(s) {
			t.Fatalf("want false for %q", s)
		}
	}
}

// * helpers

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func assertEqSlice(t *testing.T, got, want []string) {
	t.Helper()
	if !equalStrings(got, want) {
		t.Fatalf("mismatch:\n got=%v\nwant=%v", got, want)
	}
}
