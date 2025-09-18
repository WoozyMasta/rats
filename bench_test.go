package rats

import (
	"math/rand"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/woozymasta/semver"
)

// global sink
var benchResult []string

// --- helpers -----------------------------------------------------------------

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

func onlyValidSemverMixed(n int) []semver.Semver {
	r := rand.New(rand.NewSource(77))
	out := make([]semver.Semver, 0, n)
	for len(out) < n {
		maj := r.Intn(100)
		min := r.Intn(100)
		pat := r.Intn(100)

		s := strconv.Itoa(maj) + "." + strconv.Itoa(min) + "." + strconv.Itoa(pat)
		if r.Intn(3) == 0 {
			s += "-rc." + strconv.Itoa(r.Intn(10))
		}

		v, ok := semver.Parse(s)
		if ok && v.IsValid() {
			out = append(out, v)
		}
	}

	return out
}

// --- Filter fast path ---------------------------------------------------------

// hits the fast path: !FilterSemver && !ReleaseOnly (regex + sig drop only)
func BenchmarkFilter_FastPath_SignaturesOnly(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(60000)
	opt := Options{
		FilterSemver:      false,
		ReleaseOnly:       false,
		ExcludeSignatures: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_FastPath_OneCheapRegex(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(60000)

	opt := Options{
		FilterSemver: false,
		ReleaseOnly:  false,
		Include:      regexp.MustCompile(`^[A-Za-z0-9.+_-]+$`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_FastPath_OneComplexRegex(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(60000)

	opt := Options{
		FilterSemver: false,
		ReleaseOnly:  false,
		Exclude:      regexp.MustCompile(`(([2-3]\.){1,2}[0-2]+)(?:-alpine|-windows|-win)$`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_FastPath_Full(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(60000)

	opt := Options{
		FilterSemver:      false,
		ReleaseOnly:       false,
		ExcludeSignatures: true,
		Include:           regexp.MustCompile(`^[A-Za-z0-9.+_-]+$`),
		Exclude:           regexp.MustCompile(`(([2-3]\.){1,2}[0-2]+)(?:-alpine|-windows|-win)$`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

// --- Range variants -----------------------------------------------------------

func BenchmarkFilter_RangeMin(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		Format:            FormatAll,
		Depth:             DepthMajor,
		Range:             Range{Min: "1.0.5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_RangeMax(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		Format:            FormatAll,
		Depth:             DepthMajor,
		Range:             Range{Max: "10.0.5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_RangeBoth(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		Format:            FormatAll,
		Depth:             DepthMajor,
		Range:             Range{Min: "1.0.5", Max: "10.0.5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

// --- Depth variants -----------------------------------------------------------

func BenchmarkFilter_ReleaseOnly_DepthMajor(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Format:            FormatAll,
		Depth:             DepthMajor,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

func BenchmarkFilter_ReleaseOnly_DepthLatest(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(50000)

	opt := Options{
		FilterSemver:      true,
		ReleaseOnly:       true,
		ExcludeSignatures: true,
		Format:            FormatAll,
		Depth:             DepthLatest,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Filter(tags, opt)
	}
}

// --- latestPer* internals -----------------------------------------------------

func BenchmarkLatest_PerMajor(b *testing.B) {
	b.ReportAllocs()
	vs := onlyValidSemverMixed(120000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := latestPerMajor(vs)
		_ = res
	}
}

func BenchmarkLatest_PerMinor(b *testing.B) {
	b.ReportAllocs()
	vs := onlyValidSemverMixed(120000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := latestPerMinor(vs)
		_ = res
	}
}

// --- prefilters ---------------------------------------------------------------

func BenchmarkPrefilter_Signatures(b *testing.B) {
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

func BenchmarkPrefilter_VPrefix_RequireV(b *testing.B) {
	b.ReportAllocs()
	in := makeTags(60000)
	opt := Options{VPrefix: PrefixV}

	b.ResetTimer()
	n := 0
	for i := 0; i < b.N; i++ {
		for _, t := range in {
			if prefilterTag(t, opt) {
				n++
			}
		}
	}

	if time.Now().UnixNano() == 0 { // keep n "used"
		b.Log(n)
	}
}

func BenchmarkPrefilter_VPrefix_WithoutV(b *testing.B) {
	b.ReportAllocs()
	in := makeTags(60000)
	opt := Options{VPrefix: PrefixNone}

	b.ResetTimer()
	n := 0
	for i := 0; i < b.N; i++ {
		for _, t := range in {
			if prefilterTag(t, opt) {
				n++
			}
		}
	}

	if time.Now().UnixNano() == 0 {
		b.Log(n)
	}
}

// --- Range compiler branches --------------------------------------------------

func BenchmarkRange_compileMin_ShorthandNoPre(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"1", "2.5", "10", "3.7"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, _, ok := compileMin(s, false, false)
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

func BenchmarkRange_compileMin_ShorthandWithPre(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"1", "2.5", "10", "3.7"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, _, ok := compileMin(s, false, true) // adds -0
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

func BenchmarkRange_compileMin_Full(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"1.2.3", "4.5.6-alpha.1", "7.8.9+build.2"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, _, ok := compileMin(s, true, false)
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

func BenchmarkRange_compileMaxExclusive_ShorthandInclusive(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"2", "10.5"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, ok := compileMaxExclusive(s, false) // inclusive
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

func BenchmarkRange_compileMaxExclusive_ShorthandExclusive(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"2", "10.5"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, ok := compileMaxExclusive(s, true) // exclusive
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

func BenchmarkRange_compileMaxExclusive_FullExclusive(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"1.2.3", "1.2.3-alpha.1"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, ok := compileMaxExclusive(s, true)
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

func BenchmarkRange_compileMaxExclusive_FullInclusive(b *testing.B) {
	b.ReportAllocs()
	cases := []string{"1.2.3", "1.2.3-alpha.1"}

	b.ResetTimer()
	var okCnt int
	for i := 0; i < b.N; i++ {
		for _, s := range cases {
			_, ok := compileMaxExclusive(s, false)
			if ok {
				okCnt++
			}
		}
	}

	if okCnt == 0 {
		b.Fatal("no ok")
	}
}

// --- Sort variants ------------------------------------------------------------

func BenchmarkSort_Semver_Asc(b *testing.B) {
	b.ReportAllocs()
	raw := make([]string, 0, 20000)

	r := rand.New(rand.NewSource(3))
	for len(raw) < cap(raw) {
		maj := r.Intn(100)
		min := r.Intn(100)
		pat := r.Intn(100)
		raw = append(raw, strconv.Itoa(maj)+"."+strconv.Itoa(min)+"."+strconv.Itoa(pat))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Sort(raw, SortAsc)
	}
}

// --- Select + Limit -----------------------------------------------------------

func BenchmarkSelect_WithLimit(b *testing.B) {
	b.ReportAllocs()
	tags := makeTags(80000)

	opt := DefaultOptions()
	opt.Limit = 50 // trim after sort

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchResult = Select(tags, opt)
	}
}
