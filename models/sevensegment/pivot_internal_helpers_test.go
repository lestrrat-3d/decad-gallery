package sevensegment

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// pivotSection cuts body with the plane normal to segment seg's axis at u from its middle and returns its loops as
// (s, z - RotorAxisZ): the horizontal coordinate across the rotor and the height above the modeled axis.
func pivotSection(t *testing.T, body *decad.Body, seg Segment, u float64) [][]r3.Vec {
	t.Helper()
	f := rotorFrame(seg)
	origin := f.Origin().Add(f.N().Scale(u))
	return planeSection(t, body, origin, f.U(), r3.Vec{Z: 1})
}

// scanEdge returns the coordinate along one axis of a section at which the material of loops starts or ends, found by
// bisection between a point in the void (void) and a point in the material (solid), each given as the coordinate along
// the axis; at returns the section point for a coordinate.
func scanEdge(loops [][]r3.Vec, at func(c float64) r3.Vec, void, solid float64) float64 {
	lo, hi := void, solid
	for i := 0; i < 40; i++ {
		m := (lo + hi) / 2
		if insideLoops(loops, at(m)) == insideLoops(loops, at(solid)) {
			hi = m
		} else {
			lo = m
		}
	}
	return (lo + hi) / 2
}

// placedPivotBlock returns segment seg's block at the +u (end 1) or -u (end 0) end, in the world at turn 0.
func placedPivotBlock(t *testing.T, r *run, seg Segment, end int) *decad.Body {
	t.Helper()
	proto, err := r.pivotBlock()
	require.NoError(t, err)
	sgn := float64(2*end - 1)
	b, err := r.toSegment(seg, sgn, proto)
	require.NoError(t, err)
	return b
}

// pivotMesh tessellates body to tolerance tol (mm).
func pivotMesh(t *testing.T, body *decad.Body, tol float64) *decad.Mesh {
	t.Helper()
	m, err := body.Tessellate(t.Context(), units.Millimeters(tol))
	require.NoError(t, err)
	return m
}

// pivotMeshGap returns the lower bound of the distance between the triangles of a that keep accepts (all of them when
// keep is nil) and the triangles of b moved by shift. keep takes a triangle's centre.
func pivotMeshGap(t *testing.T, a *decad.Mesh, keep func(c r3.Vec) bool, b *decad.Mesh, shift r3.Vec) float64 {
	t.Helper()
	var faces [][3]int
	vs := a.Vertices()
	for _, tri := range a.Triangles() {
		c := vs[tri[0]].Add(vs[tri[1]]).Add(vs[tri[2]]).Scale(1.0 / 3)
		if keep == nil || keep(c) {
			faces = append(faces, tri)
		}
	}
	ma, err := meshgap.New(vs, faces, a.Bound().Base())
	require.NoError(t, err)
	bv := b.Vertices()
	moved := make([]r3.Vec, len(bv))
	for i, v := range bv {
		moved[i] = v.Add(shift)
	}
	mb, err := meshgap.New(moved, b.Triangles(), b.Bound().Base())
	require.NoError(t, err)
	g, _, err := ma.Exceeds(mb, 0)
	require.NoError(t, err)
	return g + ma.Bound() + mb.Bound()
}
