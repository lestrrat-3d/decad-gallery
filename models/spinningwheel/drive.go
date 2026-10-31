package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) drive() {
	s, a := b.p.Scale, math.Remainder(b.p.WheelAngle, 2*math.Pi)
	w := b.a.State.Wheel
	b.part("wheel-rim", "rim", "wheel", func() *decad.Body {
		return b.revolve(scaled(s, point2{255, 0}, point2{280, 0}, point2{280, 7},
			point2{274, 10}, point2{280, 13}, point2{280, 20}, point2{255, 20}), shaftFrame(w.X, 10*s, w.Z, a))
	})
	b.part("wheel-hub", "hub", "wheel", func() *decad.Body {
		return b.dAnnulus(4.1, 3.1, 30*s, 30*s, shaftFrame(w.X, 15*s, w.Z, a))
	})
	for i := range 6 {
		angle := a + float64(i)*math.Pi/3
		b.part(fmt.Sprintf("wheel-spoke-%d", i+1), "spoke", "wheel", func() *decad.Body {
			return b.panel(25*s, -9*s, 260*s, 9*s, 12*s, shaftFrame(w.X, 6*s, w.Z, angle))
		})
	}
	b.part("wheel-axle", "shaft", "wheel", func() *decad.Body {
		body := b.cylinder(4, 148*s, axisFrame(r3.Vec{}, r3.Vec{Z: 1}))
		body = b.flatShaft(body, b.box(3, 5, -5, 5, 0, 12*s), b.box(3, 5, -5, 5, 70*s, 106*s))
		return b.place(body, shaftFrame(w.X, 88*s, w.Z, a))
	})
	b.part("wheel-crank", "crank", "wheel", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -9*s, -9*s, 34*s, 9*s)
		dOutline(section, 4.1, 3.1)
		circle(section, 25*s, 0, 2.6)
		return b.extrude(section, 2, 8*s, shaftFrame(w.X, 86*s, w.Z, a))
	})
	c := b.a.State.CrankPin
	b.part("wheel-crank-pin", "pin", "wheel", func() *decad.Body {
		return b.cylinder(2.5, 34*s+7, axisFrame(r3.Vec{X: c.X, Y: 78*s - 3.5, Z: c.Z}, r3.Vec{Y: 1}))
	})
	for i, y := range []float64{78*s - 3.2, 105*s + 2.7} {
		b.part(fmt.Sprintf("wheel-crank-pin-collar-%d", i+1), "collar", "wheel", func() *decad.Body {
			return b.annulus(2.5, 4.5, 3, axisFrame(r3.Vec{X: c.X, Y: y, Z: c.Z}, r3.Vec{Y: 1}))
		})
	}
	b.driveGimbal()
	b.driveFootman()
	b.driveTreadle()
	b.driveBand()
}

func (b *builder) driveGimbal() {
	s, g := b.p.Scale, b.a.State.GimbalFrame
	b.part("gimbal-crank-605", "bearing", "gimbal", func() *decad.Body {
		return b.annulus(2.5, 7, 5, frame(g.Origin().Sub(g.N().Scale(2.5)), g.U(), g.V()))
	})
	b.part("gimbal-bridge", "gimbal", "gimbal", func() *decad.Body {
		return b.panel(-11*s, -22*s, 10*s, 22*s, 5,
			frame(g.Origin().Sub(g.N().Scale(2.5)), g.U(), g.V()), hole{0, 0, 7})
	})
	for i, offset := range []float64{-4.7, 2.7} {
		b.part(fmt.Sprintf("gimbal-crank-retainer-%d", i+1), "collar", "gimbal", func() *decad.Body {
			return b.annulus(5, 9, 2, frame(g.Origin().Add(g.N().Scale(offset)), g.U(), g.V()))
		})
	}
	for i, offset := range []float64{-22 * s, 15 * s} {
		b.part(fmt.Sprintf("gimbal-cheek-%d", i+1), "gimbal", "gimbal", func() *decad.Body {
			return b.panel(8*s, -9*s, 46*s, 9*s, 7*s,
				frame(g.Origin().Add(g.V().Scale(offset)), g.U(), g.N().Scale(-1)), hole{32 * s, 0, 2.6})
		})
	}
	c := b.a.State.RodHead
	b.part("gimbal-cross-pin", "pin", "gimbal", func() *decad.Body {
		return b.cylinder(2.5, 50*s+6, axisFrame(c.Sub(g.V().Scale(25*s+3)), g.V()))
	})
	for i, offset := range []float64{-22*s - 3.2, 22*s + .2} {
		b.part(fmt.Sprintf("gimbal-cross-pin-collar-%d", i+1), "collar", "gimbal", func() *decad.Body {
			return b.annulus(2.5, 4.5, 3, axisFrame(c.Add(g.V().Scale(offset)), g.V()))
		})
	}
}

func (b *builder) driveFootman() {
	s, r := b.p.Scale, b.a.State.RodFrame
	b.part("footman", "footman", "footman", func() *decad.Body {
		return b.panel(20*s, -4*s, 476*s, 4*s, 20*s,
			frame(r.Origin().Sub(r.V().Scale(10*s)), r.U(), r.N().Scale(-1)), hole{26 * s, 0, 2.2})
	})
	b.part("footman-head-boss", "footman", "footman", func() *decad.Body {
		return b.panel(460*s, -10*s, 498*s, 10*s, 24*s,
			frame(r.Origin().Sub(r.V().Scale(12*s)), r.U(), r.N().Scale(-1)), hole{486 * s, 0, 7})
	})
	c := b.a.State.RodHead
	for i, offset := range []float64{-12 * s, 12*s - 5} {
		b.part(fmt.Sprintf("footman-cross-605-%d", i+1), "bearing", "footman", func() *decad.Body {
			return b.annulus(2.5, 7, 5, axisFrame(c.Add(r.V().Scale(offset)), r.V()))
		})
	}
	b.part("footman-cross-spacer", "spacer", "footman", func() *decad.Body {
		return b.annulus(2.5, 4, 24*s-10, axisFrame(c.Add(r.V().Scale(-12*s+5)), r.V()))
	})
	for i, offset := range []float64{-12*s - 2.2, 12*s + .2} {
		b.part(fmt.Sprintf("footman-cross-retainer-%d", i+1), "collar", "footman", func() *decad.Body {
			return b.annulus(5, 9, 2, axisFrame(c.Add(r.V().Scale(offset)), r.V()))
		})
	}
	d, p := b.a.State.Toe, b.a.State.PedalFrame
	lower, upper := d.Sub(p.N().Scale(12*s)), d.Add(r.U().Scale(26*s))
	b.part("treadle-connector-lower", "compliant", "treadle", func() *decad.Body {
		return b.cut(b.span(d.Sub(p.N().Scale(16*s)), d, 3),
			b.cylinder(2.2, 24*s, axisFrame(lower.Sub(p.U().Scale(12*s)), p.U())))
	})
	b.part("treadle-connector-upper", "compliant", "footman", func() *decad.Body {
		return b.cut(b.span(d, d.Add(r.U().Scale(32*s)), 3),
			b.cylinder(2.2, 20*s, axisFrame(upper.Sub(r.V().Scale(10*s)), r.V())))
	})
	for _, clamp := range []struct {
		name, motion string
		centre, axis r3.Vec
		length, face float64
	}{
		{"treadle-lower-clamp", "treadle", lower, p.U(), 30*s + 4, 12 * s},
		{"footman-upper-clamp", "footman", upper, r.V(), 24*s + 4, 10 * s},
	} {
		b.part(clamp.name+"-pin", "pin", clamp.motion, func() *decad.Body {
			return b.cylinder(2, clamp.length, axisFrame(clamp.centre.Sub(clamp.axis.Scale(clamp.length/2)), clamp.axis))
		})
		b.part(clamp.name+"-head", "fastener", clamp.motion, func() *decad.Body {
			return b.cylinder(3.5, 2, axisFrame(clamp.centre.Sub(clamp.axis.Scale(clamp.face+2.2)), clamp.axis))
		})
		b.part(clamp.name+"-nut", "fastener", clamp.motion, func() *decad.Body {
			return b.annulus(2, 3.5, 2, axisFrame(clamp.centre.Add(clamp.axis.Scale(clamp.face+.2)), clamp.axis))
		})
	}
}

func (b *builder) driveTreadle() {
	s, p := b.p.Scale, b.a.State.PedalFrame
	at := func(x, y, z float64) r3.Vec { return p.ToWorld(r3.Vec{X: x * s, Y: y * s, Z: z * s}) }
	b.part("treadle", "treadle", "treadle", func() *decad.Body {
		return b.panel(5*s, -30*s, 115*s, 240*s, 24*s, frame(at(0, 0, -12), p.U(), p.V()))
	})
	b.part("treadle-heel-rail", "treadle", "treadle", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -12*s, -12*s, 12*s, 12*s)
		dOutline(section, 4.1, 3.1)
		return b.extrude(section, 1, 426*s, frame(at(-233, 0, 0), p.V(), p.N()))
	})
	b.part("treadle-actuating-rail", "treadle", "treadle", func() *decad.Body {
		start, end := at(-160, 0, 0), at(170, 450, 0)
		u, _ := end.Sub(start).Normalize()
		v := p.N().Cross(u)
		return b.panel(0, -12*s, end.Sub(start).Len(), 12*s, 24*s, frame(start.Sub(p.N().Scale(12*s)), u, v))
	})
	b.part("treadle-rear-crossarm", "treadle", "treadle", func() *decad.Body {
		return b.panel(160*s, 438*s, 220*s, 462*s, 12*s, frame(at(0, 0, -12), p.U(), p.V()))
	})
	b.part("treadle-connector-block", "treadle", "treadle", func() *decad.Body {
		return b.panel(438*s, -12*s, 462*s, 12*s, 24*s, frame(at(208, 0, 0), p.V(), p.N()), hole{450 * s, 8 * s, 2.2})
	})
	b.part("heel-pin", "shaft", "treadle", func() *decad.Body {
		body := b.cylinder(4, 526*s, axisFrame(r3.Vec{}, r3.Vec{Z: 1}))
		body = b.flatShaft(body, b.box(3, 5, -5, 5, 49*s, 477*s))
		return b.place(body, frame(at(-283, 0, 0), p.V(), p.N()))
	})
	for i, x := range []float64{-260*s - 6.7, 220*s + 3.7} {
		b.part(fmt.Sprintf("heel-collar-%d", i+1), "collar", "treadle", func() *decad.Body {
			return b.annulus(4, 6, 3, frame(p.ToWorld(r3.Vec{X: x}), p.V(), p.N()))
		})
	}
	for i, interval := range [][2]float64{{-260*s + 3.7, -233*s - .2}, {193*s + .2, 220*s - 3.7}} {
		b.part(fmt.Sprintf("heel-spacer-%d", i+1), "spacer", "treadle", func() *decad.Body {
			return b.annulus(4, 5.5, interval[1]-interval[0], frame(p.ToWorld(r3.Vec{X: interval[0]}), p.V(), p.N()))
		})
	}
}

func (b *builder) driveBand() {
	state := b.a.State
	u, _ := state.Flyer.Sub(state.Wheel).Normalize()
	v := r3.Vec{X: -u.Z, Z: u.X}
	d := state.Flyer.Sub(state.Wheel).Len()
	q := (state.WheelPitch - state.FlyerPitch) / d
	k := math.Sqrt(1 - q*q)
	plus, minus := u.Scale(q).Add(v.Scale(k)), u.Scale(q).Sub(v.Scale(k))
	alpha := math.Acos(q)
	b.part("drive-band-wheel-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.torus(state.WheelPitch, 1, math.Atan2(plus.Z, plus.X), 2*math.Pi-2*alpha, state.Wheel)
	})
	b.part("drive-band-flyer-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.torus(state.FlyerPitch, 1, math.Atan2(minus.Z, minus.X), 2*alpha, state.Flyer)
	})
	for i, n := range []r3.Vec{plus, minus} {
		b.part(fmt.Sprintf("drive-band-span-%d", i+1), "drive-band", "fixed", func() *decad.Body {
			return b.span(state.Wheel.Add(n.Scale(state.WheelPitch)), state.Flyer.Add(n.Scale(state.FlyerPitch)), 1)
		})
	}
}
