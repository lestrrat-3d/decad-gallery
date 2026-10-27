package screwgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// gear is one ribbon placed in the pair frame, where C is the origin, ê is X, k̂ is Y and
// n̂ is Z (see pairFrame). ez is the gear's axis, ex points at the mating gear so that the
// mounting angle means the same thing for both members, and ey = ez × ex completes the
// cross-section frame (û, v̂).
type gear struct {
	p          Params
	origin     r3.Vec
	ex, ey, ez r3.Vec
	mount      float64 // this gear's cross-section angle where the axes cross
	phase      float64 // the tooth phase, the only thing the motion moves
}

// angle is the cross-section's rotation about the axis at station s.
func (g gear) angle(s float64) float64 { return s/g.p.Lambda() + g.mount }

// local maps a pair-frame point to (u, v, s) with the twist undone.
func (g gear) local(pt r3.Vec) (float64, float64, float64) {
	d := pt.Sub(g.origin)
	x, y := d.Dot(g.ex), d.Dot(g.ey)
	s := d.Dot(g.ez)
	c, sn := math.Cos(g.angle(s)), math.Sin(g.angle(s))
	return x*c + y*sn, -x*sn + y*c, s
}

// world is local's inverse: the pair-frame point of section coordinates (u, v) at station s.
func (g gear) world(u, v, s float64) r3.Vec {
	c, sn := math.Cos(g.angle(s)), math.Sin(g.angle(s))
	x, y := u*c-v*sn, u*sn+v*c
	return g.origin.Add(g.ex.Scale(x)).Add(g.ey.Scale(y)).Add(g.ez.Scale(s))
}

// pairFrame is the two gears in the pair frame. The frame is orthonormal: its origin is
// the mechanism's centre C, its X axis is ê (the placement's U), its Y axis is
// k̂ = n̂ × ê (the placement's V) and its Z axis is the common perpendicular n̂ (the
// placement's normal). Working in it keeps the construction's maths independent of where the
// mechanism is placed; world maps a pair-frame point onto the placement.
type pairFrame struct {
	p     Params
	frame r3.Frame
	gears [2]gear
}

// newPairFrame derives the pair from p's placement. Gear A's axis passes through
// C - (A/2)*n̂ and gear B's through C + (A/2)*n̂, their directions sitting at +Sigma/2 and
// -Sigma/2 about n̂ from ê. Each gear's û points at the other gear: +n̂ for A and -n̂ for B.
// Gear B is built at the assembly phase and gear A at zero.
func newPairFrame(p Params) (pairFrame, error) {
	frame, err := p.placementFrame()
	if err != nil {
		return pairFrame{}, fmt.Errorf("screwgear: placement: %w", err)
	}
	a := p.AxisOffset()
	half := p.CrossAngle / 2
	dirA := r3.NewVec(math.Cos(half), math.Sin(half), 0)
	dirB := r3.NewVec(math.Cos(half), -math.Sin(half), 0)
	up := r3.NewVec(0, 0, 1)

	ga := gear{p: p, origin: r3.NewVec(0, 0, -a/2), ex: up, ez: dirA, mount: p.MountAngleA}
	ga.ey = ga.ez.Cross(ga.ex)
	gb := gear{p: p, origin: r3.NewVec(0, 0, a/2), ex: up.Scale(-1), ez: dirB, mount: p.MountAngleB,
		phase: p.AssemblyPhase}
	gb.ey = gb.ez.Cross(gb.ex)
	return pairFrame{p: p, frame: frame, gears: [2]gear{ga, gb}}, nil
}

// world maps a pair-frame point onto the placement.
func (pf pairFrame) world(pt r3.Vec) r3.Vec { return pf.frame.ToWorld(pt) }
