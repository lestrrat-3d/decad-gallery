package handcrankfan

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Crank dimensions in mm. The arm, the pin and the shaft meet in hexagonal stubs, which pass the torque through six
// faces, and the pin's stub is also clamped by one M2 x 10 bolt and nut. Every joint keeps a gap of its fit class (fits.go).
const (
	crankThrow      = 55.0 // distance from the crank axis to the pin
	armRadius       = 10.0 // the arm is a stadium 20 wide
	armThick        = 8.0
	armStart        = crankFace + 4 // X of the arm's inner face
	handleStart     = crankFace + 12 + regularFit
	handleLength    = 50.0 // a handle for the thumb, index and middle fingers
	gripCounterbore = 3.8  // depth of the pin head's counterbore at the handle's outer end
	pinHeadStart    = handleStart + handleLength - gripCounterbore + regularFit
	stubFlats       = 6.9282 // the pin's hexagonal stub is 8 across corners
)

// alongPin is the plane at x whose normal is +X and whose coordinates are (y, z - pin height).
func alongPin(x float64) r3.Frame {
	return frame(r3.Vec{X: x, Z: crankHeight + crankThrow}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

// crank builds the crank gear, the crank shaft, the arm, the pin, the handle and the pin's bolt and nut.
func (b *builder) crank() {
	b.part("crank-gear", "gear", "crank", func() *decad.Body { return b.spurGear(crankHeight, false) })
	b.part("crank-shaft", "shaft", "crank", b.crankShaft)
	b.part("crank-arm", "arm", "crank", b.crankArm)
	b.part("crank-pin", "pin", "crank", b.crankPin)
	b.part("crank-grip", "grip", "crank", b.crankGrip)
	b.hardware(sitesNamed("crank-pin-bolt")[0])
}

// crankShaft builds the crank shaft: a round shaft with a collar at its left end, outside the tray's left post, that
// slides in from the left through the post, the crank gear, the crank-side wall and its bearing block to a
// hexagonal stub for the arm. The collar and the arm, which sits against the block's end, hold it. The flats are cut
// from the bare shaft before the collar joins it, because Decad refuses to cut after the union.
func (b *builder) crankShaft() *decad.Body {
	return b.shaftWithCollar(crankHeight, crankFace+10-tightFit, func(shaft *decad.Body) *decad.Body {
		// The grooves run from the crank gear to the end, so the gear and the arm slide on over the round shaft.
		return b.grooves(shaft, crankSpline, -10, crankFace+11)
	})
}

// crankArm builds a stadium 20 wide with rounded faces, a bolt hole at the pin end, and a hexagonal pocket for the shaft
// stub from the inner face and one for the pin stub from the outer face.
func (b *builder) crankArm() *decad.Body {
	s := b.section(alongCrank(armStart))
	if s != nil {
		top := s.CreatePoint(0, crankThrow)
		s.AddConstraint(sketch.NewVerticalPoints(s.Origin(), top), sketch.NewVerticalDistance(s.Origin(), top, crankThrow))
		r1, l1 := s.CreatePoint(armRadius, 0), s.CreatePoint(-armRadius, 0)
		r2, l2 := s.CreatePoint(armRadius, crankThrow), s.CreatePoint(-armRadius, crankThrow)
		bottomArc, topArc := s.CreateArc(s.Origin(), l1, r1), s.CreateArc(top, r2, l2)
		right, left := s.CreateLine(r1, r2), s.CreateLine(l2, l1)
		// Equal end radii make the line-arc joins exactly tangent, which Decad's loop fillet needs.
		s.AddConstraint(sketch.NewRadius(bottomArc, armRadius), sketch.NewEqualRadius(bottomArc, topArc),
			sketch.NewTangent(right, bottomArc), sketch.NewTangent(right, topArc),
			sketch.NewTangent(left, bottomArc), sketch.NewTangent(left, topArc))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, boltHole), boltHole))
	}
	arm := b.roundCaps(b.extrude(s, 1, armThick), 1.5)
	// The shaft's splined end is a Tight fit in this bore and the bore's floor stands tightFit from the shaft's end. The
	// pin's stub has the same gap on every flat and at its end.
	arm = b.splineCut(arm, crankSpline, crankFace+3, 7, tightFit)
	arm = b.cut(arm, b.hexagon(alongPin(crankFace+7), stubFlats+2*tightFit, 6))
	return b.fastenerCuts(arm, sitesNamed("crank-pin-bolt")[0], true, false, false)
}

// crankPin builds the pin: a hexagonal stub with a bolt hole and a cavity for the bolt's nut, a round shank for the handle and a head
// that sits in the handle's counterbore. The handle slides on over the stub, so nothing on the pin is wider than its
// bore except the head.
func (b *builder) crankPin() *decad.Body {
	stub := b.hexagon(alongPin(crankFace+7+tightFit), stubFlats, 5.3)
	shank := b.cylinder(alongPin(crankFace+12+tightFit), 4.5, pinHeadStart+0.4-(crankFace+12+tightFit))
	head := b.cylinder(alongPin(pinHeadStart), 5.8, 3)
	pin := b.union(stub, shank, head)
	return b.fastenerCuts(pin, sitesNamed("crank-pin-bolt")[0], false, true, true)
}

// handleProfile is the crank handle's meridian: (distance along the pin from the inner end, radius). A small barrel
// with a waist and a flared outer end, with a bore for the pin and a counterbore for its head. Every corner is
// cut by a small chamfer, because a chamfered handle is made of cones and cylinders that Decad can decide against
// the pin, and a handle with fillets is not.
var handleProfile = [...][2]float64{
	{0.5, 4.9}, {46.2, 4.9}, {46.2, 6.4}, {49.5, 6.4}, {50, 6.9}, {50, 8.3}, {48.8, 9.3},
	{43, 8}, {36, 7.25}, {22, 8}, {8, 7.8}, {1.2, 7.2}, {0, 6}, {0, 5.4},
}

// crankGrip builds the handle by revolving its meridian about the pin axis. Every vertex of the meridian is
// dimensioned from the sketch origin, which sits on the axis at the inner end.
func (b *builder) crankGrip() *decad.Body {
	s := b.section(frame(r3.Vec{X: handleStart, Z: crankHeight + crankThrow}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
	if s != nil {
		points := make([]*sketch.Point, len(handleProfile))
		for i, p := range handleProfile {
			points[i] = sketchcheck.DimensionPoint(s, p[0], p[1])
		}
		for i := range points {
			s.CreateLine(points[i], points[(i+1)%len(points)])
		}
	}
	return b.revolve(s, decad.ConstructionAxis{
		Origin: r3.Vec{X: handleStart, Z: crankHeight + crankThrow}, Dir: r3.Vec{X: 1}})
}
