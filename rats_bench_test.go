package rats

import (
	"math/rand"
	"regexp"
	"strconv"
	"testing"
)

// global sink
var benchResult []string

// * helpers

// makeTags independent generator
func makeTags(n int) []string {
	r := rand.New(rand.NewSource(42))
	out := make([]string, n)

	for i := 0; i < n; i++ {
		switch x := r.Intn(100); {
		case x < 55:
			maj := r.Intn(30)
			min := r.Intn(40)
			pat := r.Intn(60)

			s := strconv.Itoa(maj) + "." + strconv.Itoa(min) + "." + strconv.Itoa(pat)

			// ~30% prerelease; ~20% build
			if r.Intn(100) < 30 {
				kind := []string{"alpha", "beta", "rc"}[r.Intn(3)]

				// Sometimes numeric part without dot to exercise comparator behavior
				if r.Intn(2) == 0 {
					s += "-" + kind + "." + strconv.Itoa(r.Intn(10))
				} else {
					s += "-" + kind + strconv.Itoa(r.Intn(10))
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

		case x < 75: // shorthands X / X.Y / X.Y.Z
			maj := r.Intn(30)
			min := r.Intn(40)
			pat := r.Intn(60)

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

		default:
			out[i] = []string{"latest", "stable", "dev", "edge", "foo", "bar"}[r.Intn(6)]
		}
	}

	return out
}

var tagsCount = 5000

func Benchmark_Baseline_CopyOnly(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := make([]string, 0, len(tags))
		out = append(out, tags...)
		benchResult = out
	}
}

func Benchmark_Select_NoWorkFastPath(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	// Этот набор включает твой RAW fast-path:
	// * без SemVer-гейта
	// * нет Range/Dedup/Aggregation/Sort/Canonical
	// * VPrefix=any, нет regex и сигнатур
	opt := Options{
		FilterSemver:      false,
		ReleaseOnly:       false,
		Deduplicate:       false,
		OutputCanonical:   false,
		ExcludeSignatures: false,
		Depth:             DepthPatch,
		Sort:              SortNone,
		VPrefix:           PrefixAny,
		// Range пустой, Limit 0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

// * Filter fast path

// hits the fast path: !FilterSemver && !ReleaseOnly (regex + sig drop only)
func Benchmark_Select_SignaturesOnly(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)
	opt := Options{
		FilterSemver:      false,
		ReleaseOnly:       false,
		ExcludeSignatures: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

func Benchmark_Select_OneCheapRegex(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver: false,
		ReleaseOnly:  false,
		Include:      regexp.MustCompile(`^[A-Za-z0-9.+_-]+$`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

func Benchmark_Select_OneComplexRegex(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver: false,
		ReleaseOnly:  false,
		Exclude:      regexp.MustCompile(`(([2-3]\.){1,2}[0-2]+)(?:-alpine|-windows|-win)$`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

func Benchmark_Select_BothRegex(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver: false,
		ReleaseOnly:  false,
		Include:      regexp.MustCompile(`^[A-Za-z0-9.+_-]+$`),
		Exclude:      regexp.MustCompile(`(([2-3]\.){1,2}[0-2]+)(?:-alpine|-windows|-win)$`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

// * Range variants

func Benchmark_Select_RangeMin(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatAll,
		Depth:        DepthMajor,
		Range:        Range{Min: "1.0.5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

func Benchmark_Select_RangeMax(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatAll,
		Depth:        DepthMajor,
		Range:        Range{Max: "10.0.5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

func Benchmark_Select_RangeBoth(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver: true,
		ReleaseOnly:  true,
		Format:       FormatAll,
		Depth:        DepthMajor,
		Range:        Range{Min: "1.0.5", Max: "10.0.5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

// * Depth variants

func Benchmark_Select_ReleaseOnly_DepthMajor(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Format:            FormatAll,
		Depth:             DepthMajor,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

func Benchmark_Select_ReleaseOnly_DepthLatest(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Format:            FormatAll,
		Depth:             DepthLatest,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

// * Sort variants

func Benchmark_Select_SortSemverAsc(b *testing.B) {
	b.ReportAllocs()
	raw := make([]string, 0, 20000)

	opt := DefaultOptions()
	opt.Sort = SortAsc

	r := rand.New(rand.NewSource(3))
	for len(raw) < cap(raw) {
		maj := r.Intn(100)
		min := r.Intn(100)
		pat := r.Intn(100)
		raw = append(raw, strconv.Itoa(maj)+"."+strconv.Itoa(min)+"."+strconv.Itoa(pat))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(raw, opt)
	}
}

// * Select + Limit

func Benchmark_Select_WithLimit(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(tagsCount)

	opt := DefaultOptions()
	opt.Limit = 50 // trim after sort

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}

// * prefilters

func Benchmark_PrefilterSignatures(b *testing.B) {
	b.ReportAllocs()

	// 75% valid sigs, 25% near-misses
	r := rand.New(rand.NewSource(123))
	in := make([]string, 0, 50000)
	const hex = "0123456789abcdefABCDEF"
	nextHex := func(n int) string {
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = hex[r.Intn(len(hex))]
		}
		return string(buf)
	}

	for i := 0; i < cap(in); i++ {
		if r.Intn(4) != 0 {
			in = append(in, "sha256-"+nextHex(64)+".sig")
		} else {
			in = append(in, "sha256-"+nextHex(63)+".sigX") // miss
		}
	}

	b.ResetTimer()
	n := 0
	for i := 0; i < b.N; i++ {
		for _, s := range in {
			if isSigTag(s) {
				n++
			}
		}
	}

	if n == 0 {
		// prevent dead-code elimination
		b.Fatalf("unexpected zero")
	}
}
