package handcrankfan

import (
	"context"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
)

// Gear stage dimensions in mm. The crank shaft sits centerDist below the drive shaft, which carries the bevel gear. A
// pair of equal 20-tooth spur gears, side by side in the tray left of the bevel gear, joins them: the crank turns the
// drive shaft at the same speed and the other way round, and the bevel pair doubles it. Both shafts run in a post on
// the tray's left end and in the right wall, and each has a collar outside the post, so the shaft slides in from the
// left and the collar and the crank arm hold it. Every joint keeps a gap of 0.1 to 0.2 mm, as the joint looks just
// before it is tightened.
const (
	spurTeeth   = 20
	spurModule  = 2
	spurThick   = 6.0
	centerDist  = spurModule*spurTeeth + 0.4 // the pitch circles are 40 apart, plus 0.4 for backlash
	crankHeight = axisHeight - centerDist
	gearTip     = (spurTeeth + 2) * spurModule / 2 // the spur gears' tip radius, 22

	postFace = boxLeftX + wallThick // the left post's inner face, X -10.2
	gearX    = postFace + 1         // the spur gears' faces nearest the post
	collarX  = boxLeftX - 0.3       // the shafts' collars end here, 0.3 clear of the post
	collarT  = 3.5

	driveShaftEnd = crankWallX + 6 - 0.2 // the drive shaft ends in a pocket 6 deep in the right wall
)

// shaftWithCollar builds a round shaft of radius 4 from the collar at the left of the post to X end, with a collar of
// radius 6 at its left end. cut runs on the bare shaft first, because Decad refuses to cut after a union with the
// collar.
func (b *builder) shaftWithCollar(z, end float64, cut func(shaft *decad.Body) *decad.Body) *decad.Body {
	f := frame(r3.Vec{X: collarX - collarT, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	// The shaft starts 1 inside the collar so the union has no shared face.
	shaft := b.cylinder(frame(r3.Vec{X: collarX - collarT + 1, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 4, end-(collarX-collarT+1))
	if cut != nil {
		shaft = cut(shaft)
	}
	collar := b.cylinder(f, 6, collarT)
	return b.union(shaft, collar)
}

// spurGear builds a 20-tooth spur gear on the plane at gearX whose normal is +X, centred on the axis at height z. The
// drive gear is turned half a tooth about its axis, so that its tooth meets the crank gear's gap on the line between
// the two axes.
func (b *builder) spurGear(z float64, turned bool) *decad.Body {
	if b.err != nil {
		return nil
	}
	f := frame(r3.Vec{X: gearX, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	gp := spurgear.DefaultParams()
	gp.Module, gp.Teeth, gp.Thickness, gp.BoreDiameter, gp.Placement = spurModule, spurTeeth, spurThick, crankBore, &f
	// Decad cannot fillet the roots of a module-2 spur gear ("loft fillets consume section segment"), so the roots
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
