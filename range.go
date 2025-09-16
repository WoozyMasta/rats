package rats

import (
	"fmt"
	"strconv"

	"github.com/woozymasta/semver"
)

// clipRange keeps only versions inside the given Range.
// It runs after parsing and before depth aggregation.
func clipRange(vs []semver.Semver, r Range) []semver.Semver {
	var (
		haveMin, haveMax bool
		minFloor         semver.Semver
		minIsExclusive   bool
		maxExclusive     semver.Semver // always strict exclusive bound
	)

	// Compile Min: floor + exclusivity
	if r.Min != "" {
		minFloor, minIsExclusive, haveMin = compileMin(r.Min, r.MinExclusive, r.IncludePrerelease)
	}

	// Compile Max: convert to strict exclusive upper bound
	if r.Max != "" {
		maxExclusive, haveMax = compileMaxExclusive(r.Max, r.MaxExclusive)
	}

	keepAt := 0
	for _, v := range vs {
		if haveMin {
			cmp := v.Compare(minFloor)
			if minIsExclusive {
				if cmp <= 0 {
					continue
				}
			} else {
				if cmp < 0 {
					continue
				}
			}
		}

		if haveMax {
			// v must be strictly less than the exclusive ceiling
			if v.Compare(maxExclusive) >= 0 {
				continue
			}
		}

		vs[keepAt] = v
		keepAt++
	}

	return vs[:keepAt]
}

// compileMin builds the floor and tells whether it's exclusive (> floor) or inclusive (>= floor).
func compileMin(raw string, minExclusive bool, includePreAtFloor bool) (semver.Semver, bool, bool) {
	kind, maj, min, pat, ok := classifyBound(raw)
	if !ok {
		// Full semver bound (may include prerelease)
		if v, ok := semver.ParseNoCanon(raw); ok && v.IsValid() {
			return v, minExclusive, true
		}
		return semver.Semver{}, false, false
	}

	// Build shorthand floor
	var s string
	switch kind {
	case 1: // X
		s = fmt.Sprintf("%d.0.0", maj)
	case 2: // X.Y
		s = fmt.Sprintf("%d.%d.0", maj, min)
	default: // 3: X.Y.Z
		s = fmt.Sprintf("%d.%d.%d", maj, min, pat)
	}

	// Optionally include pre-releases at the floor for X / X.Y
	if includePreAtFloor && kind < 3 {
		s += "-0"
	}

	v, ok := semver.ParseNoCanon(s)
	if !ok || !v.IsValid() {
		return semver.Semver{}, false, false
	}
	return v, minExclusive, true
}

// compileMaxExclusive converts the Max bound to a strict exclusive upper bound.
// If MaxExclusive=false (inclusive), we compute the smallest SemVer greater than Max.
// If MaxExclusive=true (exclusive), we use Max itself as the ceiling.
func compileMaxExclusive(raw string, maxExclusive bool) (semver.Semver, bool) {
	kind, maj, min, pat, ok := classifyBound(raw)
	if !ok {
		// Full SemVer bound
		v, ok := semver.ParseNoCanon(raw)
		if !ok || !v.IsValid() {
			return semver.Semver{}, false
		}

		// < v
		if maxExclusive {
			return v, true
		}

		// <= v -> < next after v
		if v.Prerelease != "" {
			// e.g. <= 1.2.3-alpha -> < 1.2.3-alpha.0
			s := fmt.Sprintf("%d.%d.%d-%s.0", v.Major, v.Minor, v.Patch, v.Prerelease)
			if nv, ok := semver.ParseNoCanon(s); ok && nv.IsValid() {
				return nv, true
			}
		}

		// release: <= 1.2.3 -> < 1.2.4-0
		s := fmt.Sprintf("%d.%d.%d-0", v.Major, v.Minor, v.Patch+1)
		if nv, ok := semver.ParseNoCanon(s); ok && nv.IsValid() {
			return nv, true
		}

		return semver.Semver{}, false
	}

	// Shorthand Max
	var s string
	switch kind {
	case 1: // X
		if maxExclusive {
			s = fmt.Sprintf("%d.0.0-0", maj) // < X.0.0-0
		} else {
			s = fmt.Sprintf("%d.0.0-0", maj+1) // <= X  -> < (X+1).0.0-0
		}

	case 2: // X.Y
		if maxExclusive {
			s = fmt.Sprintf("%d.%d.0-0", maj, min) // < X.Y.0-0
		} else {
			s = fmt.Sprintf("%d.%d.0-0", maj, min+1) // <= X.Y -> < X.(Y+1).0-0
		}

	default: // 3: X.Y.Z
		if maxExclusive {
			s = fmt.Sprintf("%d.%d.%d-0", maj, min, pat) // < X.Y.Z-0
		} else {
			s = fmt.Sprintf("%d.%d.%d-0", maj, min, pat+1) // <= X.Y.Z -> < X.Y.(Z+1)-0
		}
	}

	v, ok := semver.ParseNoCanon(s)
	if !ok || !v.IsValid() {
		return semver.Semver{}, false
	}

	return v, true
}

// classifyBound detects shorthand/full bound and extracts numbers.
// Returns kind: 1 -> X, 2 -> X.Y, 3 -> X.Y.Z.
func classifyBound(s string) (kind, maj, min, pat int, ok bool) {
	if m := relX.FindStringSubmatch(s); m != nil {
		maj, _ = strconv.Atoi(m[1])
		return 1, maj, 0, 0, true
	}

	if m := relXY.FindStringSubmatch(s); m != nil {
		maj, _ = strconv.Atoi(m[1])
		min, _ = strconv.Atoi(m[2])
		return 2, maj, min, 0, true
	}

	if m := relXYZ.FindStringSubmatch(s); m != nil {
		maj, _ = strconv.Atoi(m[1])
		min, _ = strconv.Atoi(m[2])
		pat, _ = strconv.Atoi(m[3])
		return 3, maj, min, pat, true
	}

	return 0, 0, 0, 0, false
}
