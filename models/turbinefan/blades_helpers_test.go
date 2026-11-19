package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// meshRing returns the vertices whose Y lies on the radial ring nearest r, and that Y.
func meshRing(verts []r3.Vec, r float64) (float64, []r3.Vec) {
	ring := math.Inf(1)
	for _, v := range verts {
		if math.Abs(v.Y-r) < math.Abs(ring-r) {
			ring = v.Y
		}
	}
	var out []r3.Vec
	for _, v := range verts {
		if math.Abs(v.Y-ring) <= 1e-6 {
			out = append(out, v)
		}
	}
	return ring, out
}

// farthest returns the vertex of ring farthest from the point, skipping those skip claims.
func farthest(ring []r3.Vec, from r3.Vec, skip func(r3.Vec) bool) r3.Vec {
	var best r3.Vec
	distance := -1.0
	for _, v := range ring {
		if skip != nil && skip(v) {
			continue
		}
		if d := v.Sub(from).Len(); d > distance {
			best, distance = v, d
		}
	}
	return best
}

// turnedBox returns the bounds of body's mesh, tessellated at tol, turned back by angle about +Z.
func turnedBox(t *testing.T, body *decad.Body, tol, angle float64) (lo, hi r3.Vec) {
	t.Helper()
	xf, err := r3.RotationAround(r3.Vec{}, r3.Vec{Z: 1}, units.Radians(angle))
	require.NoError(t, err)
	lo, hi = r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for _, v := range meshOf(t, body, tol).Vertices() {
		v = xf.Apply(v)
		lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y), Z: math.Min(lo.Z, v.Z)}
		hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y), Z: math.Max(hi.Z, v.Z)}
	}
	return lo, hi
}

// mesh builds a meshgap mesh of body's tessellation at tol, moved by d.
func gapMesh(t *testing.T, body *decad.Body, tol float64, d r3.Vec) meshgap.Mesh {
	t.Helper()
	mesh := meshOf(t, body, tol)
	vs := mesh.Vertices()
	moved := make([]r3.Vec, len(vs))
	for i, v := range vs {
		moved[i] = v.Add(d)
	}
	g, err := meshgap.New(moved, mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return g
}
