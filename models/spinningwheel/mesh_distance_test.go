package spinningwheel_test

import (
	"math"
	"sync"
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

// Boundary separation alone does not exclude containment. Test vertices of every possible component as well.
func (s collisionShape) meshSeparation(other collisionShape) (float64, bool) {
	return prepareShape(s).separation(prepareShape(other))
}

// meshSeparationBy is meshSeparation requiring the boundary gap to exceed limit millimetres after both bounds.
func (s collisionShape) meshSeparationBy(other collisionShape, limit float64) (float64, bool) {
	return prepareShape(s).separationBy(prepareShape(other), limit)
}

// preparedShape is a collision shape whose meshgap.Mesh is built on first use and then reused.
// sync.OnceValues makes the build safe when several goroutines share one shape, and copies share the result.
type preparedShape struct {
	collisionShape
	mesh func() (meshgap.Mesh, error)
}

func prepareShape(s collisionShape) preparedShape {
	return preparedShape{collisionShape: s, mesh: sync.OnceValues(func() (meshgap.Mesh, error) {
		return meshgap.New(s.points, s.triangles, s.bound)
	})}
}

func (s preparedShape) separation(other preparedShape) (float64, bool) {
	a, err := s.mesh()
	if err != nil {
		return 0, false
	}
	b, err := other.mesh()
	if err != nil {
		return 0, false
	}
	clear, err := a.ExceedsClearance(b, 0)
	if err != nil {
		return 0, false
	}
	gap := 0.0
	if !clear {
		gap, _, err = a.Exceeds(b, 0)
		if err != nil {
			return 0, false
		}
		return gap, false
	}
	if s.contains(other) {
		return 0, false
	}
	return 0, true
}

// separationBy is meshSeparationBy on shapes whose meshes may already be built.
func (s preparedShape) separationBy(other preparedShape, limit float64) (float64, bool) {
	a, err := s.mesh()
	if err != nil {
		return 0, false
	}
	b, err := other.mesh()
	if err != nil {
		return 0, false
	}
	gap, clear, err := a.Exceeds(b, limit)
	if err != nil || !clear {
		return gap, false
	}
	// The surface check proved the meshes apart by a margin, so no connected component crosses the other.
	// Each component is then wholly inside or wholly outside, and one vertex per component decides it.
	if s.contains(other) {
		return 0, false
	}
	return gap, true
}

// contains checks one vertex per component after the surface check proves that no component crosses either mesh.
func (s preparedShape) contains(other preparedShape) bool {
	for _, p := range componentRepresentatives(s.collisionShape) {
		if meshContains(other.collisionShape, p) {
			return true
		}
	}
	for _, p := range componentRepresentatives(other.collisionShape) {
		if meshContains(s.collisionShape, p) {
			return true
		}
	}
	return false
}

// componentRepresentatives returns one vertex of each connected component of the triangle mesh.
// Vertices that no triangle uses are their own components, so none is skipped.
func componentRepresentatives(s collisionShape) []r3.Vec {
	parent := make([]int, len(s.points))
	for i := range parent {
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for _, f := range s.triangles {
		parent[root(f[1])] = root(f[0])
		parent[root(f[2])] = root(f[0])
	}
	seen := make(map[int]struct{})
	var reps []r3.Vec
	for i, p := range s.points {
		r := root(i)
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		reps = append(reps, p)
	}
	return reps
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
