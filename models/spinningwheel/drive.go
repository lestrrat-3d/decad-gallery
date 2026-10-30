package spinningwheel

import (
	"fmt"
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"math"
)

func (b *builder) drive() {
	s := b.p.Scale
	st := b.a.State
	a := math.Remainder(b.p.WheelAngle, 2*math.Pi)
	wf := func(y, t float64) r3.Frame { return shaftFrame(st.Wheel.X, y, st.Wheel.Z, t) }
	b.part("wheel-rim", "wheel", "wheel", func() *decad.Body {
		return b.revolve(scaled(s, point2{255, 0}, point2{280, 0}, point2{280, 7}, point2{274, 10}, point2{280, 13}, point2{280, 20}, point2{255, 20}), wf(10*s, a))
	})
	b.part("wheel-hub", "wheel", "wheel", func() *decad.Body { return b.dAnnulus(4.1, 3.1, 30*s, 30*s, wf(15*s, a)) })
	for i := range 8 {
		angle := a + float64(i)*math.Pi/4
		b.part(fmt.Sprintf("wheel-spoke-%d", i+1), "wheel", "wheel", func() *decad.Body {
			return b.panel(25*s, -9*s, 260*s, 9*s, 12*s, wf(6*s, angle))
		})
	}
	b.part("wheel-axle", "shaft", "wheel", func() *decad.Body {
		body := b.cylinder(4, 138*s, wf(60*s, a))
		c1 := b.panel(3, -5, 5, 5, 36*s, wf(18*s, a))
		c2 := b.panel(3, -5, 5, 5, 12*s, wf(-66*s, a))
		return b.flatShaft(body, c1, c2)
	})
	b.part("wheel-crank", "crank", "wheel", func() *decad.Body {
		body := b.panel(-9*s, -9*s, 34*s, 9*s, 8*s, wf(-68*s, a), hole{25 * s, 0, 2.6})
		return b.cut(body, b.dTool(4.1, 3.1, 8*s, wf(-68*s, a)))
	})
	b.part("crank-pin", "pin", "wheel", func() *decad.Body {
		return b.cylinder(2.5, 41*s+7, shaftFrame(st.CrankPin.X, -68*s+3.5, st.CrankPin.Z, 0))
	})
	b.part("footman", "footman", "footman", func() *decad.Body {
		return b.panel(-12*s, -12*s, 382*s, 12*s, 8*s, shaftFrame(st.Toe.X, -101*s, st.Toe.Z, st.RodAngle), hole{0, 0, 7}, hole{370 * s, 0, 7})
	})
	for i, c := range []r3.Vec{st.Toe, st.CrankPin} {
		b.part(fmt.Sprintf("footman-bearing-%d", i+1), "bearing", "footman", func() *decad.Body {
			return b.annulus(2.5, 7, 5, shaftFrame(c.X, -105*s+2.5, c.Z, st.RodAngle))
		})
	}
	b.part("treadle", "treadle", "treadle", func() *decad.Body {
		sk := b.section(false)
		if b.err != nil {
			return nil
		}
		polygon(sk, scaled(s, point2{-18, -18}, point2{18, -18}, point2{18, -12}, point2{212, -12}, point2{212, 12}, point2{18, 12}, point2{18, 18}, point2{-18, 18}))
		circle(sk, 0, 0, 11)
		circle(sk, 200*s, 0, 2.6)
		return b.extrude(sk, 2, 80*s, shaftFrame(st.Heel.X, -150*s, st.Heel.Z, st.PedalAngle))
	})
	b.part("toe-pin", "pin", "treadle", func() *decad.Body { return b.cylinder(2.5, 129*s+7, shaftFrame(st.Toe.X, -101*s+3.5, st.Toe.Z, 0)) })
	b.part("heel-pin", "pin", "fixed", func() *decad.Body { return b.cylinder(4, 125*s, shaftFrame(st.Heel.X, -128*s, st.Heel.Z, 0)) })
	for i, y := range []float64{-228*s + 7, -152 * s} {
		b.part(fmt.Sprintf("heel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.annulus(4, 11, 7, shaftFrame(st.Heel.X, y, st.Heel.Z, 0))
		})
	}
	for i, ys := range [][2]float64{{-246, -236}, {-144, -134}} {
		b.part(fmt.Sprintf("heel-bracket-%d", i+1), "frame", "fixed", func() *decad.Body {
			return b.panel(60*s, 30*s, 100*s, 84*s, (ys[1]-ys[0])*s, shaftFrame(0, ys[1]*s, 0, 0), hole{st.Heel.X, st.Heel.Z, 4.1})
		})
	}
	for i, lim := range [][2]float64{{-236 * s, -228 * s}, {-152 * s, -144 * s}, {-228*s + 7, -152*s - 7}} {
		b.part(fmt.Sprintf("heel-spacer-%d", i+1), "spacer", "fixed", func() *decad.Body {
			return b.annulus(4, 5.5, lim[1]-lim[0], shaftFrame(st.Heel.X, lim[1], st.Heel.Z, 0))
		})
	}
	for i, v := range []struct {
		c           r3.Vec
		y, r, outer float64
		motion      string
	}{
		{st.CrankPin, -109*s - .2, 2.5, 4.5, "wheel"}, {st.CrankPin, -68*s + .2 + 3, 2.5, 4.5, "wheel"},
		{st.Toe, -230*s - .2, 2.5, 4.5, "treadle"}, {st.Toe, -101*s + .2 + 3, 2.5, 4.5, "treadle"},
		{st.Heel, -246 * s, 4, 6, "fixed"}, {st.Heel, -134*s + 3, 4, 6, "fixed"},
	} {
		b.part(fmt.Sprintf("linkage-collar-%d", i+1), "collar", v.motion, func() *decad.Body { return b.annulus(v.r, v.outer, 3, shaftFrame(v.c.X, v.y, v.c.Z, 0)) })
	}
	for i, v := range []struct {
		c      r3.Vec
		lo, hi float64
		motion string
	}{
		{st.CrankPin, -109*s - .2, -105*s - 2.7, "wheel"}, {st.CrankPin, -105*s + 2.7, -76*s - .2, "wheel"},
		{st.Toe, -150*s + .2, -105*s - 2.7, "treadle"}, {st.Toe, -105*s + 2.7, -101*s - .2, "treadle"},
	} {
		b.part(fmt.Sprintf("linkage-sleeve-%d", i+1), "spacer", v.motion, func() *decad.Body { return b.annulus(2.5, 4, v.hi-v.lo, shaftFrame(v.c.X, v.hi, v.c.Z, 0)) })
	}
	b.belt()
}

func (b *builder) belt() {
	st := b.a.State
	u, _ := st.Flyer.Sub(st.Wheel).Normalize()
	v := r3.Vec{X: -u.Z, Z: u.X}
	d := st.Flyer.Sub(st.Wheel).Len()
	q := (st.WheelPitch - st.FlyerPitch) / d
	k := math.Sqrt(1 - q*q)
	plus, minus := u.Scale(q).Add(v.Scale(k)), u.Scale(q).Sub(v.Scale(k))
	alpha := math.Acos(q)
	awp, awm := st.Wheel.Add(plus.Scale(st.WheelPitch)), st.Wheel.Add(minus.Scale(st.WheelPitch))
	afp, afm := st.Flyer.Add(plus.Scale(st.FlyerPitch)), st.Flyer.Add(minus.Scale(st.FlyerPitch))
	b.part("drive-band-wheel-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.torus(st.WheelPitch, 1, math.Atan2(plus.Z, plus.X), 2*math.Pi-2*alpha, st.Wheel)
	})
	b.part("drive-band-flyer-wrap", "drive-band", "fixed", func() *decad.Body { return b.torus(st.FlyerPitch, 1, math.Atan2(minus.Z, minus.X), 2*alpha, st.Flyer) })
	b.part("drive-band-lower-span", "drive-band", "fixed", func() *decad.Body { return b.span(awm, afm, 1) })
	b.part("drive-band-upper-span", "drive-band", "fixed", func() *decad.Body { return b.span(afp, awp, 1) })
}
