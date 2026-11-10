package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Crank dimensions in mm. The crank is built at CrankAngle 0, with the arm pointing up (+Z) from C. The arm presses onto
// the crank shaft's D-flat and the pin's hexagonal stub sits in a hexagonal pocket in the arm; one M2 x 10 bolt and nut
// clamp the stub. Fits.go names the clearance classes.
const (
	crankThrow  = 55.0 // distance from the crank axis to the pin
	armRadius   = 10.0 // the arm is a stadium 20 wide
	armFace     = 38.6 // the arm's inner face
	armThick    = 8.0
	armBoreFrom = 37.6 // the D-bore tool starts 1 before the inner face and runs 8, to the floor at Y 45.6
	pinHoleR    = 1.3  // the bolt hole through the arm
	pocketFlats = 7.43 // the arm's hexagonal pocket for the pin's stub: 6.93 plus 0.25 each side
	pocketFrom  = 41.85
	pocketLen   = 5.75
	stubFlats   = 6.93
	stubFrom    = 42.1
	stubLen     = 5.05
	shankR      = 4.5
	shankFrom   = 46.85
	shankTo     = 94.0
	headR       = 5.8
	headFrom    = 93.6
	headLen     = 3.0
	handleFrom  = 46.0 // the handle sketch's origin, 1 behind the handle's inner end
)

// pinAxis is the pin's axis at CrankAngle 0, as a point at Y 0.
var pinAxis = r3.Vec{X: crankOffset, Z: rotorHeight + crankThrow}

// crankFrame is the plane at Y = y across the pin, with U = +X and V = -Z, whose normal is +Y.
func crankFrame(y float64) r3.Frame {
	return frame(r3.Vec{X: crankOffset, Y: y, Z: rotorHeight}, r3.Vec{X: 1}, r3.Vec{Z: -1})
}

// crank builds the arm, the pin, the handle and the pin's bolt and nut.
func (b *builder) crank() {
	b.part("crank-arm", CrankContainerName, "crank", b.crankArm)
	b.part("crank-pin", CrankContainerName, "crank", b.crankPin)
	b.part("crank-handle", CrankContainerName, "crank", b.crankHandle)
	b.hardware(sitesNamed("crank-pin-bolt")[0])
}

// crankArm builds a stadium 20 wide with chamfered faces and a bolt hole at the pin end, the shaft's D-bore from the inner
// face, a hexagonal pocket for the pin's stub from the outer face and the pin bolt's head pocket. In (u, v) the pin end
// lies at v = -55, because V = -Z.
func (b *builder) crankArm() *decad.Body {
	s := b.section(crankFrame(armFace))
	if s != nil {
		top := s.CreatePoint(0, -crankThrow)
		s.AddConstraint(sketch.NewVerticalPoints(s.Origin(), top), sketch.NewVerticalDistance(s.Origin(), top, -crankThrow))
		r1, l1 := s.CreatePoint(armRadius, 0), s.CreatePoint(-armRadius, 0)
		r2, l2 := s.CreatePoint(armRadius, -crankThrow), s.CreatePoint(-armRadius, -crankThrow)
		hubArc, pinArc := s.CreateArc(s.Origin(), r1, l1), s.CreateArc(top, l2, r2)
		right, left := s.CreateLine(r1, r2), s.CreateLine(l2, l1)
		// Equal end radii make the line-arc joins exactly tangent, which the cap chamfer needs.
		s.AddConstraint(sketch.NewRadius(hubArc, armRadius), sketch.NewEqualRadius(hubArc, pinArc),
			sketch.NewTangent(right, hubArc), sketch.NewTangent(right, pinArc),
			sketch.NewTangent(left, hubArc), sketch.NewTangent(left, pinArc))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, pinHoleR), pinHoleR))
	}
	arm := b.extrude(s, 1, armThick)
	arm = b.chamferCaps(arm)
	_, _, crankAxis := b.axes()
	arm = b.cut(arm, b.dBoreTool(crankAxis, armBoreFrom, armThick, boreClearance))
	arm = b.cut(arm, b.hexagon(frame(r3.Vec{X: crankOffset, Y: pocketFrom, Z: rotorHeight + crankThrow},
		r3.Vec{X: 1}, r3.Vec{Z: -1}), pocketFlats, pocketLen))
	return b.fastenerCuts(arm, sitesNamed("crank-pin-bolt")[0], true, false, false)
}

// chamferCaps chamfers the outer loops of both end caps of an extruded body by MinChamfer.
func (b *builder) chamferCaps(body *decad.Body) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body)))
	out, err := decad.Chamfer(b.ctx, body, loops, units.Millimeters(0.5))
	b.fail("chamfer caps", err)
	return out
}

// crankPin builds the pin: a hexagonal stub with a bolt hole and a cavity for the bolt's nut, a round shank for the
// handle and a head that sits in the handle's counterbore.
func (b *builder) crankPin() *decad.Body {
	at := func(y float64) r3.Frame {
		return frame(r3.Vec{X: crankOffset, Y: y, Z: rotorHeight + crankThrow}, r3.Vec{X: 1}, r3.Vec{Z: -1})
	}
	stub := b.hexagon(at(stubFrom), stubFlats, stubLen)
	shank := b.cylinder(at(shankFrom), shankR, shankTo-shankFrom)
	head := b.cylinder(at(headFrom), headR, headLen)
	pin := b.union(stub, shank, head)
	return b.fastenerCuts(pin, sitesNamed("crank-pin-bolt")[0], false, true, true)
}

// handleProfile is the crank handle's meridian: (distance along the pin from the sketch origin, radius). A small barrel
// with a waist and a flared outer end, with a bore for the pin and a counterbore for its head. Every corner is cut by a
// small chamfer, because a chamfered handle is made of cones and cylinders that Decad can decide against the pin, and a
// handle with fillets is not. The origin is 1 behind the inner end, so no axis-aligned segment passes through it.
var handleProfile = [...]point2{
	{1.5, 4.9}, {47.2, 4.9}, {47.2, 6.4}, {50.5, 6.4}, {51, 6.9}, {51, 8.3}, {49.8, 9.3},
	{44, 8}, {37, 7.25}, {23, 8}, {9, 7.8}, {2.2, 7.2}, {1, 6}, {1, 5.4},
}

// crankHandle builds the handle by revolving its meridian about the pin axis. U runs along the pin (+Y) and V is the radius (+X).
func (b *builder) crankHandle() *decad.Body {
	origin := r3.Vec{X: crankOffset, Y: handleFrom, Z: rotorHeight + crankThrow}
	s := b.section(frame(origin, r3.Vec{Y: 1}, r3.Vec{X: 1}))
	b.fail("draw handle meridian", meridianPolygon(s, handleProfile[:]))
	return b.revolve(s, decad.ConstructionAxis{Origin: origin, Dir: r3.Vec{Y: 1}})
}
