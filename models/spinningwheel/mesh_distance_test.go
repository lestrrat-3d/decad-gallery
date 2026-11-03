package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
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
			_, clear := ring.meshSeparation(makeShape(c.inner, c.outer, c.width, c.z))
			require.True(t, clear == c.clear, "clear=%t, want %t", clear, c.clear)
		})
	}
}

// Boundary separation alone does not exclude containment. Test vertices of every possible component as well.
func (s collisionShape) meshSeparation(other collisionShape) (float64, bool) {
	return s.meshSeparationBy(other, 0)
}

// meshSeparationBy is meshSeparation requiring the boundary gap to exceed limit millimetres after both bounds.
func (s collisionShape) meshSeparationBy(other collisionShape, limit float64) (float64, bool) {
	a, err := meshgap.New(s.points, s.triangles, s.bound)
	if err != nil {
		return 0, false
	}
	b, err := meshgap.New(other.points, other.triangles, other.bound)
	if err != nil {
		return 0, false
	}
	gap, clear, err := a.Exceeds(b, limit)
	if err != nil || !clear {
		return gap, false
	}
	for _, p := range s.points {
		if meshContains(other, p) {
			return 0, false
		}
	}
	for _, p := range other.points {
		if meshContains(s, p) {
			return 0, false
		}
	}
	return gap, true
}

func meshContains(s collisionShape, p r3.Vec) bool {
	angle := 0.0
	for _, f := range s.triangles {
		a, b, c := s.points[f[0]].Sub(p), s.points[f[1]].Sub(p), s.points[f[2]].Sub(p)
		la, lb, lc := a.Len(), b.Len(), c.Len()
		denom := la*lb*lc + a.Dot(b)*lc + b.Dot(c)*la + c.Dot(a)*lb
		angle += 2 * math.Atan2(a.Dot(b.Cross(c)), denom)
	}
	return math.Abs(angle) > 2*math.Pi
}
