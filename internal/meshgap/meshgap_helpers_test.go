package meshgap_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func bearingMesh(t *testing.T, p ballbearing.Params, origin r3.Vec) meshgap.Mesh {
	t.Helper()
	f, err := r3.NewFrame(origin, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	p.Placement = &f
	m, err := ballbearing.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified() && mesh.VolumeVerified())
	result, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return result
}
