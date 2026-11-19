package planetarygearbox_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestInsideCountsRepeatedProjectedVertices(t *testing.T) {
	frame, err := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	inner, _, err := fastenertool.HexPrism(t.Context(), decad.New(), frame, 1, 2)
	require.NoError(t, err)
	outer, _, err := fastenertool.HexPrism(t.Context(), decad.New(), frame, 3, 2)
	require.NoError(t, err)
	mesh, err := inner.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	require.NoError(t, err)
	points := make(map[[2]float64]struct{})
	for _, vertex := range mesh.Vertices() {
		points[[2]float64{vertex.X, vertex.Y}] = struct{}{}
	}
	require.Greater(t, len(mesh.Vertices()), len(points))
	require.Equal(t, len(mesh.Vertices()), inside(t, inner, outer))
	require.Zero(t, inside(t, outer, inner))
}
