package sevensegment_test

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/r3"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentFlagKinematics(t *testing.T) {
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	// The rotor's angle at the unlit and lit dwells from the pin line: the pin sits PinOffset from the axis along s and
	// (lift - Travel/2 - 0.25 + raise) above it, the lift being 0 (unlit) or Travel (lit); the 0.25 is the pin line's
	// modelled position below the middle of the stroke.
	want := map[sevensegment.Segment][2]float64{}
	for _, s := range sevensegment.Segments() {
		raise := sevensegment.RaiseOf(s)
		want[s] = [2]float64{
			math.Atan2(-sevensegment.Travel/2-0.25+raise, sevensegment.PinOffset) * 180 / math.Pi,
			math.Atan2(sevensegment.Travel/2-0.25+raise, sevensegment.PinOffset) * 180 / math.Pi,
		}
	}
	for _, s := range sevensegment.Segments() {
		for k := range 10 {
			lit, _ := sevensegment.Lit(k, s)
			w := want[s][0]
			if lit {
				w = want[s][1]
			}
			require.InDelta(t, w, dm.FlagAngle(s, 36*float64(k)), 0.01, "segment %s digit %d", s, k)
		}
		for th := 0.0; th < 360; th += 0.1 {
			lift := dm.Lift(s, th)
			a := dm.FlagAngle(s, th)
			for _, dwell := range []struct{ lift, angle float64 }{{0, want[s][0]}, {3, want[s][1]}} {
				if math.Abs(lift-dwell.lift) <= 0.05 {
					require.InDelta(t, dwell.angle, a, 1.2, "segment %s theta %.1f", s, th)
				}
			}
		}
		for _, r := range dm.Ramps[s] {
			prev := dm.FlagAngle(s, r.Start-90)
			for psi := r.Start; psi <= r.End+1e-9; psi += 0.1 {
				cur := dm.FlagAngle(s, psi-90)
				if r.Rising {
					require.GreaterOrEqual(t, cur, prev-1e-9)
				} else {
					require.LessOrEqual(t, cur, prev+1e-9)
				}
				prev = cur
			}
		}
	}
}

// bands returns the lead-side rest bands of a rotor for slider play s (along the rotor's x) and z, slot clearance cl and
// dwell offset c: the unlit band [lo, uw] and the lit band [lw, hi], and the greatest pin radius.
func bands(s, z, cl, c float64) (um, uw, lw, lm, rmax float64) {
	uw, lw = -99, 99
	for i := 0; i <= 20; i++ {
		si := 1.5 - s + 2*s*float64(i)/20
		for j := 0; j <= 20; j++ {
			z0 := -z + 2*z*float64(j)/20
			for _, lit := range []bool{false, true} {
				zz := -1.5 - c + z0
				if lit {
					zz = 1.5 - c + z0
				}
				rho := math.Hypot(si, zz)
				sl := math.Asin(cl/rho) * 180 / math.Pi
				p := math.Atan2(zz, si) * 180 / math.Pi
				rmax = math.Max(rmax, rho)
				if lit {
					lw, lm = math.Min(lw, p+sl), math.Max(lm, p+sl)
				} else {
					uw, um = math.Max(uw, p-sl), math.Min(um, p-sl)
				}
			}
		}
	}
	return um, uw, lw, lm, rmax
}

// restBands are the rest bands of a class at its dwell offset c (0.15 for a, g, d; 0.15 plus the lowering of the b-f pins
// for b to f), at the slot clearance of the model.
func TestSevenSegmentFlagRestBand(t *testing.T) {
	cAGD, cBF := 0.15, 0.15+sevensegment.RaiseBelowDroop
	um, uw, lw, lm, rmax := bands(sevensegment.PinLinePlay, 0.24, slotClear, cAGD)
	require.InDelta(t, -66.7, um, 0.1)
	require.InDelta(t, -40.33, uw, 0.1)
	require.InDelta(t, 34.47, lw, 0.1)
	require.InDelta(t, 63.12, lm, 0.1)
	require.LessOrEqual(t, rmax, 2.75)
	_, uw, lw, _, rmax = bands(sevensegment.PinLinePlay, 0.30, slotClear, cAGD) // a, g, d worst case
	require.InDelta(t, -39.23, uw, 0.1)
	require.InDelta(t, 33.20, lw, 0.1)
	require.LessOrEqual(t, rmax, 2.80)
	_, uw, lw, lm, _ = bands(sevensegment.PinLinePlay, 0.24, slotClear, cBF) // b to f design
	require.InDelta(t, -41.22, uw, 0.1)
	require.InDelta(t, 33.41, lw, 0.1)
	require.InDelta(t, 62.42, lm, 0.1)
	_, uw, lw, _, rmax = bands(sevensegment.PinLinePlay, 0.42, slotClear, cBF) // b to f worst case
	require.InDelta(t, -37.90, uw, 0.1)
	require.InDelta(t, 29.39, lw, 0.1)
	require.LessOrEqual(t, rmax, sevensegment.SlotEnd+slotClear, "the pin stays inside the slot's end")
}

// rotorLoad is one rotor type's load: weight W_r, centre of mass e toward beta_c, slider pin offset and weight.
type rotorLoad struct{ wr, e, bc, xoff, ws, xcg float64 }

// petg is the weight of PETG per mm3 in N: 1.27 g/cm3.
const petg = 1.27e-3 * 9.81e-3

// flagLoads are the loads of rotor types A and B and the slider weight, taken from the built bodies.
type flagLoads struct {
	a, b rotorLoad
	ws   float64 // the heaviest slider's weight, N
}

var (
	loadsOnce sync.Once
	loads     flagLoads
	loadsErr  error
)

// builtLoads measures the two rotor types (segments g and b) and the heaviest slider on the built bodies: weight from the
// volumes, centre of mass as a distance and an angle from the rotor's x in its section, the pin's station from the
// slider's dimensions, and the slider's centre of mass off its stem.
func builtLoads(t *testing.T) flagLoads {
	t.Helper()
	parts := builtDigit0(t)
	loadsOnce.Do(func() {
		dm := parts.Dimensions
		rotor := func(s sevensegment.Segment, station float64) (rotorLoad, error) {
			vy, err := parts.Flags[s].Yellow.Volume()
			if err != nil {
				return rotorLoad{}, err
			}
			vb, err := parts.Flags[s].Black.Volume()
			if err != nil {
				return rotorLoad{}, err
			}
			cy, err := parts.Flags[s].Yellow.Centroid()
			if err != nil {
				return rotorLoad{}, err
			}
			cb, err := parts.Flags[s].Black.Centroid()
			if err != nil {
				return rotorLoad{}, err
			}
			y, b := vy.Value.Base(), vb.Value.Base()
			c := cy.Value.Scale(y / (y + b)).Add(cb.Value.Scale(b / (y + b)))
			point, axis := segmentAxis(s)
			sAxis := r3.Vec{Z: 1}.Cross(axis)
			d := c.Sub(point)
			ds, dz := d.Dot(sAxis), d.Z
			// The rotor is built turned by its angle at digit 0; the angle in its own section is that of the centre of mass
			// less the turn.
			bc := math.Atan2(dz, ds)*180/math.Pi - dm.FlagAngle(s, 0)
			return rotorLoad{wr: (y + b) * petg, e: math.Hypot(ds, dz), bc: bc, xoff: station}, nil
		}
		slider := func(s sevensegment.Segment) (w, off float64, err error) {
			v, err := parts.Sliders[s].Volume()
			if err != nil {
				return 0, 0, err
			}
			c, err := parts.Sliders[s].Centroid()
			if err != nil {
				return 0, 0, err
			}
			_, cy, horizontal := segmentCentre(s)
			if horizontal {
				return v.Value.Base() * petg, math.Abs(c.Value.Y - cy), nil
			}
			return v.Value.Base() * petg, math.Abs(c.Value.X), nil
		}
		if loads.a, loadsErr = rotor(sevensegment.SegG, sevensegment.PinStationAGD); loadsErr != nil {
			return
		}
		if loads.b, loadsErr = rotor(sevensegment.SegE, sevensegment.PinStationEF); loadsErr != nil {
			return
		}
		for _, s := range sevensegment.Segments() {
			w, off, err := slider(s)
			if err != nil {
				loadsErr = err
				return
			}
			loads.ws = math.Max(loads.ws, w)
			if s == sevensegment.SegG {
				loads.a.ws, loads.a.xcg = w, off
			}
			if s == sevensegment.SegE {
				loads.b.ws, loads.b.xcg = w, off
			}
		}
	})
	require.NoError(t, loadsErr)
	return loads
}

// slotClear is the pin's clearance in its pocket's slot.
const slotClear = sevensegment.SlotRadius - sevensegment.TonguePinR

// journalAcross is the extra side play when the journal sits fully across its hole instead of settled (0.11).
const journalAcross = sevensegment.JournalAcross

// audit is the friction audit of the drive with the pin bearing on one wall of its slot: the worst extra cam force F_x over
// the pin's poses (the side from smin to 3 - smin, the height over the stroke and the play) and the least ratio of drive
// arm to friction arms. The rotor's angle is the pin's polar angle plus or minus asin(cl / rho), the contact normal and the
// imbalance torque follow it, and the friction arm of the slot wall is the pocket radius. across takes the journal fully
// across its hole (a further 0.11 of side play). It returns ok false when the drive locks.
func audit(mu float64, l rotorLoad, across bool) (fx, minRatio float64, ok bool) {
	const length = 25.0
	rj, cl, w := sevensegment.JournalRadius, slotClear, sevensegment.SlotRadius
	smin := sevensegment.PinOffset - sevensegment.PinLinePlay
	if across {
		smin -= journalAcross
	}
	smax := 2*sevensegment.PinOffset - smin
	zlo, zhi := -sevensegment.PinOffset-0.15-0.24, sevensegment.PinOffset-0.15+0.24
	mup := math.Sin(math.Atan(mu))
	minRatio, ok = 99, true
	for i := 0; i <= 40; i++ {
		s := smin + (smax-smin)*float64(i)/40
		for j := 0; j <= 300; j++ {
			z := zlo + (zhi-zlo)*float64(j)/300
			rho := math.Hypot(s, z)
			if rho > sevensegment.SlotEnd+cl || rho <= cl {
				continue
			}
			for _, side := range []float64{-1, 1} {
				phi := math.Atan2(z, s) + side*math.Asin(cl/rho)
				arm := math.Sqrt(rho*rho - cl*cl)
				fr := mu*w + mup*rj*math.Sqrt(1+mu*mu)
				minRatio = math.Min(minRatio, arm/fr)
				k := arm - fr
				if k <= 0 {
					ok = false
					continue
				}
				n := (mup*rj*l.wr + l.wr*l.e*math.Abs(math.Cos(l.bc*math.Pi/180+phi))) / k
				v := n * (math.Abs(math.Cos(phi)) + mu*math.Abs(math.Sin(phi)))
				h := n * (math.Abs(math.Sin(phi)) + mu*math.Abs(math.Cos(phi)))
				f := v*(1+2*mu*l.xoff/length) + mu*h*(1+2*2.0/length) + l.ws*2*mu*l.xcg/length
				fx = math.Max(fx, f)
			}
		}
	}
	return fx, minRatio, ok
}

// bias is the ratio of the gravity torque holding a rotor at its end pose to its journal friction at mu.
func bias(mu float64, l rotorLoad, phi float64) float64 {
	return l.e * math.Abs(math.Cos((l.bc+phi)*math.Pi/180)) / (math.Sin(math.Atan(mu)) * sevensegment.JournalRadius)
}

// TestSevenSegmentRestBias checks that gravity holds a rotor at its end pose against its journal friction at friction 0.5:
// the bias ratio at the worst-case ends of the rest bands (a, g, d at z play 0.30, b to f at 0.42) stays above 1.50 (A) and
// 1.75 (B). The rotor's weight and centre of mass come from the built bodies.
func TestSevenSegmentRestBias(t *testing.T) {
	startDigit0()
	t.Parallel()
	l := builtLoads(t)
	_, uwA, lwA, _, _ := bands(sevensegment.PinLinePlay, 0.30, slotClear, 0.15)
	_, uwB, lwB, _, _ := bands(sevensegment.PinLinePlay, 0.42, slotClear, 0.15+sevensegment.RaiseBelowDroop)
	for _, phi := range []float64{uwA, lwA} {
		require.GreaterOrEqual(t, bias(0.5, l.a, phi), 1.50, "A at %.1f", phi)
	}
	for _, phi := range []float64{uwB, lwB} {
		require.GreaterOrEqual(t, bias(0.5, l.b, phi), 1.75, "B at %.1f", phi)
	}
}

// TestSevenSegmentDriveAudit audits the drive on the built rotors and sliders with the pin on one wall of its slot: at
// friction 0.5 the drive arm is at least 1.50 times the friction arms with the journal settled and 1.35 with it fully
// across (the computed 1.554 and 1.382). The loads are logged.
func TestSevenSegmentDriveAudit(t *testing.T) {
	startDigit0()
	t.Parallel()
	l := builtLoads(t)
	t.Logf("rotor A: weight %.5f N, com %.3f at %.1f deg; rotor B: weight %.5f N, com %.3f at %.1f deg; slider %.5f N",
		l.a.wr, l.a.e, l.a.bc, l.b.wr, l.b.e, l.b.bc, l.ws)
	for _, c := range []struct {
		name string
		l    rotorLoad
	}{{"A", l.a}, {"B", l.b}} {
		for _, across := range []bool{false, true} {
			fx, ratio, ok := audit(0.5, c.l, across)
			t.Logf("rotor %s at friction 0.5 (journal across %v): drive arm / friction arm %.3f, F_x %.4f", c.name, across, ratio, fx)
			require.True(t, ok, "the drive locks")
			want := 1.50
			if across {
				want = 1.35
			}
			require.GreaterOrEqual(t, ratio, want, "rotor %s, journal across %v", c.name, across)
		}
	}
}

// sweep returns the least gaps of the slider's tongue and pin to the rotor's notch walls over every pose, for a tongue
// that spans [-al, ar] about the pin line with its top at -top and a head line at zh below the pin, and slider rises of
// capU (unlit) and capL (lit), the pin line sitting c below its dwell, and the notch floor `floor` from the rotor axis. It
// is the sweep of instructions.md ("Tongue, head and notch sweep"), ported from a one-off script that is not in the
// repository.
func sweep(al, ar, top, zh, capU, capL, c, floor float64) (tongueY, tongueB, pinY float64) {
	// The rotor, pin and slot dimensions come from the model; slot (the slot's clearance to the pin) and rj (the journal's)
	// are the Tight fit, and ss is the slider's play along s less the settled journal (0.11), which the rotor carries along.
	slot, rj := slotClear, sevensegment.JournalHoleR-sevensegment.JournalRadius
	d, s0, rp, hullR := sevensegment.RotorHalfWidth, sevensegment.PinOffset, sevensegment.TonguePinR, sevensegment.TongueHullR
	segEnd, ss := sevensegment.SlotEnd, sevensegment.PinLinePlay-(sevensegment.JournalHoleR-sevensegment.JournalRadius)*0.4472135955
	r2 := 1 / math.Sqrt2
	type pt struct{ x, y float64 }
	var circle []pt
	for a := 0; a < 360; a += 2 {
		r := float64(a) * math.Pi / 180
		circle = append(circle, pt{hullR * math.Cos(r), hullR * math.Sin(r)})
	}
	pts := convexHull(append([]pt{{-al, -top}, {ar, -top}}, circle...))
	for i := 0; i <= 90; i++ {
		for _, x := range []float64{-al, ar} {
			pts = append(pts, pt{x, -top - 9*float64(i)/90})
		}
	}
	for i := 0; i <= 240; i++ {
		pts = append(pts, pt{-6 + 12*float64(i)/240, zh})
	}
	for a := 0; a <= 90; a += 5 {
		r := float64(a) * math.Pi / 180
		pts = append(pts, pt{ar + 0.7 - 0.7*math.Cos(r), zh + 0.67 - 0.7*math.Sin(r)},
			pt{-al - 0.7 + 0.7*math.Cos(r), zh + 0.67 - 0.7*math.Sin(r)})
	}
	var pin []pt
	for a := 0; a < 360; a += 3 {
		r := float64(a) * math.Pi / 180
		pin = append(pin, pt{rp * math.Cos(r), rp * math.Sin(r)})
	}
	type pose struct{ sp, zp, phi float64 }
	var poses []pose
	rmax := segEnd + slot
	for _, c := range []struct{ zn, cap float64 }{{-sevensegment.PinOffset - c - 0.24, capU}, {sevensegment.PinOffset - c - 0.24, capL}} {
		for i := 0; i < 9; i++ {
			sOff := -ss + 2*ss*float64(i)/8
			for k := 0; k < 12; k++ {
				th := 2 * math.Pi * float64(k) / 12
				sp := s0 + sOff - rj*math.Cos(th)
				for j := 0; j < 11; j++ {
					zs := c.cap * float64(j) / 10
					zp := c.zn + zs - rj*math.Sin(th)
					rho := math.Hypot(sp, zp)
					if rho > rmax {
						continue
					}
					b := math.Asin(math.Min(slot/rho, 1))
					for ib := -1; ib <= 1; ib++ {
						poses = append(poses, pose{sp, zp, math.Atan2(zp, sp) + float64(ib)*b})
					}
				}
			}
		}
	}
	for i := 0; i <= 60; i++ {
		zp := -1.5 + 3*float64(i)/60
		rho := math.Hypot(s0, zp)
		b := math.Asin(slot / rho)
		for ib := -1; ib <= 1; ib++ {
			poses = append(poses, pose{s0, zp, math.Atan2(zp, s0) + float64(ib)*b})
		}
	}
	inside := func(x, y float64) bool {
		a, b := (x+y)*r2, (y-x)*r2
		qa, qb := math.Abs(a)-(d-sevensegment.RotorEdgeR), math.Abs(b)-(d-sevensegment.RotorEdgeR)
		return (qa <= 0 && qb <= 0) || math.Hypot(math.Max(qa, 0), math.Max(qb, 0)) <= sevensegment.RotorEdgeR
	}
	gaps := func(shape []pt) (wy, wb float64) {
		wy, wb = 9, 9
		for _, p := range poses {
			c, s := math.Cos(-p.phi), math.Sin(-p.phi)
			for _, q := range shape {
				wx, wz := p.sp+q.x, p.zp+q.y
				x, y := c*wx-s*wz, s*wx+c*wz
				if !inside(x, y) {
					continue
				}
				wy = math.Min(wy, floor-(x+y)*r2)
				wb = math.Min(wb, floor-(y-x)*r2)
			}
		}
		return wy, wb
	}
	tongueY, tongueB = gaps(pts)
	pinY, _ = gaps(pin)
	return tongueY, tongueB, pinY
}

// convexHull returns the convex hull of points by Andrew's monotone chain.
func convexHull[P interface{ ~struct{ x, y float64 } }](pts []P) []P {
	type pt = struct{ x, y float64 }
	q := make([]pt, len(pts))
	for i, p := range pts {
		q[i] = pt(p)
	}
	for i := 1; i < len(q); i++ { // insertion sort by x then y
		for j := i; j > 0 && (q[j].x < q[j-1].x || q[j].x == q[j-1].x && q[j].y < q[j-1].y); j-- {
			q[j], q[j-1] = q[j-1], q[j]
		}
	}
	cross := func(o, a, b pt) float64 { return (a.x-o.x)*(b.y-o.y) - (a.y-o.y)*(b.x-o.x) }
	var lo, up []pt
	for _, p := range q {
		for len(lo) >= 2 && cross(lo[len(lo)-2], lo[len(lo)-1], p) <= 0 {
			lo = lo[:len(lo)-1]
		}
		lo = append(lo, p)
	}
	for i := len(q) - 1; i >= 0; i-- {
		p := q[i]
		for len(up) >= 2 && cross(up[len(up)-2], up[len(up)-1], p) <= 0 {
			up = up[:len(up)-1]
		}
		up = append(up, p)
	}
	h := append(lo[:len(lo)-1], up[:len(up)-1]...)
	out := make([]P, len(h))
	for i, p := range h {
		out[i] = P(p)
	}
	return out
}

func TestSevenSegmentTongueSweep(t *testing.T) {
	headLine := sevensegment.SliderHeadTop - sevensegment.PinAxisZ
	cAGD, cBF := 0.15, 0.15+sevensegment.RaiseBelowDroop
	// The notch floor is measured on the built rotor, and the sweep runs on it: a notch cut shallower than designed shrinks
	// the gaps below.
	floor := sevensegment.MeasureNotchFloor(t, sevensegment.SegG)
	require.InDelta(t, sevensegment.NotchFloor, floor, 0.02, "the built notch floor")
	require.InDelta(t, sevensegment.NotchFloor, sevensegment.MeasureNotchFloor(t, sevensegment.SegB), 0.02, "the built notch floor of type B")
	// The cap is the turned-over rise: 1.28 for a, g, d, and 1.88 for b to f (the tilt reverses: twice the droop of about 0.30).
	for _, c := range []struct {
		name         string
		cap, dwell   float64
		yellow       float64 // the least tongue to yellow shell gap, +-0.005
		black        float64 // the tongue to the black leg
		pinYellowMin float64
	}{{"a, g, d", 1.28, cAGD, 0.164, 0.381, 0.25}, {"b to f", 1.88, cBF, 0.160, 0.303, 0.25}} {
		ty, tb, py := sweep(sevensegment.TongueWidth, 0.0, sevensegment.TongueTopGap, headLine, c.cap, c.cap, c.dwell, floor)
		t.Logf("%s turned over (rise %.2f): tongue to the yellow shell %.3f, to the black leg %.3f, pin %.3f", c.name, c.cap, ty, tb, py)
		require.InDelta(t, c.yellow, ty, 0.005, "%s: tongue to the yellow shell", c.name)
		require.InDelta(t, c.black, tb, 0.005, "%s: tongue to the black leg", c.name)
		require.GreaterOrEqual(t, py, c.pinYellowMin, "%s: the pin's gap to the pocket walls", c.name)
	}
	// With the head line 1.0 lower than it is (the old 73.8 top) the black leg is entered: why the top is 72.8.
	_, tb, _ := sweep(sevensegment.TongueWidth, 0.0, sevensegment.TongueTopGap, headLine+1, 1.28, 1.28, cAGD, floor)
	require.Less(t, tb, 0.0)
}

// TestSevenSegmentFlagBudget is the detent budget of the built model: the detent drive over the opposing torque at every shaft
// angle where a cam is moving a segment, over all ten steps, with the extra cam force of the pin on its slot wall (journal
// across its hole) and the loads of the built bodies. With the cams in phase it is at least 3.0 at friction 0.35, 1.8 at
// 0.45 and 1.0 at 0.5 (computed 3.545, 1.830 and 1.025). With the cams leading the ratchet by the D-bores' play, as they do
// while the pawls drive the shaft back, it is at least 2.0 at 0.35 and 1.0 at 0.45 (computed 2.058 and 1.062).
func TestSevenSegmentFlagBudget(t *testing.T) {
	startDigit0()
	t.Parallel()
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	l := builtLoads(t)
	for _, c := range []struct{ mu, want float64 }{{0.35, 3.0}, {0.45, 1.8}, {0.5, 1.0}} {
		// The budget takes the drive's flag force with the journal across its hole, the larger of the two poses.
		for _, rl := range []rotorLoad{l.a, l.b} {
			across, _, _ := audit(c.mu, rl, true)
			require.GreaterOrEqual(t, budgetFx(t, c.mu), across, "the budget's flag force at friction %.2f", c.mu)
		}
		got := worstBudget(t, dm, c.mu)
		t.Logf("friction %.2f: worst detent budget %.3f", c.mu, got)
		require.GreaterOrEqual(t, got, c.want, "the budget at friction %.2f", c.mu)
	}
	t.Logf("friction 0.40: worst detent budget %.3f", worstBudget(t, dm, 0.40))
	swing := phaseSwing()
	for _, c := range []struct{ mu, want float64 }{{0.35, 2.0}, {0.40, 0}, {0.45, 1.0}, {0.5, 0}} {
		got := worstPhaseBudget(t, dm, c.mu, swing)
		t.Logf("friction %.2f: worst detent budget with the cams %.2f degrees ahead %.3f", c.mu, swing, got)
		require.GreaterOrEqual(t, got, c.want, "the phase budget at friction %.2f", c.mu)
	}
	// Knob torque: the climbing torque of both pawls at the stiff end of PETG plus the opposing torque, at friction 0.5,
	// over the climb to the crest, every step.
	fx := budgetFx(t, 0.5)
	c := pawlCompliance(dm, knobE)
	peak := 0.0
	for psi := 0.0; psi <= dm.CrestAngle; psi += 0.1 {
		climb := detentTorque(dm, c, psi, 0.5, false)
		for k := range 10 {
			load := 2 * 0.1
			if psi >= 7.98 && psi <= 29.55 {
				load, _ = flagLoad(dm, k, psi, 0.5, fx, l.ws)
			}
			peak = math.Max(peak, climb+load)
		}
	}
	t.Logf("peak knob torque at friction 0.5: %.1f N.mm (%.2f N at the knob rim)", peak, peak/sevensegment.KnobRadius)
	require.LessOrEqual(t, peak, 250.0)
}

// worstFlagBudget is the least budget over the ten steps at angle psi at friction 0.35.
func worstFlagBudget(t *testing.T, dm sevensegment.Dimensions, psi float64) float64 {
	t.Helper()
	l := builtLoads(t)
	fx := budgetFx(t, 0.35)
	worst := math.Inf(1)
	for k := range 10 {
		r, _ := flagBudgetRatio(dm, k, psi, 0.35, fx, l.ws)
		worst = math.Min(worst, r)
	}
	return worst
}

// budgetFx is the extra cam force the budget uses at friction mu: the larger of the two rotor types, with the journal fully
// across its hole. The drive's own force pushes the stub across: at the pose of the largest force the pin's push and its
// slot friction lean the journal load 30 degrees or more off vertical, and the stub may sit a further friction angle round
// its hole, so it reaches 0.16 to 0.20 of the 0.20 clearance along the rotor's x, where settled assumes 0.09.
func budgetFx(t *testing.T, mu float64) float64 {
	t.Helper()
	l := builtLoads(t)
	fx := 0.0
	for _, rl := range []rotorLoad{l.a, l.b} {
		f, _, ok := audit(mu, rl, true)
		require.True(t, ok, "the drive locks at friction %.3f", mu)
		fx = math.Max(fx, f)
	}
	return fx
}

// worstPhaseBudget is the least release budget over the ten steps at friction mu with each cam up to swing degrees ahead
// of the ratchet wheel: for every cam angle c in [7.98, 29.55] the detent torque at every ratchet angle in [c - swing, c]
// over the opposing torque at c.
func worstPhaseBudget(t *testing.T, dm sevensegment.Dimensions, mu, swing float64) float64 {
	t.Helper()
	l := builtLoads(t)
	fx := budgetFx(t, mu)
	c := pawlCompliance(dm, pawlE)
	const step = 0.05
	td := make([]float64, int(30/step)+1)
	for i := range td {
		td[i] = detentTorque(dm, c, float64(i)*step, mu, true)
	}
	worst := math.Inf(1)
	for cam := 7.98; cam <= 29.55+1e-9; cam += step {
		load := 0.0
		for k := range 10 {
			ld, _ := flagLoad(dm, k, cam, mu, fx, l.ws)
			load = math.Max(load, ld)
		}
		for r := math.Max(0, cam-swing); r <= cam+1e-9; r += step {
			worst = math.Min(worst, td[int(math.Round(r/step))]/load)
		}
	}
	return worst
}
