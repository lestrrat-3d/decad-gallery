package cycloidal

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// Both ends of a lobe lie on the root circle, and the lobes repeat every 2π/L.
func TestLobeEndsLieOnTheRootCircle(t *testing.T) {
	p := DefaultParams()
	d := resolve(p)
	ts := lobeParameters(p, d, lobeTurnLimits[0])
	require.Greater(t, len(ts), 20)
	require.Less(t, len(ts), 80)
	first := discPoint(p, d, ts[0], 0, 0, 0)
	last := discPoint(p, d, ts[len(ts)-1], 0, 0, 0)
	require.InDelta(t, d.rootRadius, math.Hypot(first.x, first.y), 1e-9)
	require.InDelta(t, d.rootRadius, math.Hypot(last.x, last.y), 1e-9)
	// The tip of the lobe stays inside the pin circle.
	reach := 0.0
	for _, tt := range ts {
		q := discPoint(p, d, tt, 0, 0, 0)
		reach = math.Max(reach, math.Hypot(q.x, q.y))
	}
	require.Greater(t, reach, d.rootRadius)
	require.Less(t, reach, d.pinCircleRadius)
	// The next lobe starts where this one ends, rotated by one lobe pitch.
	span := 2 * math.Pi / float64(d.lobes)
	next := discPoint(p, d, span, 0, 0, 0)
	require.InDelta(t, last.x, next.x, 1e-9)
	require.InDelta(t, last.y, next.y, 1e-9)
}
