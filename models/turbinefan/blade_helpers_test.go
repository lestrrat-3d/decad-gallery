package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// bladeSizes are the rotor diameters every blade test covers: the compact, default and large rotors.
var bladeSizes = []float64{128, 192, 224}

// bladePitches are the pitches every blade test covers: the least, the default and the greatest.
var bladePitches = []float64{turbinefan.MinBladePitch, turbinefan.DefaultBladePitch, turbinefan.MaxBladePitch}

func mustSize(t *testing.T, d float64) turbinefan.Size {
	t.Helper()
	sz, err := turbinefan.SizeFor(d)
	require.NoError(t, err)
	return sz
}

// requireThickness compares the ring's extent across the chord direction with its extent along it. The chord runs at
// angle from +X toward -Z.
func requireThickness(t *testing.T, want, angle float64, ring []r3.Vec, tol, y float64) {
	t.Helper()
	along := r3.Vec{X: math.Cos(angle), Z: -math.Sin(angle)}
	across := r3.Vec{X: math.Sin(angle), Z: math.Cos(angle)}
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, v := range ring {
		for i, axis := range []r3.Vec{along, across} {
			d := v.X*axis.X + v.Z*axis.Z
			lo[i], hi[i] = math.Min(lo[i], d), math.Max(hi[i], d)
		}
	}
	require.InDelta(t, want, (hi[1]-lo[1])/(hi[0]-lo[0]), tol, "thickness ratio at radius %g", y)
}

func requireBladePrintEdgeCorners(t *testing.T, blade *turbinefan.Blade, sz turbinefan.Size, mirror bool) {
	t.Helper()
	sign := 1.0
	if mirror {
		sign = -1
	}
	r := sz.BladeStudRadius
	findings, err := cornercheck.CheckAt(t.Context(), blade.Body, r3.Vec{X: sign},
		[]float64{-r + 0.2, -2 * r / 3, -r / 3, 0, r / 3, 2 * r / 3, r - 0.01, r - 0.001, r}, nil)
	require.NoError(t, err)
	// The last cut touches the stud only along its tangent generator. Check the
	// cone and cylinder sections before accepting that zero-area tip finding.
	for i, want := range []struct{ z, radius float64 }{
		{sz.BladeStudStart, sz.BladeStudRadius - 0.3},
		{sz.BladeStudStart + 0.3, sz.BladeStudRadius},
		{sz.HubRadius, sz.BladeStudRadius},
	} {
		s := blade.Sketches[i]
		frame, err := s.Plane().Frame()
		require.NoError(t, err)
		require.InDelta(t, want.z, frame.Origin().Z, 1e-12)
		require.Equal(t, r3.Vec{X: 1}, frame.U())
		require.Equal(t, r3.Vec{Y: 1}, frame.V())
		profiles := s.Profiles()
		require.Len(t, profiles, 1)
		require.Len(t, profiles[0].Outer, 4)
		for _, edge := range profiles[0].Outer {
			arc, ok := edge.Entity.(*sketch.Arc)
			require.True(t, ok)
			cx := frame.Origin().X + arc.Center.X()
			cy := frame.Origin().Y + arc.Center.Y()
			require.InDelta(t, 0, cx, 1e-12)
			require.InDelta(t, 0, cy, 1e-12)
			ax, ay := arc.Start.X()-arc.Center.X(), arc.Start.Y()-arc.Center.Y()
			bx, by := arc.End.X()-arc.Center.X(), arc.End.Y()-arc.Center.Y()
			require.InDelta(t, want.radius, math.Hypot(ax, ay), 1e-12)
			require.InDelta(t, want.radius, math.Hypot(bx, by), 1e-12)
			require.InDelta(t, 0, ax*bx+ay*by, 1e-12)
			require.InDelta(t, want.radius*want.radius, ax*by-ay*bx, 1e-12)
		}
	}
	for _, f := range findings {
		if f.Kind != cornercheck.Tip || math.Abs(f.At.Z-(sz.BladeStudStart+0.3)) > 1e-12 ||
			math.Abs(sign*f.At.X-f.Height) > 1e-12 {
			t.Fatalf("unexpected corner: %s", cornercheck.Report([]cornercheck.Finding{f}, 1))
		}
		r := sz.BladeStudRadius
		h := f.Height
		require.Contains(t, []float64{r - 0.01, r - 0.001, r}, h)
		// A cone section has minimum radius h and two joins with material
		// angles above 90 degrees. The scanner can combine those turns only
		// while the cap is shorter than its 0.8 mm tip window.
		require.Greater(t, h, r-0.3)
		require.GreaterOrEqual(t, h, 0.4)
		b2 := r*r - h*h
		require.GreaterOrEqual(t, b2, 0.0)
		require.Less(t, 2*math.Sqrt(b2*(1+b2/(r*r))), cornercheck.TipWindow)
		require.LessOrEqual(t, math.Abs(f.At.Y), math.Sqrt(b2)+1e-9)
	}

}

func requireBladeBedContact(t *testing.T, mesh *decad.Mesh, sz turbinefan.Size, mirror bool) {
	t.Helper()
	sign := 1.0
	if mirror {
		sign = -1
	}
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	vertices := mesh.Vertices()
	bed := sz.BladeStudRadius
	for _, v := range vertices {
		require.LessOrEqual(t, sign*v.X, bed+1e-9, "material below bed")
	}
	area := 0.0
	stations := []float64{26 * sz.Scale, 40 * sz.Scale, 48 * sz.Scale, sz.TipRadius}
	limits := make([][2]float64, len(stations))
	for i := range limits {
		limits[i] = [2]float64{math.Inf(1), math.Inf(-1)}
	}
	low, high := math.Inf(1), math.Inf(-1)
	for _, triangle := range mesh.Triangles() {
		a, b, c := vertices[triangle[0]], vertices[triangle[1]], vertices[triangle[2]]
		if math.Abs(sign*a.X-bed) > 1e-9 || math.Abs(sign*b.X-bed) > 1e-9 || math.Abs(sign*c.X-bed) > 1e-9 {
			continue
		}
		area += math.Abs((b.Y-a.Y)*(c.Z-a.Z)-(c.Y-a.Y)*(b.Z-a.Z)) / 2
		for i, z := range stations {
			for _, edge := range [][2]r3.Vec{{a, b}, {b, c}, {c, a}} {
				p, q := edge[0], edge[1]
				if z < math.Min(p.Z, q.Z)-1e-9 || z > math.Max(p.Z, q.Z)+1e-9 {
					continue
				}
				if math.Abs(q.Z-p.Z) < 1e-12 {
					limits[i][0] = math.Min(limits[i][0], math.Min(p.Y, q.Y))
					limits[i][1] = math.Max(limits[i][1], math.Max(p.Y, q.Y))
					continue
				}
				y := p.Y + (q.Y-p.Y)*(z-p.Z)/(q.Z-p.Z)
				limits[i][0], limits[i][1] = math.Min(limits[i][0], y), math.Max(limits[i][1], y)
			}
		}
		low = math.Min(low, math.Min(a.Z, math.Min(b.Z, c.Z)))
		high = math.Max(high, math.Max(a.Z, math.Max(b.Z, c.Z)))
	}
	for i, z := range stations {
		require.InDelta(t, 2, limits[i][1]-limits[i][0], 1e-8, "bed contact width at radius %g", z)
	}
	wantArea := 26*sz.Scale - 14.6*sz.HubScale + 2*(sz.TipRadius-26*sz.Scale)
	require.InDelta(t, wantArea, area, 0.01, "bed contact area")
	require.InDelta(t, 14.6*sz.HubScale, low, 1e-9, "bed contact begins at the stud")
	require.InDelta(t, sz.TipRadius, high, 1e-9, "bed contact reaches the tip")
}
