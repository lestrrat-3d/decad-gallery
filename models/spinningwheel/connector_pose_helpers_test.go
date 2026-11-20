package spinningwheel_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireConnectorPose rebuilds the compliant connector at one pose and checks the
// rebuilt solid on its own, without the full assembly.
func requireConnectorPose(t *testing.T, p spinningwheel.Params) (volume, bound float64) {
	t.Helper()
	doc := spinningwheel.NewDocument()
	connector, err := spinningwheel.BuildConnector(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()), "only the one-piece connector may remain live")
	mesh, err := connector.Body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified(), "boundary at wheel angle %g", p.WheelAngle)
	require.True(t, mesh.VolumeVerified(), "volume at wheel angle %g", p.WheelAngle)
	require.Equal(t, 1, meshComponents(len(mesh.Vertices()), mesh.Triangles()), "one physical piece")
	require.Equal(t, -2, meshEulerCharacteristic(mesh.Triangles()), "both clamp passages stay open")
	require.Greater(t, len(connector.Sketches), 0)
	for i, section := range connector.Sketches {
		report := section.Verify(t.Context())
		require.True(t, report.Status == sketch.FullyConstrained, "sketch %d status %s", i, report.Status)
		require.Zero(t, report.DOF, "sketch %d", i)
		require.Zero(t, len(report.Redundant), "sketch %d", i)
		require.Zero(t, len(report.Conflicts), "sketch %d", i)
		require.NoError(t, report.Check(), "sketch %d", i)
	}
	v, err := connector.Body.Volume()
	require.NoError(t, err)
	return v.Value.Base(), v.Bound.Base()
}
