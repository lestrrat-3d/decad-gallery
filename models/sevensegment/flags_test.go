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

// TestSevenSegmentFlagRestBand checks the rest bands of a class at its dwell offset c (0.15 plus the 0.05 lowering of the pins,
// for every class). The inner band ends (toward the nominal rests) are set at the cone's least slot clearance, 0.200 at
// the slider's full insertion; the outer ends at its greatest, 0.481 at the full withdrawal.
func TestSevenSegmentFlagRestBand(t *testing.T) {
	c := dwellOffset
	um, uw, lw, lm, rmax := bands(sevensegment.PinLinePlay, 0.24, slotClearMin, c) // design, every class
	require.InDelta(t, -60.37, um, 0.1)
	require.InDelta(t, -42.14, uw, 0.1)
	require.InDelta(t, 36.94, lw, 0.1)
	require.InDelta(t, 56.43, lm, 0.1)
	require.LessOrEqual(t, rmax, 4.10)
	um, _, _, lm, _ = bands(sevensegment.PinLinePlay, 0.24, slotClearMax, c)
	require.InDelta(t, -65.41, um, 0.1)
	require.InDelta(t, 62.05, lm, 0.1)
	// Worst case: a, g, d at z play 0.432 (0.30 and the separate pin's sag tolerance 0.132), b to f at 0.42.
	_, uw, lw, _, rmax = bands(sevensegment.PinLinePlay, zPlayAGD, slotClearMin, c)
	require.InDelta(t, -39.74, uw, 0.1)
	require.InDelta(t, 34.17, lw, 0.1)
	require.LessOrEqual(t, rmax, 4.15)
	_, uw, lw, _, rmax = bands(sevensegment.PinLinePlay, zPlayBF, slotClearMin, c)
	require.InDelta(t, -39.89, uw, 0.1)
	require.InDelta(t, 34.34, lw, 0.1)
	require.LessOrEqual(t, rmax, sevensegment.SlotEnd+slotClearMin, "the cone's axis stays inside the slot's end")
}

// TestSevenSegmentRestBias checks that gravity holds a rotor at its end pose against its journal friction at friction 0.5:
// the bias ratio at the worst-case ends of the rest bands (a, g, d at z play 0.432, b to f at 0.42) stays above 1.50 (A) and
// 1.75 (B): 1.547 and 1.517 for A. The rotor's weight and centre of mass come from the built bodies.
func TestSevenSegmentRestBias(t *testing.T) {
	t.Parallel()
	l := builtLoads(t)
	_, uwA, lwA, _, _ := bands(sevensegment.PinLinePlay, zPlayAGD, slotClearMin, dwellOffset)
	_, uwB, lwB, _, _ := bands(sevensegment.PinLinePlay, zPlayBF, slotClearMin, dwellOffset)
	for _, phi := range []float64{uwA, lwA} {
		require.GreaterOrEqual(t, bias(0.5, l.a, phi), 1.50, "A at %.1f", phi)
	}
	for _, phi := range []float64{uwB, lwB} {
		require.GreaterOrEqual(t, bias(0.5, l.b, phi), 1.75, "B at %.1f", phi)
	}
}

// TestSevenSegmentDriveAudit audits the drive on the built rotors and sliders with the cone on one wall of its slot, at the
// slot's greatest clearance (the cone's full withdrawal, the pocket radius 1.15): at friction 0.5 the drive arm is at least
// 1.50 times the friction arms with the journal settled and 1.35 with it fully across (the computed 1.646 and 1.508). The
// loads are logged.
func TestSevenSegmentDriveAudit(t *testing.T) {
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

// TestSevenSegmentTongueSweep sweeps the tongue and the free span of the drive cone between the tongue and the pocket mouth
// over every pose of the turned-over unit. At the slot's least clearance (0.200) the gaps are the design's values; at its
// greatest (0.481, the cone's full withdrawal, which lets the rotor turn further) the tongue and the free span keep at least
// 0.25 from the yellow shell and the tongue at least 0.7 from the black leg.
func TestSevenSegmentTongueSweep(t *testing.T) {
	headLine := sevensegment.SliderHeadTop - sevensegment.PinAxisZ
	c := dwellOffset
	// The notch floor is measured on the built rotor, and the sweep runs on it: a notch cut shallower than designed shrinks
	// the gaps below.
	floor := sevensegment.MeasureNotchFloor(t, sevensegment.SegG)
	require.InDelta(t, sevensegment.NotchFloor, floor, 0.02, "the built notch floor")
	require.InDelta(t, sevensegment.NotchFloor, sevensegment.MeasureNotchFloor(t, sevensegment.SegB), 0.02, "the built notch floor of type B")
	// The cap is the turned-over rise: 1.808 for a, g, d (1.28 and the separate pin's reversed sag, 0.528) and 1.88 for b to f (the tilt
	// reverses: twice the droop of about 0.30); a, g, d at the worst sag tolerance rise 2.07.
	// The strip's top lies 1.8 below the drive axis; the end disc hulls it. The free span is the cone's base and junction circle.
	for _, c := range []struct {
		name       string
		cap, dwell float64
		yellow     float64 // the least tongue to yellow shell gap at the least slot clearance, +-0.005
		black      float64 // the tongue to the black leg
		span       float64 // the free span to the yellow shell
	}{{"a, g, d", 1.808, c, 0.719, 1.861, 0.690}, {"a, g, d at the worst sag tolerance", 2.07, c, 0.714, 1.720, 0.689},
		{"b to f", 1.88, c, 0.719, 1.794, 0.691}} {
		for _, slot := range []float64{slotClearMin, slotClearMax} {
			ty, tb := sweep(sevensegment.TongueWidth, 0.0, 1.8, headLine, c.cap, c.cap, c.dwell, floor, slot, [][3]float64{endDisc()})
			fy, _ := sweep(0, 0, 0, headLine, c.cap, c.cap, c.dwell, floor, slot, freeSpan())
			t.Logf("%s turned over (rise %.2f), slot clearance %.3f: tongue to the yellow shell %.3f, to the black leg %.3f; free span to the yellow shell %.3f",
				c.name, c.cap, slot, ty, tb, fy)
			require.GreaterOrEqual(t, ty, 0.25, "%s: tongue to the yellow shell", c.name)
			require.GreaterOrEqual(t, tb, 0.7, "%s: tongue to the black leg", c.name)
			require.GreaterOrEqual(t, fy, 0.25, "%s: free span to the yellow shell", c.name)
			if slot != slotClearMin {
				continue
			}
			require.InDelta(t, c.yellow, ty, 0.005, "%s: tongue to the yellow shell", c.name)
			require.InDelta(t, c.black, tb, 0.005, "%s: tongue to the black leg", c.name)
			require.InDelta(t, c.span, fy, 0.005, "%s: free span to the yellow shell", c.name)
		}
	}
	// With the head line 2.0 higher than it is the black leg is entered.
	_, tb := sweep(sevensegment.TongueWidth, 0.0, 1.8, headLine+2.0, 1.808, 1.808, c, floor, slotClearMin, [][3]float64{endDisc()})
	require.Less(t, tb, 0.0)
}
