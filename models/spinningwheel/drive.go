package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
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
		return b.liftRevolve(b.rimLift(), zAxis(0, 0, -10*s), scaled(s,
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
			hole{0, -9, 1.1}, hole{0, 9, 1.1})
		left := b.liftPanel(l, crossAt(0, -20*s), 8*s, -9*s, 46*s, 9*s, 7*s, hole{32 * s, 0, 2.6})
		right := b.liftPanel(l, crossAt(0, 27*s), 8*s, -9*s, 46*s, 9*s, 7*s, hole{32 * s, 0, 2.6})
		// The crank bearing's bore is cut by a tool on the bearing's own wheel lift, 5 past both plate faces.
		y := b.a.State.CrankPin.Y
		return b.cut(b.union(bridge, left, right), b.liftCylinder(b.wheelLift(), wheelAt(25*s, 0, y+2.5+5), 7, 15))
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

// footmanLift carries the footman group in RodFrame: local X runs from the toe up the rod, local Y along the
// cross pin and local Z across the rod. RodHead lies at local (486s, 0, 0).
func (b *builder) footmanLift() lift { return b.liftOf(b.a.State.RodFrame) }

func (b *builder) driveFootman() {
	s := b.p.Scale
	l := b.footmanLift()
	// The tie bolt sits 2 mm inside the rod end at every scale; a fixed 12 mm offset
	// lands on the end face at Scale 1.2 and breaks out of the rod above 1.09.
	boltOffset := 10*s + 2
	head := 486 * s
	b.part("footman", "footman", "footman", func() *decad.Body {
		body := b.liftPanel(l, crossAt(0, 10*s), 20*s, -5*s, 476*s, 5*s, 20*s, hole{26 * s, 0, 2.2})
		// The tie-bolt passage runs 5 past both rod faces it opens.
		return b.cut(body, b.liftCylinder(l, crossAt(head-boltOffset, 10*s+5), 1.1, 20*s+10))
	})
	b.part("footman-head-boss", "footman", "footman", func() *decad.Body {
		// The bearing bore is cut by a tool on the cross bearings' own gimbal lift (frameJoinery), so the bore and
		// the bearings share one exact cylinder.
		return b.liftPanel(l, crossAt(0, 12*s), 460*s, -10*s, 498*s, 10*s, 24*s, hole{head - boltOffset, 0, 1.1})
	})
	// The cross bearings and their spacer are round, so they ride the cross pin's own lift, the gimbal's.
	gl := b.gimbalLift()
	for i, v := range [][2]float64{{-12 * s, -12*s + 5}, {12*s - 5, 12 * s}} {
		b.part(fmt.Sprintf("footman-cross-605-%d", i+1), "bearing", "footman", func() *decad.Body {
			return b.liftAnnulus(gl, crossAt(32*s, v[1]), 2.5, 7, v[1]-v[0])
		})
	}
	b.part("footman-cross-spacer", "spacer", "footman", func() *decad.Body {
		return b.liftAnnulus(gl, crossAt(32*s, 12*s-5), 2.5, 4, (12*s-5)-(-12*s+5))
	})
	for i, v := range [][2]float64{{-12*s - 2, -12 * s}, {12 * s, 12*s + 2}} {
		b.part(fmt.Sprintf("footman-cross-retainer-%d", i+1), "retainer", "footman", func() *decad.Body {
			return b.liftPanel(l, crossAt(head, v[1]), -18*s, -10*s, 12*s, 10*s, v[1]-v[0],
				hole{0, 0, 5}, hole{-boltOffset, 0, 1.1})
		})
	}
	bolt := head - boltOffset
	b.part("footman-retainer-bolt", "fastener", "footman", func() *decad.Body {
		// The bolt turns about local Y, its head under retainer 1.
		return b.liftRevolve(l, frame(r3.Vec{X: bolt, Y: -12*s - 3}, r3.Vec{X: 1}, r3.Vec{Y: 1}), []point2{
			{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 24*s + 7}, {0, 24*s + 7}})
	})
	b.part("footman-retainer-nut", "fastener", "footman", func() *decad.Body {
		return b.liftAnnulus(l, crossAt(bolt, 12*s+4), 1, 2, (12*s+4)-(12*s+2))
	})
	// The upper clamp sits 26s up the rod from the toe, its axis along the cross pin.
	b.driveClamp("footman-upper-clamp", "footman", l, r3.Vec{X: 26 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}, 10*s)
}

// driveClamp builds a cross clamp's headed pin and nut on l. centre is the clamp centre, axis the clamp axis
// and u a direction normal to it, all signed axes in l's local coordinates; face is the distance from the
// centre to each clamped face.
func (b *builder) driveClamp(name, motion string, l lift, centre, u, axis r3.Vec, face float64) {
	b.part(name+"-pin", "fastener", motion, func() *decad.Body {
		return b.liftRevolve(l, frame(centre.Sub(axis.Scale(face+2)), u, axis), []point2{
			{0, 0}, {3.5, 0}, {3.5, 2}, {2, 2}, {2, 2*face + 4.2}, {0, 2*face + 4.2},
		})
	})
	b.part(name+"-nut", "fastener", motion, func() *decad.Body {
		return b.liftAnnulus(l, frame(centre.Add(axis.Scale(face)), u, axis.Cross(u)), 2, 3.5, 2)
	})
}

// treadleLift carries the treadle group in PedalFrame: local X runs along the heel axis (world X), local Y
// along the treadle toward the toe and local Z out of the treadle face.
func (b *builder) treadleLift() lift { return b.liftOf(b.a.State.PedalFrame) }

// heelLift carries the round heel-axis parts. Its axes are signed world axes, local X world Y, local Y world Z
// and local Z world X, so it is exactly orthonormal like the leg-held heel bearings it shares with. The heel
// pin, its collars and its spacers are round, so they take this pose instead of the pedal's; only the key and
// the pin's keyway turn with the treadle. A body at local level x starts at world X x.
func (b *builder) heelLift() lift {
	s := b.p.Scale
	return b.liftOf(frame(r3.Vec{Y: -190 * s, Z: 60 * s}, r3.Vec{Y: 1}, r3.Vec{Z: 1}))
}

// heelAt is the treadle lift's plane normal to the heel axis at local X x.
func heelAt(x, y, z float64) r3.Frame {
	return frame(r3.Vec{X: x, Y: y, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

func (b *builder) driveTreadle() {
	s := b.p.Scale
	p := b.a.State.PedalFrame
	l := b.treadleLift()
	h := b.heelLift()
	b.part("treadle", "treadle", "treadle", func() *decad.Body {
		return b.liftPanel(l, level(0, 0, -12*s), 5*s, -30*s, 115*s, 240*s, 24*s)
	})
	b.part("treadle-heel-rail", "treadle", "treadle", func() *decad.Body {
		section := b.sectionOn(heelAt(-233*s, 0, 0))
		rectangle(section, -12*s, -12*s, 12*s, 12*s)
		keyedOutline(section, 4.1, 1.1, 4.9)
		return b.liftExtrude(section, 1, 426*s, l)
	})
	// The diagonal rail keeps its own frame along (330, 450): its sockets share that frame (diagonalStock).
	b.part("treadle-actuating-rail", "treadle", "treadle", func() *decad.Body {
		length := math.Hypot(330, 450) * s
		u := p.U().Scale(330).Add(p.V().Scale(450)).Scale(s / length)
		v := p.N().Cross(u)
		f := frame(p.ToWorld(r3.Vec{X: -160 * s, Z: -12 * s}), u, v)
		return b.panel(0, -12*s, length, 12*s, 24*s, f)
	})
	b.part("treadle-rear-crossarm", "treadle", "treadle", func() *decad.Body {
		return b.liftPanel(l, level(0, 0, -12*s), 160*s, 438*s, 220*s, 462*s, 12*s)
	})
	b.part("treadle-connector-block", "treadle", "treadle", func() *decad.Body {
		return b.liftPanel(l, heelAt(208*s, 0, 0), 438*s, -12*s, 462*s, 12*s, 24*s, hole{450 * s, 8 * s, 2.2})
	})
	b.part("heel-pin", "shaft", "treadle", func() *decad.Body {
		body := b.liftCylinder(h, level(0, 0, -283*s), 4, 526*s)
		return b.cut(body, b.liftPanel(l, heelAt(-220*s, 0, 0), 2.8, -1, 5, 1, 400*s))
	})
	b.part("heel-key", "key", "treadle", func() *decad.Body {
		return b.liftPanel(l, heelAt(-220*s, 0, 0), 2.8, -1, 4.8, 1, 400*s)
	})
	for i, start := range []float64{-260*s - 6.7, 220*s + 3.7} {
		b.part(fmt.Sprintf("heel-collar-%d", i+1), "collar", "treadle", func() *decad.Body {
			return b.liftAnnulus(h, level(0, 0, start), 4, 6, 3)
		})
	}
	for i, interval := range [][2]float64{
		{-260*s + 3.7, -233*s - 0.2}, {193*s + 0.2, 220*s - 3.7},
	} {
		b.part(fmt.Sprintf("heel-spacer-%d", i+1), "spacer", "treadle", func() *decad.Body {
			return b.liftAnnulus(h, level(0, 0, interval[0]), 4, 5.5, interval[1]-interval[0])
		})
	}
	// The toe lies at local (220s, 450s, 20s); the clamp centre 12s below it, its axis along local X.
	b.driveClamp("treadle-lower-clamp", "treadle", l, r3.Vec{X: 220 * s, Y: 450 * s, Z: 8 * s},
		r3.Vec{Y: 1}, r3.Vec{X: 1}, 12*s)
}

// bandTangents is the drive band's geometry: the unit directions from each pulley centre to the band's two
// tangent points, and the cosine q of the wrap's half gap.
type bandTangents struct {
	plus, minus r3.Vec
	q           float64
}

func (b *builder) bandTangents() bandTangents {
	state := b.a.State
	delta := state.Flyer.Sub(state.Wheel)
	d := delta.Len()
	u := delta.Scale(1 / d)
	v := r3.Vec{X: -u.Z, Z: u.X}
	q := (state.WheelPitch - state.FlyerPitch) / d
	k := math.Sqrt(1 - q*q)
	return bandTangents{plus: u.Scale(q).Add(v.Scale(k)), minus: u.Scale(q).Sub(v.Scale(k)), q: q}
}

// rimLift carries the wheel rim and the band's wheel wrap. It turns the wheel lift's axes about the axle so its
// local +X is the wrap's start ray: the wrap is a revolve from its section plane, and sharing the rim's lift lets
// Verify compare the wrap's disc with the rim's meridian. The rim is round, so its pose does not matter.
func (b *builder) rimLift() lift {
	w := b.a.State.Wheel
	t := b.bandTangents()
	return b.liftOf(shaftFrame(w.X, 0, w.Z, math.Atan2(t.plus.Z, t.plus.X)))
}

// whorlStart is the flyer wrap's start angle about the flyer axis in the untilted head, the angle the head
// tilt turns onto the wrap's world start ray.
func (b *builder) whorlStart() float64 {
	t := b.bandTangents()
	tilt := b.headLift().xf.ApplyDir(r3.Vec{X: 1})
	return math.Atan2(t.minus.Z, t.minus.X) - math.Atan2(tilt.Z, tilt.X)
}

// whorlLift does for the whorl and the band's flyer wrap what rimLift does on the wheel, on the tilted head: its
// local +X, after the head tilt, is the flyer wrap's start ray. Local Z and levels are the flyer lift's.
func (b *builder) whorlLift() lift {
	return b.tilted(b.liftOf(b.headFrame(0, b.whorlStart())))
}

// liftWrap is a band wrap of the given pitch radius and wire radius, turned angle from l's local +X about its
// local Z, in the plane at local level z.
func (b *builder) liftWrap(l lift, z, radius, wire, angle float64) *decad.Body {
	s := b.sectionOn(zAxis(0, 0, z))
	circle(s, radius, 0, wire)
	p := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, p, decad.ConstructionAxis{Dir: r3.Vec{Z: 1}},
		decad.AngleExtent{A: units.Radians(angle), Dir: decad.Along})
	if err != nil {
		b.fail("revolve band wrap", err)
		return nil
	}
	return b.lifted(body, l)
}

func (b *builder) driveBand() {
	state := b.a.State
	t := b.bandTangents()
	plus, minus, q := t.plus, t.minus, t.q
	wheelPlus := state.Wheel.Add(plus.Scale(state.WheelPitch))
	wheelMinus := state.Wheel.Add(minus.Scale(state.WheelPitch))
	flyerPlus := state.Flyer.Add(plus.Scale(state.FlyerPitch))
	flyerMinus := state.Flyer.Add(minus.Scale(state.FlyerPitch))
	b.part("drive-band-wheel-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.liftWrap(b.rimLift(), 0, state.WheelPitch, 1, 2*math.Pi-2*math.Acos(q))
	})
	// The band runs in the world plane Y 0, local level HeadShift of the whorl lift.
	b.part("drive-band-flyer-wrap", "drive-band", "fixed", func() *decad.Body {
		return b.liftWrap(b.whorlLift(), state.HeadShift, state.FlyerPitch, 1, 2*math.Acos(q))
	})
	b.part("drive-band-span-1", "drive-band", "fixed", func() *decad.Body {
		return b.span(wheelPlus, flyerPlus, 1)
	})
	b.part("drive-band-span-2", "drive-band", "fixed", func() *decad.Body {
		return b.span(wheelMinus, flyerMinus, 1)
	})
}
