package spinningwheel_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// A real hollow bearing is connected; combining two real, separated meshes must not hide a loose component.
func TestSpinningWheelMeshComponents(t *testing.T) {
	var combinedVertices []r3.Vec
	var combinedTriangles [][3]int
	for _, x := range []float64{0, 40} {
		p := ballbearing.DefaultParams()
		placement, err := r3.NewFrame(r3.Vec{X: x}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		require.NoError(t, err)
		p.Placement = &placement
		model, err := ballbearing.Build(t.Context(), decad.New(), p)
		require.NoError(t, err)
		mesh, err := model.Body.Tessellate(t.Context(), units.Millimeters(.12), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		vertices, triangles := mesh.Vertices(), mesh.Triangles()
		require.Equal(t, 1, meshComponents(len(vertices), triangles))
		require.Zero(t, meshEulerCharacteristic(triangles), "a closed bearing has one through-hole")
		require.Equal(t, -1, meshEulerCharacteristic(triangles[1:]), "removing a face must change the topology check")
		offset := len(combinedVertices)
		for _, f := range triangles {
			combinedTriangles = append(combinedTriangles, [3]int{f[0] + offset, f[1] + offset, f[2] + offset})
		}
		combinedVertices = append(combinedVertices, vertices...)
	}
	require.Equal(t, 2, meshComponents(len(combinedVertices), combinedTriangles))
}
