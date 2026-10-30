package ballbearing_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBearing605Envelope(t *testing.T) {
	doc := decad.New()
	m, err := ballbearing.Build(t.Context(), doc, ballbearing.Params605ZZ())
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()))
	box, err := m.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Sub(r3.NewVec(-7, -7, 0)).Len(), box.Bound.Base()+1e-8)
	require.InDelta(t, 0, box.Max.Sub(r3.NewVec(7, 7, 5)).Len(), box.Bound.Base()+1e-8)
	requireBore(t, m.Body, 2.5, r3.NewVec(0, 0, 1))
	volume, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, math.Pi*(49-6.25)*5, volume.Value.Base(), volume.Bound.Base()+1e-6)
	report := m.Sketch.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.03), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}
