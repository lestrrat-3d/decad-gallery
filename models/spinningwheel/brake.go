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
	leftStart, leftEnd := -310*s+shift, -285*s+shift+travel/2
	rightStart, rightEnd := -310*s+shift-travel, -285*s+shift-travel/2

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
		b.part("brake-routing-eye-"+side.name, "brake-guide", "fixed", func() *decad.Body {
			centre := r3.Vec{
				X: side.x, Y: -69*s + shift - 5 + 5/math.Sqrt(2),
				Z: z + 5 - 5/math.Sqrt(2),
			}
			axis := r3.Vec{Y: -1, Z: -1}
			axis, _ = axis.Normalize()
			eye := b.annulus(1.5, 2.5, 1.2, axisFrame(centre.Sub(axis.Scale(0.6)), axis))
			attachment := centre.Add(r3.Vec{Y: math.Sqrt(2), Z: -math.Sqrt(2)})
			root := r3.Vec{X: side.x, Y: attachment.Y, Z: 576 * s}
			return b.union(eye, b.span(root, attachment, 0.8))
		})
		b.part("brake-front-pin-"+side.name, "fastener", "fixed", func() *decad.Body {
			centre := r3.Vec{X: side.x, Y: side.start - 3, Z: z}
			pin := b.revolve([]point2{
				{0, 0}, {2.3, 0}, {2.3, 1}, {1.5, 1},
				{1.5, 7}, {2.3, 7}, {2.3, 8}, {0, 8},
			}, frame(centre.Add(r3.Vec{Z: -4}), r3.Vec{X: 1}, r3.Vec{Y: 1}))
			return b.brakeRotate(pin, centre, side.phase)
		})
		if i == 0 {
			b.part("brake-left-anchor", "anchor", "fixed", func() *decad.Body {
				centre := r3.Vec{X: left, Y: leftStart - 3, Z: z}
				clevis := b.brakeClevis(centre)
				stem := b.span(r3.Vec{X: left, Y: -325*s + shift - 6, Z: z},
					r3.Vec{X: left, Y: centre.Y - 5, Z: z}, 0.8)
				return b.union(clevis, stem)
			})
		}
	}

	shaftFront := -350*s + shift - travel
	knobFront := -348*s + shift - travel
	axisOrigin := r3.Vec{X: right, Y: rightStart - 3, Z: z}
	b.part("brake-takeup-screw", "adjustment", "fixed", func() *decad.Body {
		shaft := b.cylinder(2, rightStart-6-shaftFront,
			frame(r3.Vec{X: right, Y: shaftFront, Z: z}, r3.Vec{X: 1}, r3.Vec{Z: -1}))
		flat := b.box(right+1.3, right+3, shaftFront, knobFront+5, z-3, z+3)
		shaft = b.cut(shaft, flat)
		return b.brakeRotate(b.union(shaft, b.brakeClevis(axisOrigin)), axisOrigin, phase)
	})
	b.part("brake-takeup-knob", "adjustment", "fixed", func() *decad.Body {
		knob := b.dAnnulus(2.1, 1.4, 10*s, 5,
			frame(r3.Vec{X: right, Y: knobFront, Z: z}, r3.Vec{X: 1}, r3.Vec{Z: -1}))
		return b.brakeRotate(knob, axisOrigin, phase)
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

func (b *builder) brakeClevis(centre r3.Vec) *decad.Body {
	blank := b.box(-3.5, 3.5, -6, 2.8, -3, 3)
	slot := b.box(-4, 4, -3, 3.8, -0.8, 0.8)
	pin := b.cylinder(1.6, 6, frame(r3.Vec{Z: -3}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	return b.place(b.cut(blank, b.union(slot, pin)), frame(centre, r3.Vec{X: 1}, r3.Vec{Y: 1}))
}
