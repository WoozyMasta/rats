package rats

import "regexp"

var (
	// Signature tags, e.g. "sha256-<64-hex>.sig".
	sigRe = regexp.MustCompile(`^sha256-[0-9a-f]{64}\.sig$`)

	// Release-only: exactly X.Y.Z (optional leading "v").
	relXYZ = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

	// Release-only: exactly X.Y (optional leading "v").
	relXY = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

	// Release-only: exactly X (optional leading "v").
	relX = regexp.MustCompile(`^v?(0|[1-9]\d*)$`)
)
