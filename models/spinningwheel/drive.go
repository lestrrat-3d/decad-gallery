package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) drive() {
	b.driveWheel()
	b.driveGimbal()
	b.driveFootman()
	b.driveTreadle()
	b.part("treadle-flex-connector", "compliant", "compliant", b.flexConnector)
	b.driveBand()
}

// wheelLift carries the wheel group at the wheel's own angle. Local Z runs along the axle toward -Y, so a
// body whose extrusion starts at world Y y starts at local level -y; local X runs from the axle to the crank
// pin, which sits at local (25s, 0). The axle, its rings and the fixed wheel bearings are round, so they take
// this pose too and every mating face of the wheel axis shares one exact placement.
func (b *builder) wheelLift() lift {
	w := b.a.State.Wheel
	return b.liftOf(shaftFrame(w.X, 0, w.Z, b.p.WheelAngle))
}

// wheelAt is the wheel lift's plane through local (x, y) whose extrusion starts at world Y y0 toward -Y.
func wheelAt(x, y, y0 float64) r3.Frame { return level(x, y, -y0) }

func (b *builder) driveWheel() {
	s := b.p.Scale
	a := math.Remainder(b.p.WheelAngle, 2*math.Pi)
	w := b.a.State.Wheel
	l := b.wheelLift()
	at := func(y float64) r3.Frame { return wheelAt(0, 0, y) }
	b.part("wheel-rim", "wheel", "wheel", func() *decad.Body {
		// The section's height h lies at world Y 10s-h.
		return b.liftRevolve(l, zAxis(0, 0, -10*s), scaled(s,
			point2{255, 0}, point2{280, 0}, point2{280, 7}, point2{274, 10},
			point2{280, 13}, point2{280, 20}, point2{255, 20}))
	})
	b.part("wheel-hub", "wheel", "wheel", func() *decad.Body {
		return b.keyedAnnulus(l, at(15*s), 4.1, 1.1, 4.9, 30*s, 30*s)
	})
	// Each spoke shares its exact frame with its hub and rim socket tools (frameJoinery).
	for i := range 6 {
		b.part(fmt.Sprintf("wheel-spoke-%d", i+1), "wheel", "wheel", func() *decad.Body {
			return b.panel(25*s, -9*s, 260*s, 9*s, 12*s,
				shaftFrame(w.X, 6*s, w.Z, a+float64(i)*math.Pi/3))
		})
	}
	b.part("wheel-axle", "shaft", "wheel", func() *decad.Body {
		body := b.liftCylinder(l, at(86*s+3.2), 4, 146*s+3.2)
		hubPocket := b.liftPanel(l, at(12*s), 2.8, -1, 5, 1, 24*s)
		crankPocket := b.liftPanel(l, at(85*s), 2.8, -1, 5, 1, 6*s)
		body = b.cut(body, hubPocket)
		return b.cut(body, crankPocket)
	})
	for _, key := range []struct {
		name   string
		y0, y1 float64
	}{{"wheel-hub-key", -12 * s, 12 * s}, {"wheel-crank-key", 79 * s, 85 * s}} {
		b.part(key.name, "key", "wheel", func() *decad.Body {
			return b.liftPanel(l, at(key.y1), 2.8, -1, 4.8, 1, key.y1-key.y0)
		})
	}
	b.part("wheel-crank", "crank", "wheel", func() *decad.Body {
		section := b.sectionOn(at(86 * s))
		rectangle(section, -9*s, -9*s, 34*s, 9*s)
		keyedOutline(section, 4.1, 1.1, 4.9)
		circle(section, 25*s, 0, 2.6)
		return b.liftExtrude(section, 2, 8*s, l)
	})
	for _, ring := range []struct {
		name, group          string
		inner, outer, y0, y1 float64
	}{
		{"wheel-crank-front-collar", "collar", 4, 6, 78*s - 3.2, 78*s - 0.2},
		{"wheel-crank-rear-collar", "collar", 4, 6, 86*s + 0.2, 86*s + 3.2},
		{"wheel-hub-front-spacer", "spacer", 4, 6, -45*s + 7, -15 * s},
		{"wheel-hub-rear-spacer", "spacer", 4, 6, 15 * s, 45*s - 7},
	} {
		b.part(ring.name, ring.group, "wheel", func() *decad.Body {
			return b.liftAnnulus(l, at(ring.y1), ring.inner, ring.outer, ring.y1-ring.y0)
		})
	}
	cf := func(y float64) r3.Frame { return wheelAt(25*s, 0, y) }
	b.part("wheel-crank-pin", "pin", "wheel", func() *decad.Body {
		return b.liftCylinder(l, cf(112*s+3.5), 2.5, 34*s+7)
	})
	for _, ring := range []struct {
		name, group   string
		outer, y0, y1 float64
	}{
		{"wheel-crank-pin-collar-1", "collar", 4.5, 78*s - 3.2, 78*s - 0.2},
		{"wheel-crank-pin-collar-2", "collar", 4.5, 105*s + 2.7, 105*s + 5.7},
		{"wheel-crank-pin-spacer", "spacer", 4, 86 * s, 105*s - 2.5},
	} {
		b.part(ring.name, ring.group, "wheel", func() *decad.Body {
			return b.liftAnnulus(l, cf(ring.y1), 2.5, ring.outer, ring.y1-ring.y0)
		})
	}
}

// gimbalLift carries the gimbal group in GimbalFrame: local X runs radially from the crank pin to the cross
// pin, local Y along the cross pin and local Z along the crank pin toward -Y.
func (b *builder) gimbalLift() lift { return b.liftOf(b.a.State.GimbalFrame) }

// crossAt is the gimbal lift's plane through local (x, v, 0), normal to the cross pin, whose extrusion runs
// toward local -Y from v. The bridge cheeks and every ring on the cross pin share its axes, so their flush
// faces differ only by exact plane offsets.
func crossAt(x, v float64) r3.Frame { return frame(r3.Vec{X: x, Y: v}, r3.Vec{X: 1}, r3.Vec{Z: 1}) }

func (b *builder) driveGimbal() {
	s := b.p.Scale
	l := b.gimbalLift()
	b.part("gimbal-bridge", "gimbal", "gimbal", func() *decad.Body {
		bridge := b.liftPanel(l, level(0, 0, -2.5), -11*s, -22*s, 10*s, 22*s, 5,
			hole{0, 0, 7}, hole{0, -9, 1.1}, hole{0, 9, 1.1})
		left := b.liftPanel(l, crossAt(0, -20*s), 8*s, -9*s, 46*s, 9*s, 7*s, hole{32 * s, 0, 2.6})
		right := b.liftPanel(l, crossAt(0, 27*s), 8*s, -9*s, 46*s, 9*s, 7*s, hole{32 * s, 0, 2.6})
		return b.union(bridge, left, right)
	})
	// The crank bearing is round, so it rides its journal's lift: the crank pin's, local (25s, 0) of the wheel.
	b.part("gimbal-crank-605", "bearing", "gimbal", func() *decad.Body {
		y := b.a.State.CrankPin.Y
		return b.liftAnnulus(b.wheelLift(), wheelAt(25*s, 0, y+2.5), 2.5, 7, (y+2.5)-(y-2.5))
	})
	for i, start := range []float64{-4.5, 2.5} {
		b.part(fmt.Sprintf("gimbal-crank-retainer-%d", i+1), "retainer", "gimbal", func() *decad.Body {
			section := b.sectionOn(level(0, 0, start))
			circle(section, 0, 0, 11)
			circle(section, 0, 0, 5)
			circle(section, 0, -9, 1.1)
			circle(section, 0, 9, 1.1)
			return b.liftExtrude(section, 3, 2, l)
		})
	}
	for i, v := range []float64{-9, 9} {
		b.part(fmt.Sprintf("gimbal-retainer-bolt-%d", i+1), "fastener", "gimbal", func() *decad.Body {
			return b.liftRevolve(l, zAxis(0, v, -5.5), []point2{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 12}, {0, 12}})
		})
		b.part(fmt.Sprintf("gimbal-retainer-nut-%d", i+1), "fastener", "gimbal", func() *decad.Body {
			return b.liftAnnulus(l, level(0, v, 4.5), 1, 2, 2)
		})
	}
	// The cross pin's axis is local Y through local (32s, 0, 0), RodHead.
	x := 32 * s
	b.part("gimbal-cross-pin", "pin", "gimbal", func() *decad.Body {
		return b.liftCylinder(l, crossAt(x, 30*s+3), 2.5, 60*s+6)
	})
	for i, v := range [][2]float64{{-27*s - 3.2, -27*s - 0.2}, {27*s + 0.2, 27*s + 3.2}} {
		b.part(fmt.Sprintf("gimbal-cross-pin-collar-%d", i+1), "collar", "gimbal", func() *decad.Body {
			return b.liftAnnulus(l, crossAt(x, v[1]), 2.5, 4.5, v[1]-v[0])
		})
	}
	for i, v := range [][2]float64{{-20 * s, -12 * s}, {12 * s, 20 * s}} {
		b.part(fmt.Sprintf("gimbal-cross-spacer-%d", i+1), "spacer", "gimbal", func() *decad.Body {
			return b.liftAnnulus(l, crossAt(x, v[1]), 2.5, 4, v[1]-v[0])
		})
	}
}

func (b *builder) driveFootman() {
	s := b.p.Scale
	r := b.a.State.RodFrame
	point := r.ToWorld
	// The tie bolt sits 2 mm inside the rod end at every scale; a fixed 12 mm offset
	// lands on the end face at Scale 1.2 and breaks out of the rod above 1.09.
	boltOffset := 10*s + 2
	sectionFrame := func(v float64) r3.Frame {
		return frame(point(r3.Vec{Y: v}), r.U(), r.N())
	}
	b.part("footman", "footman", "footman", func() *decad.Body {
		body := b.panel(20*s, -5*s, 476*s, 5*s, 20*s, sectionFrame(10*s), hole{26 * s, 0, 2.2})
		return b.cut(body, b.cylinder(1.1, 20*s,
			frame(point(r3.Vec{X: 486*s - boltOffset, Y: 10 * s}), r.U(), r.N())))
	})
	b.part("footman-head-boss", "footman", "footman", func() *decad.Body {
		return b.panel(460*s, -10*s, 498*s, 10*s, 24*s, sectionFrame(12*s),
			hole{486 * s, 0, 7}, hole{486*s - boltOffset, 0, 1.1})
	})
	centre := b.a.State.RodHead
	axial := func(v float64) r3.Frame { return axisFrame(centre.Add(r.V().Scale(v)), r.V()) }
	for i, start := range []float64{-12 * s, 12*s - 5} {
		b.part(fmt.Sprintf("footman-cross-605-%d", i+1), "bearing", "footman", func() *decad.Body {
			return b.annulus(2.5, 7, 5, axial(start))
		})
	}
	b.part("footman-cross-spacer", "spacer", "footman", func() *decad.Body {
		return b.annulus(2.5, 4, 24*s-10, axial(-12*s+5))
	})
	for i, start := range []float64{-12 * s, 12*s + 2} {
		b.part(fmt.Sprintf("footman-cross-retainer-%d", i+1), "retainer", "footman", func() *decad.Body {
			f := frame(centre.Add(r.V().Scale(start)), r.U(), r.N())
			return b.panel(-18*s, -10*s, 12*s, 10*s, 2, f, hole{0, 0, 5}, hole{-boltOffset, 0, 1.1})
		})
	}
	boltCentre := centre.Sub(r.U().Scale(boltOffset))
	b.part("footman-retainer-bolt", "fastener", "footman", func() *decad.Body {
		return b.revolve([]point2{{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 24*s + 7}, {0, 24*s + 7}},
			axisFrame(boltCentre.Sub(r.V().Scale(12*s+3)), r.V()))
	})
	b.part("footman-retainer-nut", "fastener", "footman", func() *decad.Body {
		return b.annulus(1, 2, 2, axisFrame(boltCentre.Add(r.V().Scale(12*s+2)), r.V()))
	})
	b.driveClamp("footman-upper-clamp", "footman", b.a.State.Toe.Add(r.U().Scale(26*s)), r.V(), 10*s)
}

func (b *builder) driveClamp(name, motion string, centre, axis r3.Vec, face float64) {
	b.part(name+"-pin", "fastener", motion, func() *decad.Body {
		return b.revolve([]point2{
			{0, 0}, {3.5, 0}, {3.5, 2}, {2, 2}, {2, 2*face + 4.2}, {0, 2*face + 4.2},
		}, axisFrame(centre.Sub(axis.Scale(face+2)), axis))
	})
	b.part(name+"-nut", "fastener", motion, func() *decad.Body {
		return b.annulus(2, 3.5, 2, axisFrame(centre.Add(axis.Scale(face)), axis))
	})
}

func (b *builder) driveTreadle() {
	s := b.p.Scale
	p := b.a.State.PedalFrame
	point := p.ToWorld
	xy := func(z float64) r3.Frame { return frame(point(r3.Vec{Z: z}), p.U(), p.V()) }
	yz := func(x float64) r3.Frame { return frame(point(r3.Vec{X: x}), p.V(), p.N()) }
	b.part("treadle", "treadle", "treadle", func() *decad.Body {
		return b.panel(5*s, -30*s, 115*s, 240*s, 24*s, xy(-12*s))
	})
	b.part("treadle-heel-rail", "treadle", "treadle", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -12*s, -12*s, 12*s, 12*s)
		keyedOutline(section, 4.1, 1.1, 4.9)
		return b.extrude(section, 1, 426*s, yz(-233*s))
	})
	b.part("treadle-actuating-rail", "treadle", "treadle", func() *decad.Body {
		length := math.Hypot(330, 450) * s
		u := p.U().Scale(330).Add(p.V().Scale(450)).Scale(s / length)
		v := p.N().Cross(u)
		f := frame(point(r3.Vec{X: -160 * s, Z: -12 * s}), u, v)
		return b.panel(0, -12*s, length, 12*s, 24*s, f)
	})
	b.part("treadle-rear-crossarm", "treadle", "treadle", func() *decad.Body {
		return b.panel(160*s, 438*s, 220*s, 462*s, 12*s, xy(-12*s))
	})
	b.part("treadle-connector-block", "treadle", "treadle", func() *decad.Body {
		return b.panel(438*s, -12*s, 462*s, 12*s, 24*s, yz(208*s), hole{450 * s, 8 * s, 2.2})
	})
	b.part("heel-pin", "shaft", "treadle", func() *decad.Body {
		body := b.cylinder(4, 526*s, yz(-283*s))
		return b.cut(body, b.panel(2.8, -1, 5, 1, 400*s, yz(-220*s)))
	})
	b.part("heel-key", "key", "treadle", func() *decad.Body {
		return b.panel(2.8, -1, 4.8, 1, 400*s, yz(-220*s))
	})
	for i, start := range []float64{-260*s - 6.7, 220*s + 3.7} {
		b.part(fmt.Sprintf("heel-collar-%d", i+1), "collar", "treadle", func() *decad.Body {
			return b.annulus(4, 6, 3, yz(start))
		})
	}
	for i, interval := range [][2]float64{
		{-260*s + 3.7, -233*s - 0.2}, {193*s + 0.2, 220*s - 3.7},
	} {
		b.part(fmt.Sprintf("heel-spacer-%d", i+1), "spacer", "treadle", func() *decad.Body {
			return b.annulus(4, 5.5, interval[1]-interval[0], yz(interval[0]))
		})
	}
	b.driveClamp("treadle-lower-clamp", "treadle", b.a.State.Toe.Sub(p.N().Scale(12*s)), p.U(), 12*s)
}

func (b *builder) driveBand() {
	state := b.a.State
	delta := state.Flyer.Sub(state.Wheel)
	d := delta.Len()
	u := delta.Scale(1 / d)
	v := r3.Vec{X: -u.Z, Z: u.X}
	q := (state.WheelPitch - state.FlyerPitch) / d
	k := math.Sqrt(1 - q*q)
	plus := u.Scale(q).Add(v.Scale(k))
	minus := u.Scale(q).Sub(v.Scale(k))
	wheelPlus := state.Wheel.Add(plus.Scale(state.WheelPitch))
	wheelMinus := state.Wheel.Add(minus.Scale(state.WheelPitch))
	flyerPlus := state.Flyer.Add(plus.Scale(state.FlyerPitch))
	flyerMinus := state.Flyer.Add(minus.Scale(state.FlyerPitch))
	b.part("drive-band-wheel-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.torus(state.WheelPitch, 1, math.Atan2(plus.Z, plus.X),
			2*math.Pi-2*math.Acos(q), state.Wheel)
	})
	b.part("drive-band-flyer-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.torus(state.FlyerPitch, 1, math.Atan2(minus.Z, minus.X),
			2*math.Acos(q), state.Flyer)
	})
	b.part("drive-band-span-1", "drive-band", "fixed", func() *decad.Body {
		return b.span(wheelPlus, flyerPlus, 1)
	})
	b.part("drive-band-span-2", "drive-band", "fixed", func() *decad.Body {
		return b.span(wheelMinus, flyerMinus, 1)
	})
}
