package m5bolt_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	require.NotNil(t, s)
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Equal(t, 0, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.NoError(t, report.Check())
}

// headVolume integrates the chamfered hexagonal head with the midpoint rule over the corner chamfer ring.
func headVolume() float64 {
	const steps = 20000
	s, e := m5bolt.AcrossFlats, m5bolt.AcrossCorners()
	dw := m5bolt.BearingDiameter
	slope := math.Tan(m5bolt.ChamferAngle * math.Pi / 180)
	depth := (e/2 - dw/2) * slope
	hexArea := math.Sqrt(3) / 2 * s * s
	// Removed volume of one chamfer: the part of the hexagon outside the cone, integrated over height.
	removed := 0.0
	dz := depth / steps
	for i := 0; i < steps; i++ {
		z := (float64(i) + 0.5) * dz
		radius := dw/2 + (depth-z)/slope
		removed += (hexArea - hexSection(radius, s)) * dz
	}
	return hexArea*m5bolt.HeadHeight - 2*removed
}

// hexSection returns the area of the regular hexagon of across-flats s that lies inside the circle of radius r.
func hexSection(r, s float64) float64 {
	const steps = 4000
	apothem := s / 2
	// Integrate over one 30 degree wedge from a flat's midpoint to a corner, then multiply by twelve.
	area := 0.0
	dtheta := (math.Pi / 6) / steps
	for i := 0; i < steps; i++ {
		theta := (float64(i) + 0.5) * dtheta
		rho := math.Min(r, apothem/math.Cos(theta))
		area += 0.5 * rho * rho * dtheta
	}
	return 12 * area
}

func TestM5BoltDefault(t *testing.T) {
	doc := decad.New()
	m, err := m5bolt.Build(t.Context(), doc, m5bolt.DefaultParams())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, m.Body.IsSolid())
	requireFullyConstrained(t, m.Head)
	requireFullyConstrained(t, m.Shank)

	box, err := m.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	e, s := m5bolt.AcrossCorners(), m5bolt.AcrossFlats
	require.InDelta(t, -e/2, box.Min.X, tol)
	require.InDelta(t, e/2, box.Max.X, tol)
	require.InDelta(t, -s/2, box.Min.Y, tol)
	require.InDelta(t, s/2, box.Max.Y, tol)
	require.InDelta(t, -25.0, box.Min.Z, tol)
	require.InDelta(t, m5bolt.HeadHeight, box.Max.Z, tol)

	r, c, l := m5bolt.ShankDiameter/2, m5bolt.TipChamfer, 25.0
	tipRing := 2 * math.Pi * (r - c/3) * (c * c / 2)
	want := headVolume() + math.Pi*r*r*l - tipRing
	volume, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, want, volume.Value.Base(), volume.Bound.Base()+1e-3)

	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())

	counts := map[decad.SurfaceKind]int{}
	for _, f := range m.Body.Faces() {
		switch k := f.Surface().Kind(); k {
		case decad.KindPlane, decad.KindCone, decad.KindCylinder:
			counts[k]++
		default:
			require.Failf(t, "unknown face kind", "kind %v", k)
		}
	}
	require.Equal(t, 3, counts[decad.KindCone])
	require.Equal(t, 1, counts[decad.KindCylinder])
}

func TestM5BoltLengths(t *testing.T) {
	for _, l := range m5bolt.StandardLengths() {
		doc := decad.New()
		m, err := m5bolt.Build(t.Context(), doc, m5bolt.Params{Length: l})
		require.NoError(t, err, "length %v", l)
		require.Len(t, doc.Bodies(), 1)
		box, err := m.Body.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-8
		require.InDelta(t, -l, box.Min.Z, tol)
		require.InDelta(t, m5bolt.HeadHeight, box.Max.Z, tol)
	}
	_, err := m5bolt.Build(t.Context(), decad.New(), m5bolt.Params{Length: 22})
	require.Error(t, err)
}

func TestM5BoltPlacement(t *testing.T) {
	doc := decad.New()
	first, err := m5bolt.Build(t.Context(), doc, m5bolt.DefaultParams())
	require.NoError(t, err)

	// Local +Z maps to world +X, local +X to world +Y, origin at (10, 20, 30).
	frame, err := r3.NewFrame(r3.NewVec(10, 20, 30), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := m5bolt.DefaultParams()
	p.Placement = &frame
	placed, err := m5bolt.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)

	box, err := placed.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	e, s := m5bolt.AcrossCorners(), m5bolt.AcrossFlats
	require.InDelta(t, 10-25.0, box.Min.X, tol)
	require.InDelta(t, 10+m5bolt.HeadHeight, box.Max.X, tol)
	require.InDelta(t, 20-e/2, box.Min.Y, tol)
	require.InDelta(t, 20+e/2, box.Max.Y, tol)
	require.InDelta(t, 30-s/2, box.Min.Z, tol)
	require.InDelta(t, 30+s/2, box.Max.Z, tol)

	_, err = first.Body.Bounds()
	require.NoError(t, err)
}

func TestM5BoltRejectsInvalidInputs(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	for name, p := range map[string]m5bolt.Params{
		"zero":      {},
		"nan":       {Length: math.NaN()},
		"inf":       {Length: math.Inf(1)},
		"off-list":  {Length: 22},
		"placement": {Length: 25, Placement: &r3.Frame{}},
	} {
		doc := decad.New()
		_, err := m5bolt.Build(t.Context(), doc, p)
		require.Error(t, err, name)
		require.Empty(t, doc.Bodies(), name)
	}

	doc := decad.New()
	_, err := m5bolt.Build(cancelled, doc, m5bolt.DefaultParams())
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, doc.Bodies())

	_, err = m5bolt.Build(t.Context(), nil, m5bolt.DefaultParams())
	require.ErrorContains(t, err, "document")
	//nolint:staticcheck
	_, err = m5bolt.Build(nil, decad.New(), m5bolt.DefaultParams())
	require.ErrorContains(t, err, "context")
}

func TestM5BoltStandardLengths(t *testing.T) {
	want := []float64{16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80}
	got := m5bolt.StandardLengths()
	require.Equal(t, want, got)
	got[0] = 999
	require.Equal(t, want, m5bolt.StandardLengths())
}
