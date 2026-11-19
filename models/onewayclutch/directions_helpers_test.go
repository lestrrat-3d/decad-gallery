package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// dirCase is one preset in one direction.
type dirCase struct {
	name   string
	params onewayclutch.Params
}

// dirCases lists every preset in both directions.
func dirCases() []dirCase {
	var out []dirCase
	for _, shaft := range onewayclutch.Shafts() {
		for _, dir := range []onewayclutch.Direction{onewayclutch.Clockwise, onewayclutch.CounterClockwise} {
			p := onewayclutch.DefaultParams()
			p.Shaft, p.Freewheel = shaft.Name, dir
			out = append(out, dirCase{name: shaft.Name + " " + dir.String(), params: p})
		}
	}
	return out
}

// wideBoreCases are the 8 mm presets in both directions at the largest BoreClearance, 0.5 mm.
func wideBoreCases() []dirCase {
	var out []dirCase
	for _, name := range []string{"608ZZ", "608ZZ-gearbox"} {
		for _, dir := range []onewayclutch.Direction{onewayclutch.Clockwise, onewayclutch.CounterClockwise} {
			p := onewayclutch.DefaultParams()
			p.Shaft, p.Freewheel, p.BoreClearance = name, dir, 0.5
			out = append(out, dirCase{name: name + " " + dir.String() + " bore 0.5", params: p})
		}
	}
	return out
}

// boxOf returns a body's bounds, or fails the test.
func boxOf(t *testing.T, b *decad.Body, name string) decad.Box {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err, name)
	return box
}

func volumeOf(t *testing.T, b *decad.Body, name string) (float64, float64) {
	t.Helper()
	v, err := b.Volume()
	require.NoError(t, err, name)
	return v.Value.Base(), v.Bound.Base()
}

// meshOf tessellates a body into a meshgap mesh.
func meshOf(t *testing.T, b *decad.Body, name string) meshgap.Mesh {
	t.Helper()
	mesh, err := b.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err, name)
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err, name)
	return mg
}

// tessellate returns a body's mesh.
func tessellate(t *testing.T, b *decad.Body, name string) *decad.Mesh {
	t.Helper()
	mesh, err := b.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err, name)
	return mesh
}

// countInside counts the points, taken at a radius past the tooth tips and moved to mid-band, that lie inside the closed
// mesh. A point is inside
// when a ray along +X crosses the mesh an odd number of times.
func countInside(points []r3.Vec, mesh *decad.Mesh, toothTip float64) int {
	const midBand = (onewayclutch.BandFrom + onewayclutch.BandTo) / 2
	vertices, triangles := mesh.Vertices(), mesh.Triangles()
	count := 0
	for _, p := range points {
		if math.Hypot(p.X, p.Y) < toothTip+0.05 {
			continue
		}
		// The hub is a prism taller than the tooth band, so its vertices lie above and below the teeth: query at mid-band.
		p.Z = midBand
		crossings := 0
		for _, tri := range triangles {
			a, b, c := vertices[tri[0]], vertices[tri[1]], vertices[tri[2]]
			// Project on the YZ plane: the ray is a point (p.Y, p.Z) moving to +X.
			d1 := edgeSide(a, b, p)
			d2 := edgeSide(b, c, p)
			d3 := edgeSide(c, a, p)
			if (d1 < 0 || d2 < 0 || d3 < 0) && (d1 > 0 || d2 > 0 || d3 > 0) {
				continue
			}
			area := d1 + d2 + d3
			if area == 0 {
				continue
			}
			// x of the triangle's plane at (p.Y, p.Z), by barycentric weights.
			x := (d2*a.X + d3*b.X + d1*c.X) / area
			if x > p.X {
				crossings++
			}
		}
		if crossings%2 == 1 {
			count++
		}
	}
	return count
}

// edgeSide is the signed area of (a, b, p) in the YZ plane.
func edgeSide(a, b, p r3.Vec) float64 {
	return (b.Y-a.Y)*(p.Z-a.Z) - (b.Z-a.Z)*(p.Y-a.Y)
}
