package rats

import sv "github.com/woozymasta/semver"

// parseSemver picks the cheapest semver parser based on whether canonical
// output is needed later. When wantCanonical=false, avoids building Canonical.
func parseSemver(s string, wantCanonical bool) (sv.Semver, bool) {
	if wantCanonical {
		return sv.Parse(s)
	}

	return sv.ParseNoCanon(s)
}
