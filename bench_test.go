package rats

import (
	"math/rand"
	"regexp"
	"strconv"
	"testing"
)

// Global sink to avoid compiler eliminating results.
var benchResult []string

// makeTags generates a mixed dataset: semver (with/without pre/build), release shorthands,
// signatures, and junk. Distribution tuned for realistic registry noise.
func makeTags(n int) []string {
	r := rand.New(rand.NewSource(1)) // deterministic
	out := make([]string, n)

	for i := 0; i < n; i++ {
		switch x := r.Intn(100); {
		case x < 55: // full SemVer  X.Y.Z with optional pre/build
			maj := r.Intn(20)
			min := r.Intn(30)
			pat := r.Intn(50)
			s := strconv.Itoa(maj) + "." + strconv.Itoa(min) + "." + strconv.Itoa(pat)

			// ~30% prerelease; ~20% build
			if r.Intn(100) < 30 {
				kind := []string{"alpha", "beta", "rc"}[r.Intn(3)]
				num := r.Intn(12)

				// Sometimes numeric part without dot to exercise comparator behavior
				if r.Intn(2) == 0 {
					s += "-" + kind + "." + strconv.Itoa(num)
				} else {
					s += "-" + kind + strconv.Itoa(num)
				}
			}

			// ~20% meta tags
			if r.Intn(100) < 20 {
				s += "+build." + strconv.Itoa(r.Intn(100))
			}

			// ~20% leading "v"
			if r.Intn(100) < 20 {
				s = "v" + s
			}
			out[i] = s

		case x < 75: // release shorthands X / X.Y / X.Y.Z
			maj := r.Intn(20)
			min := r.Intn(30)
			pat := r.Intn(50)
			switch r.Intn(3) {
			case 0:
				out[i] = strconv.Itoa(maj)

			case 1:
				out[i] = strconv.Itoa(maj) + "." + strconv.Itoa(min)

			default:
				out[i] = strconv.Itoa(maj) + "." + strconv.Itoa(min) + "." + strconv.Itoa(pat)
			}

			if r.Intn(100) < 20 {
				out[i] = "v" + out[i]
			}

		case x < 85: // signatures
			const hexdigits = "0123456789abcdef"
			b := make([]byte, 64)
			for j := range b {
				b[j] = hexdigits[r.Intn(len(hexdigits))]
			}

			out[i] = "sha256-" + string(b) + ".sig"

		default: // junk
			junks := []string{
				"latest", "stable", "dev", "edge", "nightly",
				"foo", "bar", "alpine", "ubuntu", "win",
			}

			out[i] = junks[r.Intn(len(junks))]
		}
	}
	return out
}

// Filter benchmarks

func BenchmarkFilter_ReleaseOnly_DepthMinor(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Format:            FormatAll,
		Depth:             DepthMinor,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_ReleaseOnly_DepthMinor_Canonical(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Format:            FormatAll,
		Depth:             DepthMinor,
		OutputCanonical:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_Semver_WithRange(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,  // accept pre/build
		ReleaseOnly:       false, // not stripping pre/build
		ExcludeSignatures: true,
		Depth:             DepthPatch,
		Range: Range{
			Min:                    "1",
			IncludePrerelease: true, // >= 1.0.0-0
			Max:                    "10.5",
			MaxExclusive:           true, // < 10.5.0-0
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_Semver_WithRange_Canonical(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,  // accept pre/build
		ReleaseOnly:       false, // not stripping pre/build
		ExcludeSignatures: true,
		Depth:             DepthPatch,
		Range: Range{
			Min:                    "1",
			IncludePrerelease: true, // >= 1.0.0-0
			Max:                    "10.5",
			MaxExclusive:           true, // < 10.5.0-0
		},
		OutputCanonical: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

// Sort benchmarks

func BenchmarkSort_Semver_Desc(b *testing.B) {
	b.ReportAllocs()
	// Build a slice of valid SemVer only to avoid lex fallback.
	raw := make([]string, 0, 20000)
	r := rand.New(rand.NewSource(2))

	for len(raw) < cap(raw) {
		maj := r.Intn(100)
		min := r.Intn(100)
		pat := r.Intn(100)
		raw = append(raw, strconv.Itoa(maj)+"."+strconv.Itoa(min)+"."+strconv.Itoa(pat))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Sort mutates a copy internally (it returns a new slice), so no need to clone here.
		benchResult = Sort(raw, SortDesc, false)
	}
}

func BenchmarkSort_Mixed_LexFallback(b *testing.B) {
	b.ReportAllocs()
	raw := append(make([]string, 0, 20000), "z", "a", "foo", "bar")
	raw = append(raw, makeTags(19996)...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Sort(raw, SortDesc, true) // will likely hit lex fallback
	}
}

// End-to-end benchmarks

func BenchmarkSelect_EndToEnd_Noise(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(60000)

	inc := regexp.MustCompile(`^(?:v)?\d+\.\d+(?:\.\d+)?(?:-[A-Za-z0-9.-]+)?(?:\+[A-Za-z0-9.-]+)?$`) // semver-ish
	exc := regexp.MustCompile(`(?:-alpine|-win)$`)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Include:           inc,
		Exclude:           exc,
		Format:            FormatAll,
		Depth:             DepthMinor,
		Sort:              SortDesc,
		Range: Range{
			Min: "1.10", // >= 1.10.0  (release-only => pre/build already excluded)
			Max: "5",    // <= 5.x
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}
