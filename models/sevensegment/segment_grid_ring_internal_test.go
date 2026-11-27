package sevensegment

// CI: isolated-tests

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentSegmentGridRingWalk checks that the ring-edge walk in nearestWithin returns, bit for bit, what the
// walk over every cell of each ring returns, on random loops of varied extent (many empty cells, so rings grow wide),
// with points inside, near and far outside the grid, and with finite and infinite limits.
func TestSevenSegmentSegmentGridRingWalk(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(7, 11))
	const maxReport = 5
	mismatches := 0
	checked := 0
	for trial := range 16 {
		extent := []float64{0.05, 1, 20, 300}[trial%4]
		var loops [][]r3.Vec
		for range 1 + rng.IntN(3) {
			n := 3 + rng.IntN(6)
			cx, cy := extent*rng.Float64(), extent*rng.Float64()
			l := make([]r3.Vec, n)
			for i := range l {
				l[i] = r3.Vec{X: cx + extent*0.3*(rng.Float64()-0.5), Y: cy + extent*0.3*(rng.Float64()-0.5)}
			}
			loops = append(loops, l)
		}
		// A far point keeps the box large so the grid has mostly empty cells.
		loops = append(loops, []r3.Vec{{X: 0, Y: 0}, {X: extent, Y: extent}})
		g := newSegmentGrid(loops)
		for range 100 {
			p := r3.Vec{X: extent * (-0.5 + 2*rng.Float64()), Y: extent * (-0.5 + 2*rng.Float64())}
			for _, limit := range []float64{math.Inf(1), 0, extent * 0.01, extent * 0.2, extent * 3} {
				want := g.nearestWithinWalk(p, limit)
				got := g.nearestWithin(p, limit)
				checked++
				if math.Float64bits(want) == math.Float64bits(got) {
					continue
				}
				mismatches++
				if mismatches <= maxReport {
					t.Logf("trial %d point %v limit %g: want %g, got %g", trial, p, limit, want, got)
				}
			}
		}
	}
	require.Zero(t, mismatches, "mismatches out of %d checks (first %d logged)", checked, maxReport)
}
