package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// rotorPoint returns the world point of the rotor-local point (a, b, u) of segment s, where a and b are the diagonal
// coordinates of the yellow and black faces.
func rotorPoint(s Segment, a, b, u float64) r3.Vec {
	f := rotorFrame(s)
	xr := (a - b) / math.Sqrt2
	yr := (a + b) / math.Sqrt2
	return f.Origin().Add(f.U().Scale(xr)).Add(f.V().Scale(yr)).Add(f.N().Scale(u))
}

// inBody reports whether p lies inside the closed mesh of b: a ray off the axes crosses it an odd number of times.
func inBody(t *testing.T, b *decad.Body, p r3.Vec) bool {
	t.Helper()
	var mesh *decad.Mesh
	var err error
	for _, tol := range []float64{0.01, 0.02, 0.04, 0.05} {
		if mesh, err = b.Tessellate(t.Context(), units.Millimeters(tol)); err == nil {
			break
		}
	}
	require.NoError(t, err)
	vs, ts := mesh.Vertices(), mesh.Triangles()
	dir := r3.Vec{X: 1, Y: 1.37e-4, Z: 2.71e-4}
	crossings := 0
	for _, tri := range ts {
		a, b2, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		e1, e2 := b2.Sub(a), c.Sub(a)
		h := dir.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < 1e-12 {
			continue
		}
		s := p.Sub(a)
		u := s.Dot(h) / det
		if u < 0 || u > 1 {
			continue
		}
		q := s.Cross(e1)
		v := dir.Dot(q) / det
		if v < 0 || u+v > 1 {
			continue
		}
		if e2.Dot(q)/det > 0 {
			crossings++
		}
	}
	return crossings%2 == 1
}

// TestSevenSegmentRotor builds one rotor of each type and checks its volumes, its split, its centre of mass and its
// probes. Type A serves a, g, d and type B serves b, c, e, f.
func TestSevenSegmentRotor(t *testing.T) {
	for _, tc := range []struct {
		seg        Segment
		total      float64
		com, beta  float64
		notchProbe [][2]float64 // local (u, 0) of each notch's empty probe, and of each pocket
	}{
		{SegG, 865.94, 0.712, 92.0, nil},
		{SegB, 665.62, 0.909, 94.0, nil},
	} {
		t.Run(tc.seg.String(), func(t *testing.T) {
			// Before the flat, notches and stubs the two bodies hold the rotor: their volumes add up to it.
			y0, b0, rv := sharedRotorSplit(t)
			yv, err := y0.Volume()
			require.NoError(t, err)
			bv, err := b0.Volume()
			require.NoError(t, err)
			require.InDelta(t, rv, yv.Value.Base()+bv.Value.Base(), 0.1)
			require.InDelta(t, 1252.29, rv, 0.5)

			f := sharedFlag(t, tc.seg)
			require.True(t, f.Yellow.IsSolid())
			require.True(t, f.Black.IsSolid())
			y, err := f.Yellow.Volume()
			require.NoError(t, err)
			b, err := f.Black.Volume()
			require.NoError(t, err)
			require.InDelta(t, 105.41, y.Value.Base(), 0.5) // the slab and the tips (93.46 before the tips)
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
			require.True(t, inBody(t, f.Yellow, rotorPoint(tc.seg, 3.4, 0, 0)))
			require.True(t, inBody(t, f.Black, rotorPoint(tc.seg, 3.4, 3.4, 0)))
			// The tips: yellow inside the chamfer, a black rim beside it, black past the yellow's end.
			for _, sa := range []float64{-1, 1} {
				require.True(t, inBody(t, f.Yellow, rotorPoint(tc.seg, 2.4, 0, sa*10.9)), "tip yellow")
				require.True(t, inBody(t, f.Black, rotorPoint(tc.seg, 2.4, 2.3, sa*10.9)), "tip rim")
				require.True(t, inBody(t, f.Black, rotorPoint(tc.seg, 1.6, 0, sa*11.9)), "past the yellow end")
			}
			for _, n := range rotorNotches(tc.seg) {
				mid := (n[0] + n[1]) / 2
				require.False(t, inBody(t, f.Black, f.fromLocal(tc.seg, 0, -2, mid)), "notch")
				require.False(t, inBody(t, f.Black, f.fromLocal(tc.seg, 1.8, 0, (n[1]+n[2])/2)), "pocket")
			}

			// Corners, print axis away from the yellow face.
			up := rotorPoint(tc.seg, -1, 0, 0).Sub(rotorPoint(tc.seg, 0, 0, 0))
			fs, err := cornercheck.Check(t.Context(), f.Yellow, up, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
			var ex []cornercheck.Exempt
			for _, sa := range []float64{-1, 1} {
				for _, sb := range []float64{-1, 1} {
					p := rotorPoint(tc.seg, 3.35, sb*2.55, sa*10.4) // where the black rim meets the yellow chamfer
					ex = append(ex, cornercheck.Exempt{Reason: "shared boundary with the yellow body",
						Near: func(q r3.Vec) bool { return q.Sub(p).Len() < 0.25 }})
				}
			}
			axisDir := fr.N()
			origin := fr.Origin()
			ex = append(ex, cornercheck.Exempt{Reason: "stub root: the 45 degree cone read obliquely", Optional: true,
				Near: func(q r3.Vec) bool {
					d := q.Sub(origin)
					along := d.Dot(axisDir)
					radial := d.Sub(axisDir.Scale(along)).Len()
					return math.Abs(math.Abs(along)-12.85) < 0.15 && radial < 1.2
				}})
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
			require.InDelta(t, cx-RotorHalfLength-2, box.Min.X, 0.05)
			require.InDelta(t, cx+RotorHalfLength+2, box.Max.X, 0.05)
		} else {
			require.InDelta(t, cy-RotorHalfLength-2, box.Min.Y, 0.05)
			require.InDelta(t, cy+RotorHalfLength+2, box.Max.Y, 0.05)
		}
	}
}
