package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// rotate turns v about the axis through the origin along unit vector a by angle, by Rodrigues' formula.
func rotate(v, a r3.Vec, angle float64) r3.Vec {
	sin, cos := math.Sincos(angle)
	return v.Scale(cos).Add(a.Cross(v).Scale(sin)).Add(a.Scale(a.Dot(v) * (1 - cos)))
}

// regionMesh returns the triangles of body's mesh that keep(centroid) accepts, turned by angle about axis, as a gap
// mesh that carries the mesh's displacement bound.
func regionMesh(t *testing.T, mesh *decad.Mesh, axis r3.Vec, angle float64, keep func(c r3.Vec) bool) meshgap.Mesh {
	t.Helper()
	vertices := mesh.Vertices()
	var faces [][3]int
	for _, f := range mesh.Triangles() {
		c := vertices[f[0]].Add(vertices[f[1]]).Add(vertices[f[2]]).Scale(1.0 / 3)
		if keep(c) {
			faces = append(faces, f)
		}
	}
	require.NotEmpty(t, faces)
	turned := make([]r3.Vec, len(vertices))
	for i, v := range vertices {
		turned[i] = rotate(v, axis, angle)
	}
	m, err := meshgap.New(turned, faces, mesh.Bound().Base())
	require.NoError(t, err)
	return m
}
