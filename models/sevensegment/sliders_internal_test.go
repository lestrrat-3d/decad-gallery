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
	want := map[Segment]float64{SegA: 979.14, SegG: 979.14, SegD: 979.14, SegB: 981.64, SegC: 981.64, SegE: 996.01, SegF: 996.01}
	vols := map[Segment]float64{}
	for _, seg := range Segments() {
		t.Run(seg.String(), func(t *testing.T) {
			_, body := sharedSlider(t, seg)
			require.True(t, body.IsSolid())
			vol, err := body.Volume()
			require.NoError(t, err)
			vols[seg] = vol.Value.Base()
			require.InDelta(t, want[seg], vol.Value.Base(), 2)
			require.LessOrEqual(t, vol.Value.Base(), 1069.0)
			yi := r.dm.Station(seg)
			_, _, horizontal := segmentCentre(seg)
			ef := seg == SegE || seg == SegF
			// Head or arm top 72.8: material just under it and none just over it, away from the blade and tongue.
			headX, headY := 0.0, yi-4
			if !horizontal {
				headX, headY = 14, yi
				if ef {
					headX = -14
				}
			}
			require.True(t, inBody(t, body, r3.Vec{X: headX, Y: headY, Z: sliderHeadTop - 0.1}), "head top")
			require.False(t, inBody(t, body, r3.Vec{X: headX, Y: headY, Z: sliderHeadTop + 0.1}), "head top")
			// Blade: 1.1 thick (its rotor-side face 0.1 back from the old 1.2) and 7.0 (a, g, d) or 4.8 (b-f) long, at z 80.
			if horizontal {
				require.True(t, inBody(t, body, r3.Vec{X: 3.4, Y: yi - 7.2, Z: 80}))
				require.False(t, inBody(t, body, r3.Vec{X: 3.6, Y: yi - 7.2, Z: 80}))
				require.False(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 7.9, Z: 80}))
				require.False(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 6.65, Z: 80}))
				require.True(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 6.75, Z: 80}))
			} else {
				bx := 9.3
				if ef {
					bx = -9.3
				}
				require.True(t, inBody(t, body, r3.Vec{X: bx, Y: yi + 2.9, Z: 80}))
				require.False(t, inBody(t, body, r3.Vec{X: bx, Y: yi + 3.1, Z: 80}))
				require.True(t, inBody(t, body, r3.Vec{X: bx, Y: yi - 1.7, Z: 80}))
				require.False(t, inBody(t, body, r3.Vec{X: bx, Y: yi - 1.9, Z: 80}))
			}
			// The rotor pin: radius 0.50 on the axis (sc, zp), 3.6 long past the pin face; probes on and off it.
			var centre, along r3.Vec
			var ue float64
			switch {
			case horizontal:
				centre, along, ue = r3.Vec{Y: yi + 1.5, Z: 80.85}, r3.Vec{X: 1}, -1.9
			case ef:
				centre, along, ue = r3.Vec{X: -18, Z: 80.85 + raise(seg)}, r3.Vec{Y: 1}, yi-0.2
			default:
				centre, along, ue = r3.Vec{X: 15, Z: 80.85 + raise(seg)}, r3.Vec{Y: 1}, yi-0.2
			}
			at := func(u, dr float64, dir r3.Vec) r3.Vec {
				p := centre.Add(along.Scale(u))
				return p.Add(dir.Scale(dr))
			}
			perp := r3.Vec{Z: 1}
			require.True(t, inBody(t, body, at(ue+2, 0.45, perp)), "pin")
			require.False(t, inBody(t, body, at(ue+2, 0.6, perp)), "pin radius")
			require.False(t, inBody(t, body, at(ue+3.7, 0, perp)), "pin length")
			require.True(t, inBody(t, body, at(ue+3.5, 0, perp)), "pin length")
			if !horizontal {
				require.InDelta(t, map[bool]float64{false: 81.05, true: 81.10}[ef], 80.85+raise(seg), 1e-9)
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
// design's values: the pin axis at z 80.85 plus the slider's raise, and at its station beside the stem (1.5 along y for a, g,
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
				on = v.Z > 79 && v.Y > yi-1.0 && v.X > -1.9+0.05
			} else {
				on = v.Z > 79 && math.Abs(v.X) > 12 && v.Y > yi-0.2+0.05
			}
			if !on {
				continue
			}
			lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y), Z: math.Min(lo.Z, v.Z)}
			hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y), Z: math.Max(hi.Z, v.Z)}
		}
		require.Less(t, lo.Z, hi.Z, "no pin vertices on slider %s", seg)
		axisZ := (lo.Z + hi.Z) / 2
		require.InDelta(t, 80.85+raise(seg), axisZ, 0.02, "pin axis height of slider %s", seg)
		require.InDelta(t, 2*flagPinRadius, hi.Z-lo.Z, 0.02, "pin diameter of slider %s", seg)
		if horizontal {
			require.InDelta(t, yi+1.5, (lo.Y+hi.Y)/2, 0.02, "pin station of slider %s", seg)
		} else {
			want := 15.0
			if seg == SegE || seg == SegF {
				want = -18.0
			}
			require.InDelta(t, want, (lo.X+hi.X)/2, 0.02, "pin station of slider %s", seg)
		}
	}
}
