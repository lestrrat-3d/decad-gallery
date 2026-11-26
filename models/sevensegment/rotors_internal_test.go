package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentRotor builds one rotor of each type and checks its volumes, its split, its centre of mass and its
// probes. Type A serves a, g, d and type B serves b, c, e, f.
func TestSevenSegmentRotor(t *testing.T) {
	for _, tc := range []struct {
		seg        Segment
		total      float64
		com, beta  float64
		notchProbe [][2]float64 // local (u, 0) of each notch's empty probe, and of each pocket
	}{
		{SegG, 2808.76, 1.033, 92.4, nil},
		{SegB, 2052.95, 1.279, 95.2, nil},
	} {
		t.Run(tc.seg.String(), func(t *testing.T) {
			// Before the flat, notches and bores the two bodies hold the rotor: their volumes add up to it.
			y0, b0, rv := sharedRotorSplit(t)
			yv, err := y0.Volume()
			require.NoError(t, err)
			bv, err := b0.Volume()
			require.NoError(t, err)
			require.InDelta(t, rv, yv.Value.Base()+bv.Value.Base(), 0.1)
			require.InDelta(t, 4226.47, rv, 0.5)

			f := sharedFlag(t, tc.seg)
			require.True(t, f.Yellow.IsSolid())
			require.True(t, f.Black.IsSolid())
			y, err := f.Yellow.Volume()
			require.NoError(t, err)
			b, err := f.Black.Volume()
			require.NoError(t, err)
			require.InDelta(t, 259.44, y.Value.Base(), 0.5) // the slab and the tips (the 0.8 inset is a fit)
			require.InDelta(t, tc.total, y.Value.Base()+b.Value.Base(), 1.0)

			// Centre of mass of the two bodies, in the rotor's x and y, as a distance and an angle from x.
			fr := rotorFrame(tc.seg)
			cy, err := f.Yellow.Centroid()
			require.NoError(t, err)
			cb, err := f.Black.Centroid()
			require.NoError(t, err)
			vy, vb := y.Value.Base(), b.Value.Base()
			c := cy.Value.Scale(vy / (vy + vb)).Add(cb.Value.Scale(vb / (vy + vb))).Sub(fr.Origin())
			xr, yr := c.Dot(fr.U()), c.Dot(fr.V())
			require.InDelta(t, tc.com, math.Hypot(xr, yr), 0.02)
			require.InDelta(t, tc.beta, math.Atan2(yr, xr)*180/math.Pi, 1.0)

			// Probes: yellow on the yellow face's slab, black beside it, empty in the notches and pockets.
			require.True(t, inBody(t, f.Yellow, rotorPoint(tc.seg, 5.1, 0, 0)))
			require.True(t, inBody(t, f.Black, rotorPoint(tc.seg, 5.1, 5.1, 0)))
			// The tips: yellow inside the chamfer, a black rim beside it, black past the yellow's end.
			for _, sa := range []float64{-1, 1} {
				require.True(t, inBody(t, f.Yellow, rotorPoint(tc.seg, yellowTipInner-16.35+0.35, 0, sa*16.35)), "tip yellow")
				require.True(t, inBody(t, f.Black, rotorPoint(tc.seg, yellowTipInner-16.35+0.35, yellowTipK-16.35+0.25, sa*16.35)), "tip rim")
				require.True(t, inBody(t, f.Black, rotorPoint(tc.seg, 2.4, 0, sa*(yellowTipEnd+0.3))), "past the yellow end")
			}
			for _, n := range rotorNotches(tc.seg) {
				mid := (n[0] + n[1]) / 2
				require.False(t, inBody(t, f.Black, f.fromLocal(tc.seg, 0, -3.0, mid)), "notch")
				require.False(t, inBody(t, f.Black, f.fromLocal(tc.seg, 2.7, 0, (n[1]+n[2])/2)), "pocket")
			}

			// Corners, print axis away from the yellow face.
			up := rotorPoint(tc.seg, -1, 0, 0).Sub(rotorPoint(tc.seg, 0, 0, 0))
			fs, err := cornercheck.Check(t.Context(), f.Yellow, up, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
			var ex []cornercheck.Exempt
			for _, sa := range []float64{-1, 1} {
				for _, sb := range []float64{-1, 1} {
					p := rotorPoint(tc.seg, RotorHalfWidth-0.4, sb*(RotorHalfWidth-1.2), sa*(yellowTipK-(RotorHalfWidth-1.2))) // where the black rim meets the yellow chamfer
					ex = append(ex, cornercheck.Exempt{Reason: "shared boundary with the yellow body",
						Near: func(q r3.Vec) bool { return q.Sub(p).Len() < 0.25 }})
				}
			}
			fs, err = cornercheck.Check(t.Context(), f.Black, up, ex)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 8))
		})
	}
}

// TestSevenSegmentRotorAll builds the rotor of every segment: the two types, in their seven placements.
func TestSevenSegmentRotorAll(t *testing.T) {
	for _, s := range Segments() {
		f := sharedFlag(t, s)
		require.True(t, f.Yellow.IsSolid() && f.Black.IsSolid(), "segment %s", s)
		box, err := f.Black.Bounds()
		require.NoError(t, err)
		cx, cy, horizontal := segmentCentre(s)
		if horizontal {
			require.InDelta(t, cx-RotorHalfLength, box.Min.X, 0.05)
			require.InDelta(t, cx+RotorHalfLength, box.Max.X, 0.05)
		} else {
			require.InDelta(t, cy-RotorHalfLength, box.Min.Y, 0.05)
			require.InDelta(t, cy+RotorHalfLength, box.Max.Y, 0.05)
		}
	}
}
