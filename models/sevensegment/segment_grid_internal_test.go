package sevensegment

// CI: isolated-tests

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentSegmentGrid checks the grid against nearest on a cam section, the loops loopsGap reads most, at
// points spread over and around it: the distances agree bit for bit, and a limit only hides distances beyond it.
func TestSevenSegmentSegmentGrid(t *testing.T) {
	t.Parallel()
	r, cam := sharedCam(t, SegA)
	loops := sectionAt(t, cam, r.dm.camDiscFace(SegA)+1.0)
	g := newSegmentGrid(loops)
	rng := rand.New(rand.NewPCG(1, 2))
	pts := densify(loops, 0.37)
	for range 3000 {
		pts = append(pts, r3.Vec{X: -30 + 60*rng.Float64(), Y: -30 + 60*rng.Float64()})
	}
	for i, p := range pts {
		want := nearest(p, loops)
		require.Equal(t, math.Float64bits(want), math.Float64bits(g.nearestWithin(p, math.Inf(1))), "point %d", i)
		for _, limit := range []float64{0.01, 0.3, 2} {
			got := g.nearestWithin(p, limit)
			if want <= limit {
				require.Equal(t, math.Float64bits(want), math.Float64bits(got), "point %d limit %.2f", i, limit)
			} else {
				require.Greater(t, got, limit, "point %d limit %.2f", i, limit)
			}
		}
	}
}
