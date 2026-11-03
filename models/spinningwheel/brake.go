package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

func (b *builder) brake() {
	s, travel, shift := b.p.Scale, b.p.BrakeTravel, b.a.State.HeadShift
	x, z := -140*s, 600*s
	radius := 42*s + 0.6*math.Sqrt(5)
	left, right := x-radius, x+radius
	front := -338*s + shift
	phase := 2 * math.Pi * travel / 0.7
	// brakeShift keeps the take-up clevis behind the control rail at full travel below Scale 1.
	d := brakeShift(s)
	leftStart, leftEnd := -310*s+d+shift, -285*s+d+shift+travel/2
	rightStart, rightEnd := -310*s+d+shift-travel, -285*s+d+shift-travel/2

	hl := b.headLift()
	b.part("brake-rear-crossbar", "frame", "fixed", func() *decad.Body {
		return b.box(x-55*s, x+55*s, -80*s+shift, -60*s+shift, 565*s, 580*s)
	})
	b.part("brake-front-upright", "frame", "fixed", func() *decad.Body {
		return b.box(x-27*s, x+22*s, -338*s+shift, -330*s+shift, 570*s, 590*s)
	})
	b.part("brake-control-rail", "frame", "fixed", func() *decad.Body {
		return b.panel(x-55*s, 590*s, x+55*s, 610*s, 13*s,
			frame(r3.Vec{Y: -325*s + shift}, r3.Vec{X: 1}, r3.Vec{Z: 1}),
			hole{right, z, 2.1})
	})

	b.part("brake-band", "brake-band", "fixed", func() *decad.Body {
		return b.brakeCord(left, right, leftEnd, rightEnd, z, radius, phase)
	})
	for i, side := range []struct {
		name          string
		x, start, end float64
		phase         float64
	}{
		{"left", left, leftStart, leftEnd, 0},
		{"right", right, rightStart, rightEnd, phase},
	} {
		b.part("brake-spring-"+side.name, "spring", "fixed", func() *decad.Body {
			return b.brakeSpring(side.x, side.start, side.end, z, side.phase)
		})
		// Each piece is placed on the head lift before the Union, so Verify reads the stem against the crossbar
		// bore that shares its lift and numbers (routingEyeStem).
		b.liftedPart("brake-routing-eye-"+side.name, "brake-guide", "fixed", func() *decad.Body {
			centre := r3.Vec{
				X: side.x, Y: -69*s + shift - 5 + 5/math.Sqrt(2),
				Z: z + 5 - 5/math.Sqrt(2),
			}
			axis := r3.Vec{Y: -1, Z: -1}
			axis, _ = axis.Normalize()
			eye := b.lifted(b.annulus(1.5, 2.5, 1.2, axisFrame(centre.Sub(axis.Scale(0.6)), axis)), hl)
			y, z0, z1 := b.routingEyeStem()
			return b.union(eye, b.liftCylinder(hl, level(side.x, y, z0), 0.8, z1-z0))
		})
		b.liftedPart("brake-front-pin-"+side.name, "fastener", "fixed", func() *decad.Body {
			centre := r3.Vec{X: side.x, Y: side.start - 3, Z: z}
			return b.liftRevolve(b.brakeTurn(centre, side.phase),
				frame(centre.Add(r3.Vec{Z: -4}), r3.Vec{X: 1}, r3.Vec{Z: 1}), []point2{
					{0, 0}, {2.3, 0}, {2.3, 1}, {1.5, 1},
					{1.5, 7}, {2.3, 7}, {2.3, 8}, {0, 8},
				})
		})
		if i == 0 {
			b.liftedPart("brake-left-anchor", "anchor", "fixed", func() *decad.Body {
				centre := r3.Vec{X: left, Y: leftStart - 3, Z: z}
				clevis := b.brakeClevisOn(hl, centre)
				// The stem runs along +Y from the control rail's blind bore (anchorStem), its tool's numbers.
				y0, _ := b.anchorStem()
				stem := b.liftCylinder(hl, frame(r3.Vec{X: left, Y: y0, Z: z}, r3.Vec{Z: 1}, r3.Vec{X: 1}),
					0.8, centre.Y-5-y0)
				return b.union(clevis, stem)
			})
		}
	}

	shaftFront := -350*s + shift - travel
	knobFront := -348*s + shift - travel
	axisOrigin := r3.Vec{X: right, Y: rightStart - 3, Z: z}
	// The screw, its clevis, its knob and the right front pin share one lift: the head lift turned by the
	// screw's phase about its own axis, every piece placed before its Boolean.
	tl := b.brakeTurn(axisOrigin, phase)
	b.liftedPart("brake-takeup-screw", "adjustment", "fixed", func() *decad.Body {
		shaft := b.liftCylinder(tl, frame(r3.Vec{X: right, Y: shaftFront, Z: z}, r3.Vec{X: 1}, r3.Vec{Z: -1}),
			2, rightStart-6-shaftFront)
		flat := b.localStock(tl, right+1.3, right+3, shaftFront, knobFront+5, z-3, z+3)
		shaft = b.cut(shaft, flat)
		return b.union(shaft, b.brakeClevisOn(tl, axisOrigin))
	})
	b.liftedPart("brake-takeup-knob", "adjustment", "fixed", func() *decad.Body {
		section := b.sectionOn(frame(r3.Vec{X: right, Y: knobFront, Z: z}, r3.Vec{X: 1}, r3.Vec{Z: -1}))
		circle(section, 0, 0, 10*s)
		dOutline(section, 2.1, 1.4)
		return b.liftExtrude(section, 1, 5, tl)
	})
	b.part("brake-insert", "insert", "fixed", func() *decad.Body {
		return b.panel(-4, -4, 4, 4, 4,
			frame(r3.Vec{X: right, Y: front, Z: z}, r3.Vec{X: 1}, r3.Vec{Z: -1}),
			hole{0, 0, 2})
	})
	b.part("brake-insert-retaining-plate", "retainer", "fixed", func() *decad.Body {
		return b.panel(-8, -8, 8, 8, 1,
			frame(r3.Vec{X: right, Y: front - 1, Z: z}, r3.Vec{X: 1}, r3.Vec{Z: -1}),
			hole{0, 0, 2.2}, hole{-6, 0, 1.1}, hole{6, 0, 1.1})
	})
	for i, offset := range []float64{-6, 6} {
		b.part(fmt.Sprintf("brake-insert-mount-%d", i+1), "fastener", "fixed", func() *decad.Body {
			return b.revolve([]point2{
				{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 9}, {0, 9},
			}, frame(r3.Vec{X: right + offset, Y: front - 2, Z: z},
				r3.Vec{X: 1}, r3.Vec{Z: -1}))
		})
	}
}

func (b *builder) brakeEye(radius, wire float64, centre, u, v r3.Vec) *decad.Body {
	s := b.section(true)
	circle(s, radius, 0, wire)
	profile := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, profile,
		decad.ConstructionAxis{Dir: r3.Vec{Z: 1}}, decad.FullRevolution{})
	b.fail("revolve brake eye", err)
	return b.place(body, frame(centre, u, v))
}

func (b *builder) brakeRotate(body *decad.Body, centre r3.Vec, phase float64) *decad.Body {
	if b.err != nil || phase == 0 {
		return body
	}
	xf, err := r3.RotationAround(centre, r3.Vec{Y: 1}, units.Radians(math.Remainder(phase, 2*math.Pi)))
	if err != nil {
		b.fail("create brake phase", err)
		return nil
	}
	body, err = decad.Transform(b.ctx, body, xf)
	b.fail("place brake phase", err)
	return body
}

func (b *builder) brakeSpring(x, y0, y1, z, phase float64) *decad.Body {
	s := b.section(true)
	circle(s, 2.1, 0, 0.4)
	profile := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	coil, err := decad.Coil(b.ctx, b.doc, s, profile,
		decad.ConstructionAxis{Dir: r3.Vec{Z: 1}},
		units.Millimeters((y1-y0-0.8)/6), units.Scalar(6))
	b.fail("coil brake spring", err)
	coil = b.place(coil, frame(r3.Vec{X: x, Y: y0 + 0.4, Z: z},
		r3.Vec{X: 1}, r3.Vec{Z: -1}))
	front, rear := r3.Vec{X: x, Y: y0 - 3, Z: z}, r3.Vec{X: x, Y: y1 + 3, Z: z}
	frontEye := b.brakeEye(2.1, 0.4, front, r3.Vec{X: 1}, r3.Vec{Y: 1})
	rearEye := b.brakeEye(2.1, 0.4, rear, r3.Vec{X: 1}, r3.Vec{Y: 1})
	frontLeg := b.span(front.Add(r3.Vec{X: 2.1}), r3.Vec{X: x + 2.1, Y: y0 + 0.4, Z: z}, 0.4)
	rearLeg := b.span(r3.Vec{X: x + 2.1, Y: y1 - 0.4, Z: z}, rear.Add(r3.Vec{X: 2.1}), 0.4)
	return b.brakeRotate(b.refine(b.union(coil, frontLeg, rearLeg, frontEye, rearEye), 0.12), front, phase)
}

// brakeClevis is a clevis about centre on l, its blank, slot and pin hole placed before they are cut.
func (b *builder) brakeClevisOn(l lift, c r3.Vec) *decad.Body {
	blank := b.localStock(l, c.X-3.5, c.X+3.5, c.Y-6, c.Y+2.8, c.Z-3, c.Z+3)
	slot := b.localStock(l, c.X-4, c.X+4, c.Y-3, c.Y+3.8, c.Z-0.8, c.Z+0.8)
	// The pin hole runs 5 past both cheek faces it opens.
	pin := b.liftCylinder(l, level(c.X, c.Y, c.Z-3-5), 1.6, 6+10)
	return b.cut(blank, b.union(slot, pin))
}

// brakeTurn is the head lift turned by phase about the Y axis through centre, the lift of a part that a
// threaded brake adjuster rotates.
func (b *builder) brakeTurn(centre r3.Vec, phase float64) lift {
	hl := b.headLift()
	if b.err != nil || phase == 0 {
		return hl
	}
	xf, err := r3.RotationAround(centre, r3.Vec{Y: 1}, units.Radians(math.Remainder(phase, 2*math.Pi)))
	if err != nil {
		b.fail("create brake phase", err)
		return lift{}
	}
	if xf, err = xf.Then(hl.xf); err != nil {
		b.fail("compose brake phase", err)
		return lift{}
	}
	return lift{xf: xf}
}

// routingEyeStem is a routing eye's vertical stem on the head lift: its Y, and its Z from the rear crossbar's
// top to the eye. The crossbar's stem bores use the same numbers.
func (b *builder) routingEyeStem() (float64, float64, float64) {
	s := b.p.Scale
	return -69*s + b.a.State.HeadShift - 5 + 5/math.Sqrt(2) + math.Sqrt(2), 576 * s,
		600*s + 5 - 5/math.Sqrt(2) - math.Sqrt(2)
}

// anchorStem is the left anchor stem's span along +Y on the head lift: from 6 inside the control rail's rear
// face to the bore's blind end 3 in front of the left spring. The rail's bore uses the same numbers.
func (b *builder) anchorStem() (float64, float64) {
	s := b.p.Scale
	shift := b.a.State.HeadShift
	return -325*s + shift - 6, -310*s + brakeShift(s) + shift - 3
}
