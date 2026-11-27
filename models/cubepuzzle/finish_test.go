package cubepuzzle_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPrintedPartsHaveNoLargePlanarFace(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	for _, part := range []struct {
		name string
		body *decad.Body
	}{
		{"hub", a.Hub}, {"center", a.Centers[4]}, {"edge", a.Edges[3]}, {"corner", a.Corners[7]}, {"disk", a.Disks[4]},
	} {
		mesh, err := part.body.Tessellate(t.Context(), units.Millimeters(.2), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, part.name)
		vertices, faces, triangles := mesh.Vertices(), mesh.SourceFaces(), mesh.Triangles()
		require.Equal(t, len(triangles), len(faces), "%s mesh source count", part.name)
		type planeBox struct {
			u, v                   r3.Vec
			minU, maxU, minV, maxV float64
		}
		planes := make(map[[4]int64]*planeBox)
		for i, tri := range triangles {
			require.NotNil(t, faces[i], "%s triangle %d has no source face", part.name, i)
			plane, ok := faces[i].Surface().(decad.Plane)
			if !ok {
				continue
			}
			n := plane.Frame.N()
			if n.X < -1e-9 || (math.Abs(n.X) < 1e-9 && n.Y < -1e-9) || (math.Abs(n.X) < 1e-9 && math.Abs(n.Y) < 1e-9 && n.Z < 0) {
				n = n.Scale(-1)
			}
			d := n.Dot(plane.Frame.Origin())
			key := [4]int64{int64(math.Round(n.X * 1e6)), int64(math.Round(n.Y * 1e6)), int64(math.Round(n.Z * 1e6)), int64(math.Round(d * 1e6))}
			box := planes[key]
			if box == nil {
				box = &planeBox{u: plane.Frame.U(), v: plane.Frame.V(), minU: math.Inf(1), maxU: math.Inf(-1), minV: math.Inf(1), maxV: math.Inf(-1)}
				planes[key] = box
			}
			for _, index := range tri {
				p := vertices[index]
				u, v := p.Dot(box.u), p.Dot(box.v)
				box.minU = min(box.minU, u)
				box.maxU = max(box.maxU, u)
				box.minV = min(box.minV, v)
				box.maxV = max(box.maxV, v)
			}
		}
		require.NotEmpty(t, planes, "%s has no planar mesh sources", part.name)
		for key, box := range planes {
			bound := mesh.Bound().Base()
			upper := (box.maxU - box.minU + 2*bound) * (box.maxV - box.minV + 2*bound)
			require.Less(t, upper, 400.0, "%s plane %v bounding rectangle", part.name, key)
		}
	}
}
