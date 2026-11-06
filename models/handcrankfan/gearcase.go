package handcrankfan

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
)

// Gear stage dimensions in mm. The crank shaft sits centerDist below the drive shaft, which carries the bevel gear. A
// pair of equal 35-tooth spur gears joins them, so the crank turns the drive shaft at the same speed, the other way
// round, and the bevel pair doubles it. The spur pair lives in the gear case on the gearbox's right end: a plate that
// serves as the gearbox's crank-side wall, with a rim around the gears, closed by a cover that carries the crank
// shaft's square bearing block. Every joint keeps a gap of 0.1 to 0.2 mm, as the joint looks just before it is
// tightened.
const (
	spurTeeth   = 35
	spurModule  = 2
	spurThick   = 6.0
	centerDist  = 70.4 // the pitch circles are 70 apart, plus 0.4 for backlash
	crankHeight = axisHeight - centerDist

	plateOut     = crankWallX + wallThick // the plate's outer face, X 34
	gearX        = plateOut + 1           // the spur gears' faces nearest the plate
	caseEnd      = gearX + spurThick + 1  // the rim's free end, X 42
	coverIn      = caseEnd + 0.2          // the cover's inner face, 0.2 clear of the rim
	coverThick   = 7.0
	coverOut     = coverIn + coverThick
	crankBlock   = 8.0 // the cover's bearing block sticks out this far
	crankFace    = coverOut + crankBlock
	caseHalfY    = 46.0
	chamberHalfY = 38.2 // 1.2 clear of the gears' tips at 37
	caseBottom   = 56.4
	caseFloor    = caseBottom + 5

	crankShaftStart = plateOut - 6 + 0.2 // the crank shaft ends in a pocket 6 deep in the plate
	driveShaftEnd   = coverIn + 5 - 0.2  // the drive shaft ends in a pocket 5 deep in the cover
	screwLength     = 16
)

// caseScrews are the (Y, Z) positions of the five screws that hold the gear case's plate to the gearbox's right end:
// two in the floor's end and three in the fan-side wall's end. Each head sits in a counterbore in the plate's outer face
// and the screw cuts its own thread in a pilot hole in the tray.
var caseScrews = []point2{{-30, 131}, {0, 131}, {-41.1, 145}, {-41.1, 160}, {-41.1, 175}}

// coverScrews are the four screws that hold the cover to the case's rim, in the rim's front and back walls.
var coverScrews = []point2{{-42.1, 70}, {-42.1, 120}, {42.1, 70}, {42.1, 120}}

// outline is the gear case's outer rectangle in (Y, Z), with its corners rounded.
func outline() []point2 {
	return []point2{{-caseHalfY, caseBottom}, {caseHalfY, caseBottom}, {caseHalfY, boxTop}, {-caseHalfY, boxTop}}
}

// gearCase builds the plate, the rim around the gears and the screw holes.
func (b *builder) gearCase() {
	b.part("gear-case", "housing", "fixed", func() *decad.Body {
		// One block from the plate's inner face to the rim's free end, drawn from the free end toward -X so that
		// roundEnd rounds the inner face. The chamber is cut from the plate's outer face.
		rect := []point2{{caseBottom, -caseHalfY}, {boxTop, -caseHalfY}, {boxTop, caseHalfY}, {caseBottom, caseHalfY}}
		body := b.roundedPrism(frame(r3.Vec{X: caseEnd}, r3.Vec{Z: 1}, r3.Vec{Y: 1}), rect, []float64{3, 3, 3, 3}, caseEnd-crankWallX)
		body = b.roundEnd(body, 1.5)
		chamber := b.roundedPrism(frame(r3.Vec{X: plateOut}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{-chamberHalfY, caseFloor}, {chamberHalfY, caseFloor}, {chamberHalfY, boxTop + 1}, {-chamberHalfY, boxTop + 1}},
			[]float64{3, 3, 0, 0}, caseEnd-plateOut+2)
		body = b.cut(body, chamber)
		if b.err != nil {
			return nil
		}
		body = b.cut(body, b.cylinder(alongX(20), 4.2, 20))
		body = b.cut(body, b.cylinder(alongCrank(crankShaftStart-0.2), 4.2, 7))
		for _, at := range caseScrews {
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: trayEndX, Y: at.u, Z: at.v}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 1.15, plateOut-trayEndX+1))
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: plateOut - 2.2 - 0.1, Y: at.u, Z: at.v}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 2.15, 3.3))
		}
		for _, at := range coverScrews {
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: 30.5, Y: at.u, Z: at.v}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 0.9, caseEnd-30.5+1))
		}
		return body
	})
}

// gearCover builds the cover: a plate that closes the rim, with a pocket for the drive shaft's end, a bore and a
// square bearing block for the crank shaft, and a counterbore and hole for each cover screw.
func (b *builder) gearCover() {
	b.part("gear-cover", "housing", "fixed", func() *decad.Body {
		plate := b.roundedPrism(frame(r3.Vec{X: coverIn}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), outline(), []float64{3, 3, 3, 3}, coverThick)
		// The block starts 1 inside the plate so the union has no shared face.
		block := b.roundedPrism(frame(r3.Vec{X: coverOut - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{-blockHalf, crankHeight - blockHalf}, {blockHalf, crankHeight - blockHalf},
				{blockHalf, crankHeight + blockHalf}, {-blockHalf, crankHeight + blockHalf}},
			[]float64{3, 3, 3, 3}, crankBlock+1)
		block = b.roundEnd(block, 2)
		if b.err != nil {
			return nil
		}
		body := b.union(plate, block)
		body = b.cut(body, b.cylinder(alongCrank(coverIn-0.1), 4.2, crankFace-coverIn+0.2))
		body = b.cut(body, b.cylinder(alongX(coverIn-0.1), 4.2, driveShaftEnd+0.2-coverIn+0.1))
		for _, at := range coverScrews {
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: coverIn - 0.1, Y: at.u, Z: at.v}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 1.15, coverThick+0.2))
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: coverOut - 2.2 - 0.1, Y: at.u, Z: at.v}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 2.15, 3))
		}
		return body
	})
}

// gearBolts builds the screws of the case and the cover. Each is an M2 x 16 self-tapping screw with its head sunk in a
// counterbore and 0.1 clear of the counterbore's floor.
func (b *builder) gearBolts() {
	for i, at := range caseScrews {
		b.part(fmt.Sprintf("case-bolt-%d", i+1), "hardware", "fixed", func() *decad.Body {
			return b.screw(r3.Vec{X: plateOut - 2.2 + 0.1, Y: at.u, Z: at.v}, r3.Vec{X: -1}, r3.Vec{Z: 1}, screwLength, 0, 0.8)
		})
	}
	for i, at := range coverScrews {
		b.part(fmt.Sprintf("cover-bolt-%d", i+1), "hardware", "fixed", func() *decad.Body {
			return b.screw(r3.Vec{X: coverOut - 2.2 + 0.1, Y: at.u, Z: at.v}, r3.Vec{X: -1}, r3.Vec{Z: 1}, screwLength, 0, 0.8)
		})
	}
}

// spurGear builds a 35-tooth spur gear on the plane at x whose normal is +X, centred on the axis at height z. The
// drive gear is turned half a tooth about its axis, so that its tooth meets the crank gear's gap on the line between
// the two axes.
func (b *builder) spurGear(z float64, turned bool) *decad.Body {
	if b.err != nil {
		return nil
	}
	f := frame(r3.Vec{X: gearX, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	gp := spurgear.DefaultParams()
	gp.Module, gp.Teeth, gp.Thickness, gp.BoreDiameter, gp.Placement = spurModule, spurTeeth, spurThick, crankBore, &f
	// Decad cannot fillet the roots of a 35-tooth module-2 gear ("loft fillets consume section segment"), so the roots
	// stay sharp.
	sharp := spurgear.Variant{RootFillet: func(_ context.Context, _ *decad.Document, gear *decad.Body, _ r3.Frame) (*decad.Body, error) {
		return gear, nil
	}}
	body, err := sharp.Build(b.ctx, b.doc, gp)
	b.fail("build spur gear", err)
	if turned {
		body = b.turn(body, r3.Vec{X: gearX, Z: z}, r3.Vec{X: 1}, math.Pi/spurTeeth)
	}
	return body
}
