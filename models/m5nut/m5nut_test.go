package m5nut_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Zero(t, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.Empty(t, report.Check())
}

// clippedHexArea returns the area of the disc of radius rho clipped to the hexagon.
func clippedHexArea(rho float64) float64 {
	apothem := m5nut.AcrossFlats / 2
	hexArea := math.Sqrt(3) / 2 * m5nut.AcrossFlats * m5nut.AcrossFlats
	switch {
	case rho <= apothem:
		return math.Pi * rho * rho
	case rho >= m5nut.AcrossCorners()/2:
		return hexArea
	}
	half := math.Acos(apothem / rho)
	return math.Pi*rho*rho - 6*rho*rho*(half-math.Sin(half)*math.Cos(half))
}

func expectedVolume() float64 {
	const steps = 20000
	hexArea := math.Sqrt(3) / 2 * m5nut.AcrossFlats * m5nut.AcrossFlats
	slope := math.Tan(m5nut.ChamferAngle * math.Pi / 180)
	bearingR := m5nut.BearingDiameter / 2
	depth := (m5nut.AcrossCorners()/2 - bearingR) * slope
	chamfer := 0.0
	for i := 0; i < steps; i++ {
		z := (float64(i) + .5) / steps * depth
		chamfer += (hexArea - clippedHexArea(bearingR+z/slope)) * depth / steps
	}
	d := m5nut.BoreDiameter
	return hexArea*m5nut.Thickness - math.Pi*d*d/4*m5nut.Thickness - 2*chamfer
}

func TestM5NutDefault(t *testing.T) {
	doc := decad.New()
	m, err := m5nut.Build(t.Context(), doc, m5nut.DefaultParams())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, m.Body.IsSolid())
	requireFullyConstrained(t, m.Hexagon)
	requireFullyConstrained(t, m.LowerChamfer)
	requireFullyConstrained(t, m.UpperChamfer)

	box, err := m.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	e, s := m5nut.AcrossCorners(), m5nut.AcrossFlats
	require.InDelta(t, -e/2, box.Min.X, tol)
	require.InDelta(t, e/2, box.Max.X, tol)
	require.InDelta(t, -s/2, box.Min.Y, tol)
	require.InDelta(t, s/2, box.Max.Y, tol)
	require.InDelta(t, 0, box.Min.Z, tol)
	require.InDelta(t, m5nut.Thickness, box.Max.Z, tol)

	vol, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, expectedVolume(), vol.Value.Base(), vol.Bound.Base()+1e-3)

	counts := map[decad.SurfaceKind]int{}
	for _, f := range m.Body.Faces() {
		counts[f.Surface().Kind()]++
	}
	require.Equal(t, 8, counts[decad.KindPlane])
	require.Equal(t, 2, counts[decad.KindCone])
	require.Equal(t, 1, counts[decad.KindCylinder])
	require.Len(t, m.Body.Faces(), 11)

	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestM5NutFitsM5Bolt(t *testing.T) {
	require.Equal(t, m5bolt.ShankDiameter, m5nut.BoreDiameter)
}

func TestM5NutPlacement(t *testing.T) {
	doc := decad.New()
	first, err := m5nut.Build(t.Context(), doc, m5nut.DefaultParams())
	require.NoError(t, err)
	frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	placed, err := m5nut.Build(t.Context(), doc, m5nut.Params{Placement: &frame})
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)

	box, err := placed.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	e := m5nut.AcrossCorners()
	require.InDelta(t, 10, box.Min.X, tol)
	require.InDelta(t, 10+m5nut.Thickness, box.Max.X, tol)
	require.InDelta(t, 5-e/2, box.Min.Y, tol)
	require.InDelta(t, 5+e/2, box.Max.Y, tol)

	_, err = first.Body.Bounds()
	require.NoError(t, err)
	requireFullyConstrained(t, placed.Hexagon)
}

func TestM5NutRejectsInvalidInputs(t *testing.T) {
	doc := decad.New()
	_, err := m5nut.Build(t.Context(), doc, m5nut.Params{Placement: &r3.Frame{}})
	require.Error(t, err)
	require.Empty(t, doc.Bodies())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = m5nut.Build(ctx, doc, m5nut.DefaultParams())
	require.True(t, errors.Is(err, context.Canceled))
	require.Empty(t, doc.Bodies())

	_, err = m5nut.Build(t.Context(), nil, m5nut.DefaultParams())
	require.ErrorContains(t, err, "document")

	//nolint:staticcheck
	_, err = m5nut.Build(nil, doc, m5nut.DefaultParams())
	require.ErrorContains(t, err, "context")
}
