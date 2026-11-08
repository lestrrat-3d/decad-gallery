package handcrankfan

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
)

// Bevel pair: Module 2, 28 driving teeth and 14 pinion teeth. The heel distances are the pair's:
// the driving heel is Module*14/2 + Module*28/8 and the pinion heel is Module*28/2 + Module*14/8.
const (
	gearModule  = 2
	drivingHeel = 21
	pinionHeel  = 31.5
)

// drive builds the bevel pair, the crank parts and the pinion. The gears sit against the gearbox walls: each wall's
// inner face is 0.4 mm from its gear's back face.
func (b *builder) drive() {
	if b.err != nil {
		return
	}
	b.name = "bevel pair"
	gp := bevelgear.DefaultParams()
	gp.Module, gp.Teeth, gp.MateTeeth, gp.SpiralAngle = gearModule, 28, 14, 0
	gp.EnableBore = false // the spline bores are cut below
	placement := frame(r3.Vec{X: drivingHeel, Z: axisHeight}, r3.Vec{Y: -1}, r3.Vec{Z: 1})
	gp.Placement = &placement
	pair, err := bevelgear.BuildPair(b.ctx, b.doc, gp)
	b.fail("build bevel pair", err)
	if b.err != nil {
		return
	}

	// The pair's bodies sit in the pair's own containers, and Decad cuts only within one container, so they move to the root.
	b.fail("move the gears to the root", b.doc.Root().AddBody(pair.Driving))
	b.fail("move the gears to the root", b.doc.Root().AddBody(pair.Pinion))
	b.part("driving-gear", "gear", "drive", func() *decad.Body { return b.splineCut(pair.Driving, driveSpline, -5, 30, tightFit+bevelBound) })
	b.part("drive-gear", "gear", "drive", func() *decad.Body { return b.spurGear(axisHeight, true) })
	b.part("drive-shaft", "shaft", "drive", func() *decad.Body {
		return b.shaftWithCollar(axisHeight, driveShaftEnd, func(shaft *decad.Body) *decad.Body {
			return b.grooves(shaft, driveSpline, -10, driveShaftEnd+1)
		})
	})
	b.crank()

	b.part("pinion", "gear", "fan", func() *decad.Body { return b.splineCut(pair.Pinion, fanSpline, -2, 40, tightFit+bevelBound) })
}
