package sevensegment

import (
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func buttonIndexMesh(t *testing.T, body *decad.Body) meshgap.Mesh {
	t.Helper()
	m, err := body.Tessellate(t.Context(), units.Millimeters(.005), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	result, err := meshgap.New(m.Vertices(), m.Triangles(), m.Bound().Base())
	require.NoError(t, err)
	return result
}

func buttonIndexGroove(t *testing.T, data buttonIndexMeshData, transform r3.Transform, y float64) (float64, float64) {
	t.Helper()
	var zs []float64
	for _, face := range data.triangles {
		a := transform.Apply(data.vertices[face[0]])
		b := transform.Apply(data.vertices[face[1]])
		c := transform.Apply(data.vertices[face[2]])
		denom := (b.Y-c.Y)*(a.X-c.X) + (c.X-b.X)*(a.Y-c.Y)
		if math.Abs(denom) < 1e-12 {
			continue
		}
		u := ((b.Y-c.Y)*(-c.X) + (c.X-b.X)*(y-c.Y)) / denom
		v := ((c.Y-a.Y)*(-c.X) + (a.X-c.X)*(y-c.Y)) / denom
		if u < -1e-9 || v < -1e-9 || u+v > 1+1e-9 {
			continue
		}
		z := u*a.Z + v*b.Z + (1-u-v)*c.Z
		if z > 43.5 && z < 51.5 {
			zs = append(zs, z)
		}
	}
	slices.Sort(zs)
	var unique []float64
	for _, z := range zs {
		if len(unique) == 0 || z-unique[len(unique)-1] > .02 {
			unique = append(unique, z)
		}
	}
	require.Len(t, unique, 2, "groove wall intersections")
	return unique[0], unique[1]
}

func buttonIndexInside(data buttonIndexMeshData, transform r3.Transform, p r3.Vec) bool {
	// A skew ray avoids the axial and radial directions used by the wheel.
	d := r3.Vec{X: 1, Y: .137, Z: .233}
	var hits []float64
	for _, face := range data.triangles {
		a := transform.Apply(data.vertices[face[0]])
		b := transform.Apply(data.vertices[face[1]])
		c := transform.Apply(data.vertices[face[2]])
		e1, e2 := b.Sub(a), c.Sub(a)
		h := d.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < 1e-12 {
			continue
		}
		f := 1 / det
		s := p.Sub(a)
		u := f * s.Dot(h)
		if u < 0 || u > 1 {
			continue
		}
		q := s.Cross(e1)
		v := f * d.Dot(q)
		if v < 0 || u+v > 1 {
			continue
		}
		if distance := f * e2.Dot(q); distance > 1e-9 {
			hits = append(hits, distance)
		}
	}
	slices.Sort(hits)
	count := 0
	last := math.Inf(-1)
	for _, hit := range hits {
		if hit-last > 1e-7 {
			count++
			last = hit
		}
	}
	return count%2 == 1
}

type buttonIndexMeshData struct {
	vertices  []r3.Vec
	triangles [][3]int
	bound     float64
}

func buttonIndexData(t *testing.T, body *decad.Body) buttonIndexMeshData {
	return buttonIndexDataAt(t, body, .02)
}

func buttonIndexDataAt(t *testing.T, body *decad.Body, tolerance float64) buttonIndexMeshData {
	t.Helper()
	m, err := body.Tessellate(t.Context(), units.Millimeters(tolerance), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	return buttonIndexMeshData{m.Vertices(), m.Triangles(), m.Bound().Base()}
}

func (data buttonIndexMeshData) moved(t *testing.T, transform r3.Transform) meshgap.Mesh {
	t.Helper()
	vertices := make([]r3.Vec, len(data.vertices))
	for i, p := range data.vertices {
		vertices[i] = transform.Apply(p)
	}
	m, err := meshgap.New(vertices, data.triangles, data.bound)
	require.NoError(t, err)
	return m
}
