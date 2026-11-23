package sevensegment_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// isToothMate reports whether a and b are two of the click indexer's parts whose teeth meet.
func isToothMate(a, b string) bool {
	pair := func(x, y string) bool { return a == x && b == y || a == y && b == x }
	return pair("rotor", "housing") || pair("rotor", "button")
}

// teethAround counts the teeth of a ring: the runs of material met going round the circle of radius r about the shaft
// axis at world Y y, sampled every 0.25 degrees.
func teethAround(t *testing.T, body *decad.Body, y, r float64) int {
	t.Helper()
	mesh, _ := meshOf(t, body, 0.1)
	g := newCrossingGrid(mesh)
	inside := func(deg float64) bool {
		a := deg * math.Pi / 180
		return g.pointIn(r3.Vec{X: r * math.Cos(a), Y: y, Z: sevensegment.ShaftZ + r*math.Sin(a)})
	}
	runs := 0
	prev := inside(359.75)
	for deg := 0.0; deg < 360; deg += 0.25 {
		cur := inside(deg)
		if cur && !prev {
			runs++
		}
		prev = cur
	}
	return runs
}

// movedMesh returns the mesh of body turned by turn degrees about the shaft axis and moved by drop along +Y.
func movedMesh(t *testing.T, body *decad.Body, tol, turn, drop float64) meshgap.Mesh {
	t.Helper()
	mesh, _ := meshOf(t, body, tol)
	rot, err := r3.RotationAround(r3.Vec{Z: sevensegment.ShaftZ}, r3.Vec{Y: 1}, units.Degrees(turn))
	require.NoError(t, err)
	src := mesh.Vertices()
	out := make([]r3.Vec, len(src))
	for i, v := range src {
		out[i] = rot.Apply(v).Add(r3.Vec{Y: drop})
	}
	mg, err := meshgap.New(out, mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// partTol is the chord tolerance a part's mesh is read at: the helicoid teeth of the rotor, button and housing tessellate no
// finer than 0.1 mm.
func partTol(name string) float64 {
	switch name {
	case "rotor", "button", "housing":
		return 0.1
	}
	return 0.05
}

// pressTol is the finer tolerance of the parts pressed on the shaft: the document's chain tolerance holds a mesh to 0.01 mm.
func pressTol(string) float64 { return 0.01 }

// isPivotPress reports whether a and b are a pivot block and the plate whose pocket holds it.
func isPivotPress(a, b string) bool {
	block := func(n string) bool { return strings.HasPrefix(n, "pivot block ") }
	plate := func(n string) bool { return n == "cradle plate" || n == "top plate" }
	return block(a) && plate(b) || block(b) && plate(a)
}
