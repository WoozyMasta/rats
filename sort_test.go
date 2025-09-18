package rats

import (
	"reflect"
	"testing"
)

func TestSort_SemverAscDesc(t *testing.T) {
	t.Parallel()

	in := []string{"1.2.3", "1.10.0", "1.2.10", "1.2.3-alpha"}

	// Ascending by SemVer (pre-release < release)
	gotAsc := Sort(in, SortAsc)
	wantAsc := []string{"1.2.3-alpha", "1.2.3", "1.2.10", "1.10.0"}
	if !reflect.DeepEqual(gotAsc, wantAsc) {
		t.Fatalf("Sort asc got %v; want %v", gotAsc, wantAsc)
	}

	gotDesc := Sort(in, SortDesc)
	wantDesc := []string{"1.10.0", "1.2.10", "1.2.3", "1.2.3-alpha"}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Fatalf("Sort desc got %v; want %v", gotDesc, wantDesc)
	}
}

func TestSort_FallbackLex(t *testing.T) {
	t.Parallel()

	// Not all are valid semver -> lexicographic fallback
	in := []string{"z", "a", "1.0.0"}

	gotAsc := Sort(in, SortAsc)
	wantAsc := []string{"1.0.0", "a", "z"}
	if !reflect.DeepEqual(gotAsc, wantAsc) {
		t.Fatalf("lex asc got %v; want %v", gotAsc, wantAsc)
	}

	gotDesc := Sort(in, SortDesc)
	wantDesc := []string{"z", "a", "1.0.0"}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Fatalf("lex desc got %v; want %v", gotDesc, wantDesc)
	}
}

func TestSort_NormalizeShorthand(t *testing.T) {
	t.Parallel()

	// Mixed forms should compare as X.0.0 / X.Y.0 when normalization is enabled
	in := []string{"1", "1.2", "1.2.3"}

	gotAsc := Sort(in, SortAsc)
	wantAsc := []string{"1", "1.2", "1.2.3"}
	if !reflect.DeepEqual(gotAsc, wantAsc) {
		t.Fatalf("normalize asc got %v; want %v", gotAsc, wantAsc)
	}

	gotDesc := Sort(in, SortDesc)
	wantDesc := []string{"1.2.3", "1.2", "1"}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Fatalf("normalize desc got %v; want %v", gotDesc, wantDesc)
	}
}
