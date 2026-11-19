package clamparm_test

import (
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
)

// hexCorners exempts the six corners of the part's hex pocket: the pocket's axis is the part frame's Z through its
// origin, and its corners lie on the circle of the given radius. They mate with the sharp corners of the M5 hardware
// ("Retained sharp corners and edges" in instructions.md).
func hexCorners(p clamparm.Part, reason string, radius float64) cornercheck.Exempt {
	const band = 0.3
	return cornercheck.Exempt{
		Reason: reason,
		Near:   cornercheck.Annulus(p.Frame.Origin(), p.Frame.N(), radius-band, radius+band),
	}
}
