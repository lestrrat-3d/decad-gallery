package bevelgear_test

import (
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
)

// filletPinion is the fan's pinion: module 2, 14 teeth against 28.
func filletPinion() bevelgear.Params {
	p := bevelgear.DefaultParams()
	p.Module, p.Teeth, p.MateTeeth, p.SpiralAngle = 2, 14, 28, 0
	p.EnableBore = false
	return p
}
