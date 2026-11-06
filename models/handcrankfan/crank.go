package handcrankfan

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Crank dimensions in mm. The arm, the pin and the shaft meet in D-shaped stubs, each clamped by one M2 x 10 screw
// and nut. Every joint keeps a gap of 0.1 to 0.2 mm, as the joint looks just before it is tightened, because
// Decad cannot decide a pair of parts whose faces touch.
const (
	crankThrow      = 55.0 // distance from the crank axis to the pin
	armRadius       = 10.0 // the arm is a stadium 20 wide
	armThick        = 8.0
	armStart        = 44.0 // X of the arm's inner face
	handleStart     = 52.2
	handleLength    = 50.0 // a handle for the thumb, index and middle fingers
	gripCounterbore = 3.8  // depth of the pin head's counterbore at the handle's outer end
	pinHeadStart    = handleStart + handleLength - gripCounterbore + 0.2
)

// alongPin is the plane at x whose normal is +X and whose coordinates are (y, z - pin height).
func alongPin(x float64) r3.Frame {
	return frame(r3.Vec{X: x, Z: axisHeight + crankThrow}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

// crank builds the shaft, the arm, the pin, the handle and the two screws of the crank.
func (b *builder) crank() {
	b.part("crank-shaft", "shaft", "crank", b.crankShaft)
	b.part("crank-arm", "arm", "crank", b.crankArm)
	b.part("crank-pin", "pin", "crank", b.crankPin)
	b.part("crank-grip", "grip", "crank", b.crankGrip)
	x, z := r3.Vec{X: 1}, r3.Vec{Z: 1}
	b.part("crank-arm-bolt", "hardware", "crank", func() *decad.Body {
		return b.screw(r3.Vec{X: 52.1, Z: axisHeight}, x.Scale(-1), z, 10, 66.1-62.0, 1)
	})
	b.part("crank-pin-bolt", "hardware", "crank", func() *decad.Body {
		return b.screw(r3.Vec{X: 43.9, Z: axisHeight + crankThrow}, x, z, 10, 62.1-57.9, 1)
	})
}

// slot is a nut slot: a box on f from u0, v0 to u1, v1, 1.9 deep.
func (b *builder) slot(f r3.Frame, v0, v1 float64) *decad.Body {
	return b.panel(f, -2.15, v0, 2.15, v1, 1.9)
}

// crankShaft builds the shaft with a collar and a D-shaped stub. The stub gets its nut slot and screw hole before it
// joins the core, because Decad refuses to cut the slot after the union.
func (b *builder) crankShaft() *decad.Body {
	stub := b.dPrism(alongX(43.5), 4, 3, -1, 6.3)
	stub = b.cut(stub, b.slot(alongX(46.2), -4.5, 2.5))
	stub = b.cut(stub, b.cylinder(alongX(43), 1.2, 7.5))
	round := b.cylinder(alongX(14), 4, 29)
	collar := b.cylinder(alongX(40.3), 6, 3.5)
	core := b.union(round, collar)
	core = b.cut(core, b.cylinder(alongX(41), 1.2, 3))
	return b.union(core, stub)
}

// crankArm builds a stadium 20 wide with rounded faces, a screw hole at each end, and a D pocket for the shaft
// stub from the inner face and one for the pin stub from the outer face.
func (b *builder) crankArm() *decad.Body {
	s := b.section(alongX(armStart))
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
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), 1.2), 1.2))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, 1.2), 1.2))
	}
	arm := b.roundCaps(b.extrude(s, 2, armThick), 1.5)
	arm = b.cut(arm, b.dPrism(alongX(43), 4.15, 3.15, -1, 7))
	return b.cut(arm, b.dPrism(alongPin(46), 4.15, 3.15, 1, 7))
}

// crankPin builds the pin: a D-shaped stub with a nut slot and screw hole, a round shank for the handle and a head
// that sits in the handle's counterbore. The handle slides on over the stub, so nothing on the pin is wider than its
// bore except the head.
func (b *builder) crankPin() *decad.Body {
	stub := b.dPrism(alongPin(46.2), 4, 3, 1, 6.3)
	shank := b.cylinder(alongPin(52.1), 4.5, pinHeadStart+0.4-52.1)
	head := b.cylinder(alongPin(pinHeadStart), 5.8, 3)
	pin := b.union(stub, shank, head)
	pin = b.cut(pin, b.slot(alongPin(48), -2.5, 4.5))
	return b.cut(pin, b.cylinder(alongPin(45), 1.2, 10))
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
	s := b.section(frame(r3.Vec{X: handleStart, Z: axisHeight + crankThrow}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
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
		Origin: r3.Vec{X: handleStart, Z: axisHeight + crankThrow}, Dir: r3.Vec{X: 1}})
}
