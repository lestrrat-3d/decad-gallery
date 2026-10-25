package cycloidal

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// A ring of fit-spline repeats is drawn from one repeat's points. The pattern shares every junction, so the
// sketch is fully constrained and its one region is bounded by exactly the splines of all repeats.
func TestPatternedLoopClosesOnSharedPoints(t *testing.T) {
	const repeats = 12
	span := 2 * math.Pi / repeats
	ring := func(a, r float64) point { return point{r * math.Cos(a), r * math.Sin(a)} }
	// Two splines per repeat: valley to peak to the next valley.
	repeat := [][]point{
		{ring(0, 6), ring(span/6, 7), ring(span/3, 8.5), ring(span/2, 9)},
		{ring(span/2, 9), ring(2*span/3, 8.5), ring(5*span/6, 7), ring(span, 6)},
	}
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	require.NoError(t, err)
	require.NoError(t, drawPatternedLoop(s, s.Origin(), repeat, repeats))
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))

	profiles := s.Profiles()
	require.Len(t, profiles, 1)
	require.True(t, profiles[0].Valid)
	require.Len(t, profiles[0].Outer, 2*repeats)
	for _, edge := range profiles[0].Outer {
		require.True(t, edge.TExact, "every junction is an exact shared endpoint")
	}
}
