package rats

import "testing"

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
