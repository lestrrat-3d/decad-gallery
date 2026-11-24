package sevensegment_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentFlagKinematics(t *testing.T) {
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	// The rotor's angle at the unlit and lit dwells from the pin line: the pin sits PinOffset from the axis along s and
	// (lift - Travel/2 - 0.375 + raise) above it, the lift being 0 (unlit) or Travel (lit); the 0.25 is the pin line's
	// modelled position below the middle of the stroke.
	want := map[sevensegment.Segment][2]float64{}
	for _, s := range sevensegment.Segments() {
		raise := sevensegment.RaiseOf(s)
		want[s] = [2]float64{
			math.Atan2(-sevensegment.Travel/2-0.375+raise, sevensegment.PinOffset) * 180 / math.Pi,
			math.Atan2(sevensegment.Travel/2-0.375+raise, sevensegment.PinOffset) * 180 / math.Pi,
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
			for _, dwell := range []struct{ lift, angle float64 }{{0, want[s][0]}, {sevensegment.Travel, want[s][1]}} {
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

// restBands are the rest bands of a class at its dwell offset c (0.15 for a, g, d; 0.15 plus the lowering of the b-f pins
// for b to f), at the slot clearance of the model.
func TestSevenSegmentFlagRestBand(t *testing.T) {
	cAGD, cBF := 0.15, 0.15+sevensegment.RaiseBelowDroop
	um, uw, lw, lm, rmax := bands(sevensegment.PinLinePlay, 0.24, slotClear, cAGD)
	require.InDelta(t, -59.60, um, 0.1)
	require.InDelta(t, -41.76, uw, 0.1)
	require.InDelta(t, 37.85, lw, 0.1)
	require.InDelta(t, 56.63, lm, 0.1)
	require.LessOrEqual(t, rmax, 4.10)
	_, uw, lw, _, rmax = bands(sevensegment.PinLinePlay, 0.30, slotClear, cAGD) // a, g, d worst case
	require.InDelta(t, -41.01, uw, 0.1)
	require.InDelta(t, 37.02, lw, 0.1)
	require.LessOrEqual(t, rmax, 4.15)
	_, uw, lw, lm, _ = bands(sevensegment.PinLinePlay, 0.24, slotClear, cBF) // b to f design
	require.InDelta(t, -42.37, uw, 0.1)
	require.InDelta(t, 37.16, lw, 0.1)
	require.InDelta(t, 56.09, lm, 0.1)
	_, uw, lw, _, rmax = bands(sevensegment.PinLinePlay, 0.42, slotClear, cBF) // b to f worst case
	require.InDelta(t, -40.12, uw, 0.1)
	require.InDelta(t, 34.55, lw, 0.1)
	require.LessOrEqual(t, rmax, sevensegment.SlotEnd+slotClear, "the pin stays inside the slot's end")
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
	}{{"a, g, d", 1.28, cAGD, 0.843, 2.325, 0.25}, {"b to f", 1.88, cBF, 0.832, 1.945, 0.25}} {
		ty, tb, py := sweep(sevensegment.TongueWidth, 0.0, sevensegment.TongueTopGap, headLine, c.cap, c.cap, c.dwell, floor)
		t.Logf("%s turned over (rise %.2f): tongue to the yellow shell %.3f, to the black leg %.3f, pin %.3f", c.name, c.cap, ty, tb, py)
		require.InDelta(t, c.yellow, ty, 0.005, "%s: tongue to the yellow shell", c.name)
		require.InDelta(t, c.black, tb, 0.005, "%s: tongue to the black leg", c.name)
		require.GreaterOrEqual(t, py, c.pinYellowMin, "%s: the pin's gap to the pocket walls", c.name)
	}
	// With the head line 2.0 higher than it is the black leg is entered.
	_, tb, _ := sweep(sevensegment.TongueWidth, 0.0, sevensegment.TongueTopGap, headLine+2.0, 1.28, 1.28, cAGD, floor)
	require.Less(t, tb, 0.0)
}
