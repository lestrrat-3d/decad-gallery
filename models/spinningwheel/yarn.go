package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// refineHeld refines body only when it holds a mesh coarser than bound.
func (b *builder) refineHeld(body *decad.Body, bound float64) *decad.Body {
	if b.err != nil || body == nil {
		return body
	}
	if _, err := body.Tessellate(b.ctx, units.Millimeters(bound)); err == nil {
		return body
	}
	return b.refine(body, bound)
}

// refine certifies a held Union mesh's surface bound before the head tilt
// places it (decad.Refine; refining after placement is refused).
func (b *builder) refine(body *decad.Body, bound float64) *decad.Body {
	if b.err != nil || body == nil {
		return body
	}
	refined, err := decad.Refine(b.ctx, body, units.Millimeters(bound))
	b.fail("refine held mesh", err)
	return refined
}

func (b *builder) yarn() {
	s := b.p.Scale
	angle := b.a.State.FlyerAngle
	p := r3.Vec{X: 58 * s, Z: 22 * s}
	// The leader's last span ends on the circle R=31*Scale plus a designed 0.01, so the radius0.35 leader stands
	// 0.01 off the pack's outer radius 31*Scale-0.35 instead of touching it tangentially across two lifts.
	radius := 31*s + 0.01
	distanceSquared := p.Dot(p)
	tangent := p.Scale(radius * radius / distanceSquared).Sub(
		r3.Vec{X: -p.Z, Z: p.X}.Scale(radius * math.Sqrt(distanceSquared-radius*radius) / distanceSquared),
	)
	outgoing, ok := tangent.Sub(p).Normalize()
	if !ok {
		b.fail("construct yarn route", fmt.Errorf("invalid pack tangent direction"))
		return
	}
	activeAxis, ok := outgoing.Add(r3.Vec{Y: 1}).Normalize()
	if !ok {
		b.fail("construct yarn route", fmt.Errorf("invalid active guide bisector"))
		return
	}
	frontIncoming, ok := (r3.Vec{X: 58, Y: 3}).Normalize()
	if !ok {
		b.fail("construct yarn route", fmt.Errorf("invalid front strand direction"))
		return
	}
	frontAxis, ok := frontIncoming.Add(r3.Vec{Y: 1}).Normalize()
	if !ok {
		b.fail("construct yarn route", fmt.Errorf("invalid front guide bisector"))
		return
	}

	index := 0
	for _, x := range []float64{-58, 58} {
		for _, y := range []float64{-190, -172, -160, -142, -124, -106} {
			index++
			axis := r3.Vec{Y: 1}
			if x > 0 && y == -160 {
				axis = activeAxis
			}
			b.liftedPart(fmt.Sprintf("yarn-hook-%d", index), "guide", "flyer", func() *decad.Body {
				return b.yarnHook(r3.Vec{X: x * s, Y: y * s, Z: 22 * s}, axis, 10*s, false)
			})
		}
	}
	b.liftedPart("yarn-front-guide", "guide", "flyer", func() *decad.Body {
		return b.yarnHook(r3.Vec{X: 58 * s, Y: -224 * s, Z: 22 * s}, frontAxis, 12*s, true)
	})

	waypoints := []r3.Vec{
		{Y: -405 * s},
		{Y: -335 * s},
		{Y: -227 * s},
		{Y: -227 * s, Z: 22 * s},
		{X: 58 * s, Y: -224 * s, Z: 22 * s},
		{X: 58 * s, Y: -160 * s, Z: 22 * s},
		{X: tangent.X, Y: -160 * s, Z: tangent.Z},
	}
	// Each span is sketched in untilted head coordinates on the head lift before the Union, so the leader shares
	// the lift of the bobbin parts it passes at every head tilt.
	hl := b.headLift()
	b.liftedPart("yarn-leader", "yarn", "flyer", func() *decad.Body {
		spans := make([]*decad.Body, 0, len(waypoints)-1)
		for i := 0; i < len(waypoints)-1; i++ {
			a := b.headPoint(waypoints[i].X, waypoints[i].Y, waypoints[i].Z, angle)
			c := b.headPoint(waypoints[i+1].X, waypoints[i+1].Y, waypoints[i+1].Z, angle)
			d := c.Sub(a)
			spans = append(spans, b.liftCylinder(hl, axisFrame(a, d), 0.35, d.Len()))
		}
		// Refine each held step so the next Union reads its surface bound.
		u := spans[0]
		for _, sp := range spans[1:] {
			u = b.refineHeld(b.union(u, sp), chainTolerance)
		}
		return b.refine(u, 0.12)
	})
}

// hookStemRadius is an ordinary hook's stem radius; its arm seat is bored to the same radius (guideSeats).
const hookStemRadius = 0.9

func (b *builder) yarnHook(centre, axis r3.Vec, stemZ float64, front bool) *decad.Body {
	if b.err != nil {
		return nil
	}
	axis, ok := axis.Normalize()
	if !ok {
		b.fail("construct yarn guide", fmt.Errorf("invalid guide axis"))
		return nil
	}
	down := r3.Vec{Z: -1}
	e, ok := down.Sub(axis.Scale(down.Dot(axis))).Normalize()
	if !ok {
		b.fail("construct yarn guide", fmt.Errorf("guide axis is parallel to attachment direction"))
		return nil
	}
	// Every piece is placed on the flyer lift before the Union, so Verify reads the stem and the ring apart
	// from the arm or crossbar seat that shares the lift. The flyer point (x, y, z) lies at local (x, z, -y).
	fl := b.flyerLift()
	local := func(v r3.Vec) r3.Vec { return r3.Vec{X: v.X, Y: v.Z, Z: -v.Y} }
	attachment := centre.Add(e.Scale(4))
	bottom := r3.Vec{X: attachment.X, Y: attachment.Y, Z: stemZ}
	hook := b.torus(4, 1, -5*math.Pi/6, 5*math.Pi/3, r3.Vec{})
	ringFrame, err := r3.FromFrame(frame(local(centre), local(e), local(axis.Scale(-1))))
	if err != nil {
		b.fail("place yarn guide ring", err)
		return nil
	}
	ringLift, err := ringFrame.Then(fl.xf)
	if err != nil {
		b.fail("place yarn guide ring", err)
		return nil
	}
	hook = b.lifted(hook, lift{xf: ringLift})
	// An ordinary hook's stem is thinner than its wire: at equal radii the stem/torus Union crosses nearly
	// tangentially and its held bound exceeds the 0.12 mm check. The front guide's stem keeps diameter 2.
	stemRadius := hookStemRadius
	if front {
		stemRadius = 1
	}
	// The stem rises along the flyer's +Z, local +Y. The front guide's stem has the wire's own radius, so it stops
	// 0.5 below the wire's centreline: two equal round bars that meet at that centreline touch tangentially there.
	stemTop := attachment.Z
	if front {
		stemTop -= 0.5
	}
	stem := b.liftCylinder(fl, frame(local(bottom), r3.Vec{Z: 1}, r3.Vec{X: 1}), stemRadius, stemTop-stemZ)
	if !front {
		return b.union(hook, stem)
	}
	// The stalk runs along the flyer's +Y, local -Z, into the crossbar.
	stalk := b.liftCylinder(fl, frame(local(bottom), r3.Vec{Y: 1}, r3.Vec{X: 1}), 1, -215*b.p.Scale-bottom.Y)
	// Join the two plain cylinders first (exact prism route), then make the hook
	// the only mesh boolean over its contact, so no boolean runs on a held result.
	return b.union(hook, b.union(stem, stalk))
}
