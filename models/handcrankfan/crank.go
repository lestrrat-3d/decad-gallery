package handcrankfan

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Crank dimensions in mm. The arm, the pin and the shaft meet in hexagonal stubs, which pass the torque through six
// faces, and the pin's stub is also clamped by one M2 x 10 screw and nut. Every joint keeps a gap of 0.1 to 0.2 mm, as the joint looks just before it is tightened, because
// Decad cannot decide a pair of parts whose faces touch.
const (
	crankThrow      = 55.0 // distance from the crank axis to the pin
	armRadius       = 10.0 // the arm is a stadium 20 wide
	armThick        = 8.0
	armStart        = crankFace + 4 // X of the arm's inner face
	handleStart     = crankFace + 12.2
	handleLength    = 50.0 // a handle for the thumb, index and middle fingers
	gripCounterbore = 3.8  // depth of the pin head's counterbore at the handle's outer end
	pinHeadStart    = handleStart + handleLength - gripCounterbore + 0.2
	stubFlats       = 6.9282 // the pin's hexagonal stub is 8 across corners
)

// alongPin is the plane at x whose normal is +X and whose coordinates are (y, z - pin height).
func alongPin(x float64) r3.Frame {
	return frame(r3.Vec{X: x, Z: crankHeight + crankThrow}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

// crank builds the crank gear, the crank shaft, the arm, the pin, the handle and the pin's screw.
func (b *builder) crank() {
	b.part("crank-gear", "gear", "crank", func() *decad.Body { return b.spurGear(crankHeight, false) })
	b.part("crank-shaft", "shaft", "crank", b.crankShaft)
	b.part("crank-arm", "arm", "crank", b.crankArm)
	b.part("crank-pin", "pin", "crank", b.crankPin)
	b.part("crank-grip", "grip", "crank", b.crankGrip)
	x, z := r3.Vec{X: 1}, r3.Vec{Z: 1}
	b.part("crank-pin-bolt", "hardware", "crank", func() *decad.Body {
		return b.screw(r3.Vec{X: crankFace + 3.9, Z: crankHeight + crankThrow}, x, z, 10, 62.1-57.9, 1)
	})
}

// slot is a nut slot: a box on f from u0, v0 to u1, v1, 1.9 deep.
func (b *builder) slot(f r3.Frame, v0, v1 float64) *decad.Body {
	return b.panel(f, -2.15, v0, 2.15, v1, 1.9)
}

// crankShaft builds the crank shaft: a round shaft with a collar at its left end, outside the tray's left post, that
// slides in from the left through the post, the crank gear, the crank-side wall and its bearing block to a
// hexagonal stub for the arm. The collar and the arm, which sits against the block's end, hold it. The flats are cut
// from the bare shaft before the collar joins it, because Decad refuses to cut after the union.
func (b *builder) crankShaft() *decad.Body {
	return b.shaftWithCollar(crankHeight, crankFace+9.8, func(shaft *decad.Body) *decad.Body {
		// The grooves run from the crank gear to the end, so the gear and the arm slide on over the round shaft.
		return b.grooves(shaft, crankSpline, -10, crankFace+11)
	})
}

// crankArm builds a stadium 20 wide with rounded faces, a screw hole at each end, and a hexagonal pocket for the shaft
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
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, 1.2), 1.2))
	}
	arm := b.roundCaps(b.extrude(s, 1, armThick), 1.5)
	// The shaft's hexagonal stub is a tight fit in this bore: the bore is 0.1 larger than the stub on every flat, the gap every joint
	// of the model keeps before it is tightened, and the printed bore is made to grip the stub.
	arm = b.splineCut(arm, crankSpline, crankFace+3, 7)
	return b.cut(arm, b.hexagon(alongPin(crankFace+6), stubFlats+0.2, 7))
}

// crankPin builds the pin: a hexagonal stub with a nut slot and screw hole, a round shank for the handle and a head
// that sits in the handle's counterbore. The handle slides on over the stub, so nothing on the pin is wider than its
// bore except the head.
func (b *builder) crankPin() *decad.Body {
	stub := b.hexagon(alongPin(crankFace+6.2), stubFlats, 6.3)
	shank := b.cylinder(alongPin(crankFace+12.1), 4.5, pinHeadStart+0.4-(crankFace+12.1))
	head := b.cylinder(alongPin(pinHeadStart), 5.8, 3)
	pin := b.union(stub, shank, head)
	pin = b.cut(pin, b.slot(alongPin(crankFace+8), -2.5, 4.5))
	return b.cut(pin, b.cylinder(alongPin(crankFace+5), 1.2, 10))
}

// handleProfile is the crank handle's meridian: (distance along the pin from the inner end, radius). A small barrel
// with a waist and a flared outer end, with a bore for the pin and a counterbore for its head. Every corner is
// cut by a small chamfer, because a chamfered handle is made of cones and cylinders that Decad can decide against
// the pin, and a handle with fillets is not.
var handleProfile = [...][2]float64{
	{0.5, 4.7}, {46.2, 4.7}, {46.2, 6.4}, {49.5, 6.4}, {50, 6.9}, {50, 8.3}, {48.8, 9.3},
	{43, 8}, {36, 7.25}, {22, 8}, {8, 7.8}, {1.2, 7.2}, {0, 6}, {0, 5.2},
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
