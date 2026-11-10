package handcranksiren

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Rotor and stator dimensions in mm. Ports are the chord-sided openings that chop the air; the rotor's vanes stand on the
// lands between them.
const (
	rotorDiscRadius = 45.0
	rotorFront      = -27.50 // the disc's front face
	rotorDiscThick  = 3.0
	vaneFrom        = -25.50
	vaneLength      = 9.0
	vaneInner       = 8.5
	vaneOuter       = 44.0
	vaneHalfWidth   = 1.0
	hubRadius       = 9.0
	hubLength       = 9.0
	portRIn         = 30.0
	portROut        = 42.0
	portFrom        = -28.50 // the rotor's port tool starts here and runs 5, through the disc
	portLength      = 5.0
	chamfer         = 0.5
)

// discChamfer chamfers the outer loops of both end caps of an extruded disc.
func (b *builder) discChamfer(disc *decad.Body) *decad.Body {
	if b.err != nil || disc == nil {
		return nil
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(disc))).Or(decad.OuterLoopOf(decad.CapEnd(disc)))
	out, err := decad.Chamfer(b.ctx, disc, loops, units.Millimeters(chamfer))
	b.fail("chamfer disc caps", err)
	return out
}

// faceOnR is the plane at Y = y across the rotor axis with U = +X and V = -Z, whose normal is +Y.
func faceOnR(y float64) r3.Frame {
	return frame(r3.Vec{Z: rotorHeight, Y: y}, r3.Vec{X: 1}, r3.Vec{Z: -1})
}

// portsCut cuts the ring of Slots ports, starting at Y from, from disc.
func (b *builder) portsCut(disc *decad.Body, from float64) *decad.Body {
	s := b.section(faceOnR(from))
	port(s, portRIn, portROut, math.Pi/(2*float64(b.p.Slots)))
	return b.cut(disc, b.ring(b.extrude(s, 0, portLength), b.p.Slots))
}

// rotor builds the rotor: a chamfered disc, one vane for each land and a hub, then the ports and the D-bore. The caps are
// chamfered first, because Decad refuses to chamfer them after the port cut.
func (b *builder) rotor() *decad.Body {
	disc := b.cylinder(faceOnR(rotorFront), rotorDiscRadius, rotorDiscThick)
	disc = b.discChamfer(disc)

	half := math.Pi / float64(b.p.Slots)
	u := r3.Vec{X: math.Cos(half), Z: math.Sin(half)}
	vs := b.section(frame(r3.Vec{Z: rotorHeight, Y: vaneFrom}, u, r3.Vec{Y: 1}.Cross(u)))
	rectangle(vs, vaneInner, -vaneHalfWidth, vaneOuter, vaneHalfWidth)
	vanes := b.ring(b.extrude(vs, 0, vaneLength), b.p.Slots)
	hub := b.cylinder(faceOnR(vaneFrom), hubRadius, hubLength)
	body := b.union(disc, b.union(vanes, hub))

	body = b.portsCut(body, portFrom)
	rotor, _, _ := b.axes()
	return b.cut(body, b.dBoreTool(rotor, vaneFrom, 10, boreClearance))
}
