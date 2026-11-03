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
	radius := 31 * s
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
			b.part(fmt.Sprintf("yarn-hook-%d", index), "guide", "flyer", func() *decad.Body {
				return b.yarnHook(r3.Vec{X: x * s, Y: y * s, Z: 22 * s}, axis, 10*s, false)
			})
		}
	}
	b.part("yarn-front-guide", "guide", "flyer", func() *decad.Body {
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
	b.part("yarn-leader", "yarn", "flyer", func() *decad.Body {
		spans := make([]*decad.Body, 0, len(waypoints)-1)
		for i := 0; i < len(waypoints)-1; i++ {
			a, c := waypoints[i], waypoints[i+1]
			spans = append(spans, b.span(
				b.headPoint(a.X, a.Y, a.Z, angle),
				b.headPoint(c.X, c.Y, c.Z, angle), 0.35,
			))
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
	angle := b.a.State.FlyerAngle
	attachment := centre.Add(e.Scale(4))
	bottom := r3.Vec{X: attachment.X, Y: attachment.Y, Z: stemZ}
	hook := b.torus(4, 1, -5*math.Pi/6, 5*math.Pi/3, r3.Vec{})
	hook = b.place(hook, frame(
		b.headPoint(centre.X, centre.Y, centre.Z, angle),
		headDirection(e, angle), headDirection(axis.Scale(-1), angle),
	))
	// An ordinary hook's stem is thinner than its wire: at equal radii the stem/torus Union crosses nearly
	// tangentially and its held bound exceeds the 0.12 mm check. The front guide's stem keeps diameter 2.
	stemRadius := hookStemRadius
	if front {
		stemRadius = 1
	}
	stem := b.span(
		b.headPoint(bottom.X, bottom.Y, bottom.Z, angle),
		b.headPoint(attachment.X, attachment.Y, attachment.Z, angle), stemRadius,
	)
	if !front {
		return b.union(hook, stem)
	}
	stalk := b.span(
		b.headPoint(bottom.X, bottom.Y, bottom.Z, angle),
		b.headPoint(58*b.p.Scale, -215*b.p.Scale, 12*b.p.Scale, angle), 1,
	)
	// Join the two plain cylinders first (exact prism route), then make the hook
	// the only mesh boolean over its contact, so no boolean runs on a held result.
	return b.union(hook, b.union(stem, stalk))
}
