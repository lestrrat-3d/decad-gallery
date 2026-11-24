package sevensegment_test

// CI: isolated-tests

import (
	"math/rand/v2"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentCrossingGrid checks the grid against pointIn on the top plate, the busiest mesh, at points spread over
// and around its box and at points just off its vertices.
func TestSevenSegmentCrossingGrid(t *testing.T) {
	t.Parallel()
	mesh, _ := meshOf(t, builtDigit0(t).TopPlate, 0.05)
	g := newCrossingGrid(mesh)
	rng := rand.New(rand.NewPCG(3, 4))
	var pts []r3.Vec
	for range 1500 {
		pts = append(pts, r3.Vec{X: g.lo.X - 1 + (g.hi.X-g.lo.X+2)*rng.Float64(), Y: g.lo.Y - 1 + (g.hi.Y-g.lo.Y+2)*rng.Float64(),
			Z: g.lo.Z - 1 + (g.hi.Z-g.lo.Z+2)*rng.Float64()})
	}
	vs := mesh.Vertices()
	for range 1500 {
		v := vs[rng.IntN(len(vs))]
		pts = append(pts, v.Add(r3.Vec{X: 0.1 * (rng.Float64() - 0.5), Y: 0.1 * (rng.Float64() - 0.5), Z: 0.1 * (rng.Float64() - 0.5)}))
	}
	inside := 0
	for i, p := range pts {
		want := pointIn(p, mesh)
		require.Equal(t, want, g.pointIn(p), "point %d", i)
		if want {
			inside++
		}
	}
	require.Positive(t, inside, "some points lie inside the plate")
}
