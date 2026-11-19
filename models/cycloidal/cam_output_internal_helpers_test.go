package cycloidal

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func verifyBody(t *testing.T, name string, body *decad.Body) {
	t.Helper()
	require.True(t, body.IsSolid(), name)
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err, name)
	require.True(t, mesh.BoundaryVerified(), name)
	require.True(t, mesh.VolumeVerified(), name)
}
