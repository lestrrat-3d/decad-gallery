package m2nut_test

// CI: isolated-tests

import (
	"context"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestM2NutDefault(t *testing.T) {
	doc := decad.New()
	m, err := m2nut.Build(t.Context(), doc, m2nut.DefaultParams())
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()))
	require.True(t, m.Body.IsSolid())
	for _, s := range []*sketch.Sketch{m.Hexagon, m.LowerChamfer, m.UpperChamfer} {
		requireFullyConstrained(t, s)
	}

	box, err := m.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, -m2nut.AcrossCorners()/2, box.Min.X, tol, "corners lie on the X axis")
	require.InDelta(t, m2nut.AcrossCorners()/2, box.Max.X, tol)
	require.InDelta(t, -m2nut.AcrossFlats/2, box.Min.Y, tol, "flats face Y")
	require.InDelta(t, m2nut.AcrossFlats/2, box.Max.Y, tol)
	require.InDelta(t, 0, box.Min.Z, tol)
	require.InDelta(t, m2nut.Thickness, box.Max.Z, tol)

	volume, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, expectedVolume(), volume.Value.Base(), volume.Bound.Base()+1e-3)

	counts := map[decad.SurfaceKind]int{}
	for _, face := range m.Body.Faces() {
		counts[face.Surface().Kind()]++
	}
	require.Equal(t, 8, counts[decad.KindPlane], "six flats and two bearing faces")
	require.Equal(t, 2, counts[decad.KindCone], "two chamfers")
	require.Equal(t, 1, counts[decad.KindCylinder], "bore")
	require.Equal(t, 11, len(m.Body.Faces()))

	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestM2NutFitsM2Bolt(t *testing.T) {
	require.Equal(t, m2bolt.ShankDiameter, m2nut.BoreDiameter)
}

func TestM2NutPlacement(t *testing.T) {
	doc := decad.New()
	first, err := m2nut.Build(t.Context(), doc, m2nut.DefaultParams())
	require.NoError(t, err)
	// The nut's +Z maps to world +X: it spans x in [10, 11.6] with its corners along Z.
	frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	placed, err := m2nut.Build(t.Context(), doc, m2nut.Params{Placement: &frame})
	require.NoError(t, err)
	require.Equal(t, 2, len(doc.Bodies()))
	box, err := placed.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, 10, box.Min.X, tol)
	require.InDelta(t, 10+m2nut.Thickness, box.Max.X, tol)
	require.InDelta(t, 5-m2nut.AcrossCorners()/2, box.Min.Y, tol)
	require.InDelta(t, 5+m2nut.AcrossCorners()/2, box.Max.Y, tol)
	_, err = first.Body.Bounds()
	require.NoError(t, err, "the first nut is untouched")
	requireFullyConstrained(t, placed.Hexagon)
}

func TestM2NutRejectsInvalidInputs(t *testing.T) {
	doc := decad.New()
	_, err := m2nut.Build(t.Context(), doc, m2nut.Params{Placement: &r3.Frame{}})
	require.Error(t, err)
	require.Zero(t, len(doc.Bodies()))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = m2nut.Build(ctx, doc, m2nut.DefaultParams())
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, len(doc.Bodies()))
	_, err = m2nut.Build(t.Context(), nil, m2nut.DefaultParams())
	require.ErrorContains(t, err, "document")
	_, err = m2nut.Build(nil, doc, m2nut.DefaultParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
