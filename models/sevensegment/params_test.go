package sevensegment_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentTruthTable(t *testing.T) {
	// a b c d e f g per digit, with the a-d, b-e, c-f roles swapped for the knob-up stance (instructions.md "Truth table and ramp table").
	want := [10]string{
		"1111110", "0000110", "1101101", "1001111", "0010111", "1011011", "1111011", "0001110", "1111111", "1011111",
	}
	for digit, row := range want {
		for i, s := range sevensegment.Segments() {
			lit, err := sevensegment.Lit(digit, s)
			require.NoError(t, err)
			require.Equal(t, row[i] == '1', lit, "digit %d segment %s", digit, s)
		}
	}
	_, err := sevensegment.Lit(10, sevensegment.SegA)
	require.Error(t, err)
	require.Equal(t, "a", sevensegment.SegA.String())
	require.Equal(t, "g", sevensegment.SegG.String())
}

func TestSevenSegmentValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    sevensegment.Params
		want string
	}{
		{"low", sevensegment.Params{Digit: -1}, "sevensegment: digit -1 out of range 0..9"},
		{"high", sevensegment.Params{Digit: 10}, "sevensegment: digit 10 out of range 0..9"},
		{"placement", sevensegment.Params{Digit: 3, Placement: &r3.Frame{}}, "sevensegment: invalid placement frame"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.p.Validate()
			require.EqualError(t, err, tc.want)
			_, err = tc.p.Derive()
			require.EqualError(t, err, tc.want)
		})
	}
	require.NoError(t, sevensegment.DefaultParams().Validate())
}

func TestSevenSegmentCamProfile(t *testing.T) {
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	maxPressure := 0.0
	activeLo, activeHi := 1e9, -1e9
	for _, s := range sevensegment.Segments() {
		// Each cam's ramp list matches the table.
		want := rampsPerCam[s]
		require.Len(t, dm.Ramps[s], len(want), "segment %s", s)
		for i, r := range dm.Ramps[s] {
			require.Equal(t, want[i].from, r.From, "segment %s ramp %d", s, i)
			require.Equal(t, want[i].rising, r.Rising, "segment %s ramp %d", s, i)
			require.InDelta(t, 90+36*float64(r.From)+8, r.Start, 1e-9)
			require.InDelta(t, r.Start+21.5, r.End, 1e-9)
		}
		for k := range 10 {
			lit, _ := sevensegment.Lit(k, s)
			level := 0.0
			if lit {
				level = sevensegment.Travel
			}
			for th := 36*float64(k) - 6.5; th <= 36*float64(k)+8.0+1e-9; th += 0.1 {
				require.InDelta(t, level, dm.Lift(s, th), 1e-9, "segment %s digit %d theta %.1f", s, k, th)
			}
		}
		// Ramps are monotonic and report the active zone and pressure angle.
		for _, r := range dm.Ramps[s] {
			prev := dm.GrooveRadius(s, r.Start)
			for psi := r.Start; psi <= r.End+1e-9; psi += 0.1 {
				cur := dm.GrooveRadius(s, psi)
				if r.Rising {
					require.GreaterOrEqual(t, cur, prev-1e-9)
				} else {
					require.LessOrEqual(t, cur, prev+1e-9)
				}
				prev = cur
				maxPressure = math.Max(maxPressure, dm.PressureAngle(s, psi))
				base := r.Start - 8 - 90
				lift := cur - sevensegment.OffRadius
				dwell := 0.0
				if !r.Rising {
					dwell = sevensegment.Travel
				}
				if math.Abs(lift-dwell) > 0.05 && math.Abs(lift-(sevensegment.Travel-dwell)) > 0.05 {
					off := math.Mod(psi-(90+base)+720, 36) // angle past psi_k
					activeLo, activeHi = math.Min(activeLo, off), math.Max(activeHi, off)
				}
			}
		}
	}
	require.LessOrEqual(t, maxPressure, 35.5)
	require.GreaterOrEqual(t, activeLo, 9.5)
	require.LessOrEqual(t, activeHi, 28.0)
	// Rising lift samples, "Truth table and ramp table".
	samples := map[float64]float64{8: 0, 9: 0.019, 10: 0.078, 12: 0.297, 14: 0.555, 16: 0.842, 18: 1.161, 20: 1.514,
		22: 1.904, 24: 2.335, 26: 2.743, 28: 2.954, 29: 2.995, 29.5: 3.0}
	for off, lift := range samples {
		rise := dm.GrooveRadius(sevensegment.SegE, 90+36*6+off) - sevensegment.OffRadius // segment e, step 6->7 rises
		require.InDelta(t, lift, rise, 0.005, "offset %g", off)
	}
}

func TestSevenSegmentRampGeometry(t *testing.T) {
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	// Step 0->1 falls for cam a; step 1->2 rises (window starts at 134).
	rise := dm.Ramps[sevensegment.SegD][1].Geometry()
	require.InDelta(t, (sevensegment.OffRadius+sevensegment.RampRound)*math.Cos(134*math.Pi/180), rise.C1.X, 1e-9)
	require.InDelta(t, sevensegment.OnRadius-sevensegment.RampRound, math.Hypot(rise.C2.X, rise.C2.Y), 1e-9)
	require.InDelta(t, 4.8939, rise.LineLength, 1e-3)
	fall := dm.Ramps[sevensegment.SegD][0].Geometry()
	require.InDelta(t, sevensegment.OnRadius-sevensegment.RampRound, math.Hypot(fall.C1.X, fall.C1.Y), 1e-9)
	require.InDelta(t, sevensegment.OffRadius+sevensegment.RampRound, math.Hypot(fall.C2.X, fall.C2.Y), 1e-9)
}

func TestSevenSegmentChannel(t *testing.T) {
	t.Parallel()
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	half := math.Asin((sevensegment.GrooveWidth/2)/(sevensegment.OnRadius+sevensegment.GrooveWidth/2)) * 180 / math.Pi
	// The entry channel is at the follower when psi = EntryAngle + 360*n; it lies inside digit 8's dwell [11.5, 26].
	for _, s := range sevensegment.Segments() {
		for _, psi := range []float64{sevensegment.EntryAngle - half, sevensegment.EntryAngle, sevensegment.EntryAngle + half} {
			require.InDelta(t, sevensegment.OnRadius, dm.GrooveRadius(s, psi), 1e-9, "segment %s psi %.2f", s, psi)
		}
	}
}
