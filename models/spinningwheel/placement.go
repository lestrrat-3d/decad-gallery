package spinningwheel

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// lift is one rigid group's single placement. Every primitive of the group is
// sketched in the group's local coordinates, on an axis-aligned plane, and moved
// by this one transform. A part's position lives in its sketch coordinates and
// its plane's offset, never in a second transform: Decad compares two bodies'
// placements exactly, and two composed transforms round apart even where the
// design puts the faces flush. Sharing the lift lets Verify prove flush faces
// and equal-radius coaxial faces apart from the records alone.
type lift struct{ xf r3.Transform }

// liftOf is the lift that maps local coordinates through frame f.
func (b *builder) liftOf(f r3.Frame) lift {
	if b.err != nil {
		return lift{}
	}
	xf, err := r3.FromFrame(f)
	if err != nil {
		b.fail("create group lift", err)
		return lift{}
	}
	return lift{xf: xf}
}

// tilted composes the head tilt onto l once, so every head part shares the result.
func (b *builder) tilted(l lift) lift {
	if b.err != nil {
		return lift{}
	}
	tilt, err := r3.RotationAround(b.a.State.HeadPivot, r3.Vec{Y: -1}, units.Radians(b.a.State.HeadTilt))
	if err != nil {
		b.fail("create head tilt", err)
		return lift{}
	}
	xf, err := l.xf.Then(tilt)
	if err != nil {
		b.fail("compose head tilt", err)
		return lift{}
	}
	return lift{xf: xf}
}

// level is the local plane through (x, y, z) with the lift's own axes, so a
// body sketched on it extrudes along the lift's local Z.
func level(x, y, z float64) r3.Frame {
	return frame(r3.Vec{X: x, Y: y, Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
}

// raise is f moved along its own normal by d, the start of a body that
// continues another body's extrusion.
func raise(f r3.Frame, d float64) r3.Frame {
	return frame(f.ToWorld(r3.Vec{Z: d}), f.U(), f.V())
}

func (b *builder) lifted(body *decad.Body, l lift) *decad.Body {
	if b.err != nil {
		return nil
	}
	placed, err := decad.Transform(b.ctx, body, l.xf)
	b.fail("place on group lift", err)
	return placed
}

func (b *builder) place(body *decad.Body, f r3.Frame) *decad.Body {
	if b.err != nil {
		return nil
	}

	transform, err := r3.FromFrame(f)
	if err != nil {
		b.fail("create placement transform", err)
		return nil
	}

	placed, err := decad.Transform(b.ctx, body, transform)
	if err != nil {
		b.fail("transform body", err)
		return nil
	}
	return placed
}
