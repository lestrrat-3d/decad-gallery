package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentSliders builds the three flag slider types at the unlit pose and checks their volumes, their
// pin, blade and head, and their corners along the print axis (a, g, d: the -X face down; b-f: the -Y face down).
func TestSevenSegmentSliders(t *testing.T) {
	r := newTestRun(t)
	// The volumes with the 0.5 chamfers of "Surface edge finishing" on the stem faces. Each is the same build without
	// the chamfers (a, g, d 3325.76; b, c 3335.59; e, f 3384.44) less the chamfers: 0.5 x 0.5 / 2 = 0.125 mm2 of section along
	// every run of 20 mm or more of the two faces' outer loops (168.2, 271.3 and 278.5 mm), 21.0, 33.9 and 34.8 mm3. The built
	// removal is 23.8, 34.0 and 34.9: it agrees to 0.1 for b to f, and a, g and d lose 2.8 more where their runs meet.
	want := map[Segment]float64{SegA: 3301.94, SegG: 3301.94, SegD: 3301.94, SegB: 3301.62, SegC: 3301.62, SegE: 3349.56, SegF: 3349.56}
	vols := map[Segment]float64{}
	for _, seg := range Segments() {
		t.Run(seg.String(), func(t *testing.T) {
			_, body := sharedSlider(t, seg)
			require.True(t, body.IsSolid())
			vol, err := body.Volume()
			require.NoError(t, err)
			vols[seg] = vol.Value.Base()
			require.InDelta(t, want[seg], vol.Value.Base(), 2)
			require.LessOrEqual(t, vol.Value.Base(), 3607.875)
			yi := r.dm.Station(seg)
			_, _, horizontal := segmentCentre(seg)
			ef := seg == SegE || seg == SegF
			// Head or arm top 72.8: material just under it and none just over it, away from the blade and tongue.
			headX, headY := 0.0, yi-6.0
			if !horizontal {
				headX, headY = 21.0, yi
				if ef {
					headX = -21.0
				}
			}
			require.True(t, inBody(t, body, r3.Vec{X: headX, Y: headY, Z: sliderHeadTop - 0.1}), "head top")
			require.False(t, inBody(t, body, r3.Vec{X: headX, Y: headY, Z: sliderHeadTop + 0.1}), "head top")
			// Blade: 1.65 thick (its rotor-side face 0.1 back from the old 1.2) and 10.5 (a, g, d) or 7.2 (b-f) long, 10.8 over the head top.
			bz := sliderHeadTop + 10.8
			if horizontal {
				require.True(t, inBody(t, body, r3.Vec{X: 5.25 - 0.1, Y: yi - 10.8, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: 5.25 + 0.1, Y: yi - 10.8, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 11.7 - 0.1, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 10.05 + 0.05, Z: bz}))
				require.True(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 10.05 - 0.05, Z: bz}))
			} else {
				bx := 13.95
				if ef {
					bx = -13.95
				}
				require.True(t, inBody(t, body, r3.Vec{X: bx, Y: yi + 4.5 - 0.1, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: bx, Y: yi + 4.5 + 0.1, Z: bz}))
				require.True(t, inBody(t, body, r3.Vec{X: bx, Y: yi - StemThickness/2 + 0.1, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: bx, Y: yi - StemThickness/2 - 0.1, Z: bz}))
			}
			// The rotor pin: radius 0.75 on the axis (sc, zp), 5.4 long past the pin face; probes on and off it.
			var centre, along r3.Vec
			var ue float64
			switch {
			case horizontal:
				centre, along, ue = r3.Vec{Y: yi + pinStationAGD, Z: pinAxisZ}, r3.Vec{X: 1}, -2.85
			case ef:
				centre, along, ue = r3.Vec{X: -pinStationEF, Z: pinAxisZ + raise(seg)}, r3.Vec{Y: 1}, yi-0.3
			default:
				centre, along, ue = r3.Vec{X: pinStationBC, Z: pinAxisZ + raise(seg)}, r3.Vec{Y: 1}, yi-0.3
			}
			at := func(u, dr float64, dir r3.Vec) r3.Vec {
				p := centre.Add(along.Scale(u))
				return p.Add(dir.Scale(dr))
			}
			perp := r3.Vec{Z: 1}
			require.True(t, inBody(t, body, at(ue+3.0, flagPinRadius-0.05, perp)), "pin")
			require.False(t, inBody(t, body, at(ue+3.0, flagPinRadius+0.1, perp)), "pin radius")
			require.False(t, inBody(t, body, at(ue+flagPinLength+0.1, 0, perp)), "pin length")
			require.True(t, inBody(t, body, at(ue+flagPinLength-0.1, 0, perp)), "pin length")
			if !horizontal {
				require.InDelta(t, pinAxisZ+map[bool]float64{false: 0.20, true: 0.25}[ef], pinAxisZ+raise(seg), 1e-9)
			}
			up := r3.Vec{Y: 1}
			if horizontal {
				up = r3.Vec{X: 1}
			}
			fs, err := cornercheck.Check(t.Context(), body, up, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
	require.InDelta(t, vols[SegA], vols[SegG], 0.01)
	require.InDelta(t, vols[SegA], vols[SegD], 0.01)
	require.InDelta(t, vols[SegB], vols[SegC], 0.01)
	require.InDelta(t, vols[SegE], vols[SegF], 0.01)
}

// TestSevenSegmentBuiltKinematics reads the rotor pin of every built slider and checks where it stands against the
// design's values: the pin axis at pinAxisZ (80.85 in the original 1.0x design) plus the slider's raise, and at its station beside the stem (1.5 along y for a, g,
// d from the slider's station, x 15 for b and c, x -18 for e and f). These are the contract numbers of instructions.md
// "Kinematics and play", read from the mesh; a pin built higher or off its line moves the rotor's angles.
func TestSevenSegmentBuiltKinematics(t *testing.T) {
	r := newTestRun(t)
	for _, seg := range Segments() {
		_, body := sharedSlider(t, seg)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.05))
		require.NoError(t, err)
		yi := r.dm.Station(seg)
		_, _, horizontal := segmentCentre(seg)
		lo, hi := r3.Vec{X: 1e9, Y: 1e9, Z: 1e9}, r3.Vec{X: -1e9, Y: -1e9, Z: -1e9}
		for _, v := range mesh.Vertices() {
			var on bool
			if horizontal {
				on = v.Z > pinAxisZ-2.775 && v.Y > yi-1.5 && v.X > -2.85+0.05
			} else {
				on = v.Z > pinAxisZ-2.775 && math.Abs(v.X) > 18.0 && v.Y > yi-0.3+0.05
			}
			if !on {
				continue
			}
			lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y), Z: math.Min(lo.Z, v.Z)}
			hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y), Z: math.Max(hi.Z, v.Z)}
		}
		require.Less(t, lo.Z, hi.Z, "no pin vertices on slider %s", seg)
		axisZ := (lo.Z + hi.Z) / 2
		require.InDelta(t, pinAxisZ+raise(seg), axisZ, 0.02, "pin axis height of slider %s", seg)
		require.InDelta(t, 2*flagPinRadius, hi.Z-lo.Z, 0.02, "pin diameter of slider %s", seg)
		if horizontal {
			require.InDelta(t, yi+pinStationAGD, (lo.Y+hi.Y)/2, 0.02, "pin station of slider %s", seg)
		} else {
			want := pinStationBC
			if seg == SegE || seg == SegF {
				want = -pinStationEF
			}
			require.InDelta(t, want, (lo.X+hi.X)/2, 0.02, "pin station of slider %s", seg)
		}
	}
}
