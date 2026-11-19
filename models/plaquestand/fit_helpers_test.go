package plaquestand_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// gapMesh builds a meshgap mesh of body's verified tessellation at tol.
func gapMesh(t *testing.T, body *decad.Body, tol float64) meshgap.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	g, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return g
}
