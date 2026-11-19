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

// These tests use the real bearing producer and verified meshes, including a hollow and a contained pair.
func TestSpinningWheelMeshSeparation(t *testing.T) {
	makeShape := func(inner, outer, width, z float64) collisionShape {
		f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		require.NoError(t, err)
		m, err := ballbearing.Build(t.Context(), decad.New(), ballbearing.Params{BoreDiameter: inner, OuterDiameter: outer, Width: width, Placement: &f})
		require.NoError(t, err)
		mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		return nativeCollisionShape(m.Body, surfaceTriangles{vertices: mesh.Vertices(), triangles: mesh.Triangles(), bound: mesh.Bound().Base()})
	}
	ring := makeShape(8, 22, 7, 0)
	for _, c := range []struct {
		name                   string
		inner, outer, width, z float64
		clear                  bool
	}{
		{"inside_open_bore", 2, 4, 1, 3, true},
		{"contained_in_material", 10, 12, 1, 3, false},
		{"crossing_wall", 4, 12, 1, 3, false},
		{"disjoint_axially", 8, 22, 1, 10, true},
		{"touching_caps", 8, 22, 1, 7, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			other := makeShape(c.inner, c.outer, c.width, c.z)
			gap, clear := ring.meshSeparation(other)
			require.True(t, clear == c.clear, "clear=%t, want %t", clear, c.clear)
			numericGap, numericClear := ring.meshSeparationBy(other, 0)
			require.Equal(t, numericClear, clear, "optimized and numeric clearance verdicts differ")
			if c.name == "contained_in_material" {
				require.Zero(t, gap, "containment refusal keeps the original zero gap")
				require.Zero(t, numericGap, "numeric containment refusal keeps the original zero gap")
			} else if !clear {
				require.InDelta(t, numericGap, gap, 1e-12, "optimized and numeric failure gaps differ")
			}
		})
	}
}
