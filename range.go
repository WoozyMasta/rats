package rats

import "github.com/woozymasta/semver"

// clipRange (без изменений по сути)
func clipRange(vs []semver.Semver, r Range) []semver.Semver {
	var (
		haveMin, haveMax bool
		minFloor         semver.Semver
		minExclusive     bool
		maxCeil          semver.Semver // strict exclusive ceiling
	)

	if r.Min != "" {
		minFloor, minExclusive, haveMin = compileMin(r.Min, r.MinExclusive, r.IncludePrerelease)
	}
	if r.Max != "" {
		maxCeil, haveMax = compileMaxExclusive(r.Max, r.MaxExclusive)
	}

	keep := vs[:0]
	for _, v := range vs {
		if haveMin {
			cmp := v.Compare(minFloor)
			if minExclusive {
				if cmp <= 0 {
					continue
				}
			} else if cmp < 0 {
				continue
			}
		}

		if haveMax && v.Compare(maxCeil) >= 0 {
			continue
		}
		keep = append(keep, v)
	}

	return keep
}

// compileMin: парсим один раз bound; для шортхэндов строим floor как X.0.0 / X.Y.0,
// при необходимости добавляем prerelease "0" (это >= X.Y.0-0).
func compileMin(raw string, minExclusive bool, includePreAtFloor bool) (semver.Semver, bool, bool) {
	v, ok := semver.Parse(raw)
	if !ok || !v.IsValid() {
		return semver.Semver{}, false, false
	}

	// Шортхэнды: X / X.Y
	if !v.HasPatch() {
		maj, min := v.Major, 0
		if v.HasMinor() {
			min = v.Minor
		}

		if includePreAtFloor {
			// >= X.Y.0-0  => prerelease = "0"
			return makeSemver(maj, min, 0, "0"), minExclusive, true
		}

		return makeSemver(maj, min, 0, ""), minExclusive, true
	}

	// Полный bound — используем как есть.
	return v, minExclusive, true
}

// compileMaxExclusive: переводим верхнюю границу в строго-исключающую «потолочную».
// Шортхэнд X:  excl -> < X.0.0-0;  incl -> < (X+1).0.0-0
// Шортхэнд X.Y: excl -> < X.Y.0-0;  incl -> < X.(Y+1).0-0
// Полный:
//
//	excl -> < v
//	incl -> если pre, то < v.pre.0; иначе < (patch+1)-0
func compileMaxExclusive(raw string, maxExclusive bool) (semver.Semver, bool) {
	v, ok := semver.Parse(raw)
	if !ok || !v.IsValid() {
		return semver.Semver{}, false
	}

	// Шортхэнд X / X.Y
	if !v.HasPatch() {
		maj, min := v.Major, 0
		if v.HasMinor() {
			min = v.Minor
		}
		if maxExclusive {
			// < X.0.0-0 или < X.Y.0-0
			return makeSemver(maj, min, 0, "0"), true
		}
		// inclusive: сдвигаем следующий «бакет» и "-0"
		if !v.HasMinor() {
			return makeSemver(maj+1, 0, 0, "0"), true
		}
		return makeSemver(maj, min+1, 0, "0"), true
	}

	// Полный bound
	if maxExclusive {
		return v, true // < v
	}

	if v.HasPre() {
		// <= 1.2.3-alpha -> < 1.2.3-alpha.0
		return makeSemver(v.Major, v.Minor, v.Patch, v.Prerelease+".0"), true
	}

	// release: <= 1.2.3 -> < 1.2.4-0
	return makeSemver(v.Major, v.Minor, v.Patch+1, "0"), true
}

// makeSemver — лёгкий конструктор Semver без парсинга.
// prerelease — без ведущего '-' (например "0" или "alpha.0").
func makeSemver(maj, min, pat int, prerelease string) semver.Semver {
	flags := semver.FlagHasMajor | semver.FlagHasMinor | semver.FlagHasPatch
	if prerelease != "" {
		flags |= semver.FlagHasPre
	}

	return semver.Semver{
		Major:      maj,
		Minor:      min,
		Patch:      pat,
		Prerelease: prerelease,
		Flags:      flags,
		Valid:      true,
	}
}
