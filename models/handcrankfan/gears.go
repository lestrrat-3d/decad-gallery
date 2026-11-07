package handcrankfan

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Gear stage dimensions in mm. The crank shaft sits centerDist below the drive shaft, which carries the bevel gear. A
// pair of equal 20-tooth spur gears, side by side in the tray left of the bevel gear, joins them: the crank turns the
// drive shaft at the same speed and the other way round, and the bevel pair doubles it. Both shafts run in a post on
// the tray's left end and in the right wall, and each has a collar outside the post, so the shaft slides in from the
// left and the collar and the crank arm hold it. Fits.go lists the clearance class of every joint.
const (
	spurTeeth   = 20
	spurModule  = 2
	spurThick   = 6.0
	centerDist  = spurModule * spurTeeth // half the pitch diameter sum: the pitch circles touch, 40 apart
	crankHeight = axisHeight - centerDist

	spurBacklash      = 0.15                                                 // each tooth is thinned this much at the pitch circle, so the pair keeps about 0.3 of play
	gearRootClearance = 0.5                                                  // max(0.25*spurModule, 0.5): a tip clears its mate's root by this much
	gearRootMargin    = 0.02                                                 // measurement margin: the root goes this much deeper and the tip this much lower
	gearTip           = spurModule*spurTeeth/2 + spurModule - gearRootMargin // the spur gears' lowered tip radius, 21.98

	postFace = boxLeftX + wallThick // the left post's inner face, X -10.2
	gearX    = postFace + 1         // the spur gears' faces nearest the post
	collarX  = boxLeftX + 0.2       // the shafts' collars end here, 0.1 clear of the post bearing's face
	collarT  = 4

	driveShaftEnd = crankWallX + 7.1 // the drive shaft ends in the crank-side wall's bearing, 0.2 short of its far face
)

// shaftWithCollar builds a round shaft of radius 3.9 from the collar at the left of the post to X end, with a collar of
// radius 6 at its left end. cut runs on the bare shaft first, because Decad refuses to cut after a union with the
// collar.
func (b *builder) shaftWithCollar(z, end float64, cut func(shaft *decad.Body) *decad.Body) *decad.Body {
	f := frame(r3.Vec{X: collarX - collarT, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	// The shaft starts 1 inside the collar so the union has no shared face.
	shaft := b.cylinder(frame(r3.Vec{X: collarX - collarT + 1, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), shaftRadius, end-(collarX-collarT+1))
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
	gp.Module, gp.Teeth, gp.Thickness, gp.BoreDiameter, gp.Placement = spurModule, spurTeeth, spurThick, 0, &f
	gp.Backlash = spurBacklash
	body, err := spurgear.Variant{Define: spurRootClearance}.Build(b.ctx, b.doc, gp)
	b.fail("build spur gear", err)
	if turned {
		body = b.turn(body, r3.Vec{X: gearX, Z: z}, r3.Vec{X: 1}, math.Pi/spurTeeth)
	}
	// The spline bore is cut last, after the half-tooth turn, so that its teeth line up with the shaft's grooves.
	return b.splineCut(body, shaftAxis{r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}}, gearX-1, spurThick+2)
}

// spurRootClearance deepens the root so a tip clears its mate's root by gearRootClearance plus the margin at center
// distance 40, and lowers the tip by the margin. Lengths go in through SetValue, because a literal length inside an
// expression fails.
func spurRootClearance(t *param.Table) error {
	if err := t.SetValue("gear_root_clearance", units.Millimeters(gearRootClearance)); err != nil {
		return fmt.Errorf("define gear root clearance: %w", err)
	}
	if err := t.SetValue("gear_root_margin", units.Millimeters(gearRootMargin)); err != nil {
		return fmt.Errorf("define gear root margin: %w", err)
	}
	if err := t.SetExpr("root_radius", "pitch_radius - module - gear_root_clearance - gear_root_margin", units.Millimeter); err != nil {
		return fmt.Errorf("define gear root radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "pitch_radius + module - gear_root_margin", units.Millimeter); err != nil {
		return fmt.Errorf("define gear tip radius: %w", err)
	}
	return nil
}
