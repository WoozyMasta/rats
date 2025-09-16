package rats

import "testing"

func TestNormalizeShorthand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"1", "1.0.0"},
		{"v1", "1.0.0"},
		{"1.2", "1.2.0"},
		{"v1.2", "1.2.0"},
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
		// Non-matching: just strip leading 'v'
		{"v1.2.3-alpha", "1.2.3-alpha"},
		{"foo", "foo"},
	}

	for _, tc := range cases {
		got := normalizeShorthand(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeShorthand(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
