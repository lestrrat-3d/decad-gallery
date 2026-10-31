package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

func (b *builder) brake() {
	s, travel := b.p.Scale, b.p.BrakeTravel
	shift := b.a.State.HeadShift
	x, z := -140*s, 600*s
	radius := 42*s + 0.6*math.Sqrt(5)
	left, right := x-radius, x+radius
	rearY := -69*s + shift
	leftEnd := -285*s + travel/2 + shift
	rightEnd := -285*s - travel/2 + shift
	leftFront := -310*s + shift
	rightFront := -310*s - travel + shift
	point := func(x, y, z float64) r3.Vec { return r3.Vec{X: x, Y: y, Z: z} }

	b.part("brake-band-wrap", "brake-band", "fixed", func() *decad.Body {
		return b.torus(radius, 0.6, 0, math.Pi, point(x, rearY, 690*s))
	})
	for _, side := range []struct {
		name            string
		x, endY, frontY float64
	}{
		{"left", left, leftEnd, leftFront},
		{"right", right, rightEnd, rightFront},
	} {
		b.part("brake-band-"+side.name+"-drop", "brake-band", "fixed", func() *decad.Body {
			return b.span(point(side.x, rearY, 690*s), point(side.x, rearY, z), 0.6)
		})
		b.part("brake-band-"+side.name+"-forward", "brake-band", "fixed", func() *decad.Body {
			return b.span(point(side.x, rearY, z), point(side.x, side.endY, z), 0.6)
		})
		b.brakeSpring(side.name, side.x, side.frontY, side.endY, z)
		b.part("brake-routing-eye-"+side.name, "brake-guide", "fixed", func() *decad.Body {
			axis, _ := (r3.Vec{Y: -1, Z: -1}).Normalize()
			centre := point(side.x, rearY, z)
			return b.annulus(1.5, 2.5, 1.2, axisFrame(centre.Sub(axis.Scale(0.6)), axis))
		})
		b.part("brake-routing-stem-"+side.name, "brake-guide", "fixed", func() *decad.Body {
			return b.span(point(side.x, rearY+math.Sqrt(2), 580*s),
				point(side.x, rearY+math.Sqrt(2), z-math.Sqrt(2)), 0.8)
		})
	}

	b.part("brake-rear-crossbar", "frame", "fixed", func() *decad.Body {
		return b.box(x-55*s, x+55*s, -80*s+shift, -60*s+shift, 565*s, 580*s)
	})
	b.part("brake-front-upright", "frame", "fixed", func() *decad.Body {
		return b.box(x-27*s, x+22*s, -335*s+shift, -325*s+shift, 570*s, 590*s)
	})
	b.part("brake-control-rail", "frame", "fixed", func() *decad.Body {
		frontY := -338*s + shift
		rail := b.panel(x-55*s, -610*s, x+55*s, -590*s, 13*s,
			frame(point(0, frontY, 0), r3.Vec{X: 1}, r3.Vec{Z: -1}),
			hole{right, -z, 2.1})
		pocket := b.box(right-4.1, right+4.1, frontY-1, -334*s+4+shift, z-4.1, z+4.1)
		return b.cut(rail, pocket)
	})
	b.part("brake-insert", "adjustment", "fixed", func() *decad.Body {
		return b.panel(-4, -4, 4, 4, 4,
			frame(point(right, -334*s+shift, z), r3.Vec{X: 1}, r3.Vec{Z: -1}),
			hole{0, 0, 2})
	})
	b.part("brake-insert-retaining-plate", "adjustment", "fixed", func() *decad.Body {
		return b.panel(-5.5, -5.5, 5.5, 5.5, 1,
			frame(point(right, -338*s-1+shift, z), r3.Vec{X: 1}, r3.Vec{Z: -1}),
			hole{0, 0, 2.2})
	})
	b.part("brake-left-anchor", "brake-guide", "fixed", func() *decad.Body {
		return b.span(point(left, -325*s+shift, z), point(left, leftFront, z), 0.8)
	})

	phase := math.Remainder(2*math.Pi*travel/0.7, 2*math.Pi)
	screwFront, screwRear := -350*s-travel+shift, rightFront-2
	knobFront := -348*s - travel + shift
	b.part("brake-takeup-screw", "adjustment", "fixed", func() *decad.Body {
		local := frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		shaft := b.cylinder(2, screwRear-screwFront, local)
		cap := b.box(1.3, 3, -3, 3, screwRear-(knobFront+5), screwRear-knobFront)
		shaft = b.flatShaft(shaft, cap)
		return b.place(shaft, shaftFrame(right, screwRear, z, phase))
	})
	b.part("brake-takeup-knob", "adjustment", "fixed", func() *decad.Body {
		return b.dAnnulus(2.1, 1.4, 10*s, 5, shaftFrame(right, knobFront+5, z, phase))
	})
	b.part("brake-takeup-eye", "adjustment", "fixed", func() *decad.Body {
		axis := headDirection(r3.Vec{X: 1}, phase)
		centre := point(right, rightFront, z)
		return b.annulus(1.3, 2.5, 1.2, axisFrame(centre.Sub(axis.Scale(0.6)), axis))
	})
}

func (b *builder) brakeSpring(name string, x, y0, y1, z float64) {
	b.part("brake-spring-"+name, "spring", "fixed", func() *decad.Body {
		s := b.section(true)
		circle(s, 2.1, 0, 0.4)
		profile := b.profile(s, 0)
		if b.err != nil {
			return nil
		}
		body, err := decad.Coil(b.ctx, b.doc, s, profile,
			decad.ConstructionAxis{Dir: r3.Vec{Z: 1}},
			units.Millimeters((y1-y0-0.8)/6), units.Scalar(6))
		if err != nil {
			b.fail("coil Scotch brake spring", err)
			return nil
		}
		return b.place(body, frame(r3.Vec{X: x, Y: y0 + 0.4, Z: z},
			r3.Vec{X: 1}, r3.Vec{Z: -1}))
	})
	for _, end := range []struct {
		name     string
		y, coilY float64
	}{
		{"front", y0, y0 + 0.4},
		{"rear", y1, y1 - 0.4},
	} {
		b.part("brake-spring-"+name+"-"+end.name+"-radial-link", "spring", "fixed", func() *decad.Body {
			return b.span(r3.Vec{X: x, Y: end.y, Z: z}, r3.Vec{X: x + 2.1, Y: end.y, Z: z}, 0.4)
		})
		b.part("brake-spring-"+name+"-"+end.name+"-axial-link", "spring", "fixed", func() *decad.Body {
			return b.span(r3.Vec{X: x + 2.1, Y: end.y, Z: z}, r3.Vec{X: x + 2.1, Y: end.coilY, Z: z}, 0.4)
		})
	}
}
