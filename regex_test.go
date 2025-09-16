package rats

import (
	"reflect"
	"regexp"
	"testing"
)

func TestIncludeExclude_FastPath(t *testing.T) {
	t.Parallel()

	in := []string{"foo", "bar", "1.2.3"}
	opt := Options{
		FilterSemver:      false, // fast path
		ReleaseOnly:       false,
		ExcludeSignatures: true,
		Include:           regexp.MustCompile(`^[a-z]+$`), // only letters
		Exclude:           regexp.MustCompile(`^ba`),      // drop "bar"
		Depth:             DepthPatch,
	}
	got := Filter(in, opt)
	want := []string{"foo"} // "bar" excluded, "1.2.3" fails include
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Include/Exclude fast path: got %v; want %v", got, want)
	}
}

func TestIncludeExclude_SemverPath(t *testing.T) {
	t.Parallel()

	in := []string{
		"1", "1.2", "1.2.3", "1.0.0-rc1", "2.0.0",
		"sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.sig",
	}
	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		Format:            FormatAll,
		ExcludeSignatures: true,
		Include:           regexp.MustCompile(`^v?1(\.|$)`), // only 1, 1.*, v1.*
		Exclude:           regexp.MustCompile(`-rc`),        // drop candidates
		Depth:             DepthPatch,
	}
	got := Filter(in, opt)
	want := []string{"1", "1.2", "1.2.3"} // rc dropped, 2.0.0 excluded by Include, signature dropped
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Include/Exclude semver path: got %v; want %v", got, want)
	}
}
