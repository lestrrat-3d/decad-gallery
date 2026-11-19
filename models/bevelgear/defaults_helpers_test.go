package bevelgear_test

import (
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
)

// spiralDefaults is DefaultParams with the verified 35° spiral teeth, the pair the spiral tests build.
func spiralDefaults() bevelgear.Params {
	p := bevelgear.DefaultParams()
	p.SpiralAngle = 35 * math.Pi / 180
	return p
}
