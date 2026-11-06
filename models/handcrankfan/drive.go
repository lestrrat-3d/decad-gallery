package handcrankfan

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
)

// Bevel pair: Module 2, 32 driving teeth and 16 pinion teeth. The heel distances are the pair's:
// the driving heel is Module*16/2 + Module*32/8 and the pinion heel is Module*32/2 + Module*16/8.
const (
	gearModule  = 2
	drivingHeel = 24
	pinionHeel  = 36
	crankBore   = 8
	fanBore     = 6
)

// drive builds the bevel pair, the crank parts and the pinion. The gears sit against the gearbox walls: each wall's
// inner face is 0.2 mm from its gear's back face.
func (b *builder) drive() {
	if b.err != nil {
		return
	}
	b.name = "bevel pair"
	gp := bevelgear.DefaultParams()
	gp.Module, gp.Teeth, gp.MateTeeth, gp.SpiralAngle = gearModule, 32, 16, 0
	gp.BoreDiameter, gp.MateBoreDiameter = crankBore, fanBore
	placement := frame(r3.Vec{X: drivingHeel, Z: axisHeight}, r3.Vec{Y: -1}, r3.Vec{Z: 1})
	gp.Placement = &placement
	pair, err := bevelgear.BuildPair(b.ctx, b.doc, gp)
	b.fail("build bevel pair", err)
	if b.err != nil {
		return
	}

	b.part("driving-gear", "gear", "drive", func() *decad.Body { return pair.Driving })
	b.part("drive-gear", "gear", "drive", func() *decad.Body { return b.spurGear(axisHeight, true) })
	b.part("drive-shaft", "shaft", "drive", func() *decad.Body { return b.shaftWithCollar(axisHeight, driveShaftEnd, nil) })
	b.crank()

	b.part("pinion", "gear", "fan", func() *decad.Body { return pair.Pinion })
}
