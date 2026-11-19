package spurgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// pitchCrossings returns the angles at which the tooth section's outline crosses the pitch circle,
// found by interpolating its flank polylines.
func pitchCrossings(t *testing.T, p spurgear.Params) []float64 {
	t.Helper()
	world := sketch.NewWorld()
	_, profile, err := spurgear.BuildToothSection(t.Context(), world, world.XY(), p, 0)
	require.NoError(t, err)
	pitch := p.Module * float64(p.Teeth) / 2
	var angles []float64
	for _, edge := range profile.Outer {
		for i := 1; i < len(edge.Polyline); i++ {
			a, b := edge.Polyline[i-1], edge.Polyline[i]
			ra, rb := math.Hypot(a[0], a[1]), math.Hypot(b[0], b[1])
			if (ra-pitch)*(rb-pitch) >= 0 || ra == rb {
				continue
			}
			f := (pitch - ra) / (rb - ra)
			angles = append(angles, math.Atan2(a[1]+f*(b[1]-a[1]), a[0]+f*(b[0]-a[0])))
		}
	}
	return angles
}
