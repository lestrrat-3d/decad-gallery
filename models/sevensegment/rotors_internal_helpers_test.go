package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// rotorPoint returns the world point of the rotor-local point (a, b, u) of segment s, where a and b are the diagonal
// coordinates of the yellow and black faces.
func rotorPoint(s Segment, a, b, u float64) r3.Vec {
	f := rotorFrame(s)
	xr := (a - b) / math.Sqrt2
	yr := (a + b) / math.Sqrt2
	return f.Origin().Add(f.U().Scale(xr)).Add(f.V().Scale(yr)).Add(f.N().Scale(u))
}

// inBody reports whether p lies inside the closed mesh of b: a ray off the axes crosses it an odd number of times.
func inBody(t *testing.T, b *decad.Body, p r3.Vec) bool {
	t.Helper()
	var mesh *decad.Mesh
	var err error
	for _, tol := range []float64{0.01, 0.02, 0.04, 0.05} {
		if mesh, err = b.Tessellate(t.Context(), units.Millimeters(tol)); err == nil {
			break
		}
	}
	require.NoError(t, err)
	vs, ts := mesh.Vertices(), mesh.Triangles()
	dir := r3.Vec{X: 1, Y: 1.37e-4, Z: 2.71e-4}
	crossings := 0
	for _, tri := range ts {
		a, b2, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		e1, e2 := b2.Sub(a), c.Sub(a)
		h := dir.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < 1e-12 {
			continue
		}
		s := p.Sub(a)
		u := s.Dot(h) / det
		if u < 0 || u > 1 {
			continue
		}
		q := s.Cross(e1)
		v := dir.Dot(q) / det
		if v < 0 || u+v > 1 {
			continue
		}
		if e2.Dot(q)/det > 0 {
			crossings++
		}
	}
	return crossings%2 == 1
}
