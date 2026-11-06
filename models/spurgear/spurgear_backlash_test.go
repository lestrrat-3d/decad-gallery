package spurgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestDefaultBacklash(t *testing.T) {
	require.InDelta(t, 0.05, spurgear.DefaultParams().Backlash, 0)
}

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

// TestBacklashThinsTeeth checks that the tooth's pitch-circle edges sit at ±(π/(2N) - j/(2Rp)), so
// each tooth is j of arc thinner than the half-pitch tooth, and that the thinner gear has less volume.
func TestBacklashThinsTeeth(t *testing.T) {
	volumes := map[float64]float64{}
	bounds := map[float64]float64{}
	for _, j := range []float64{0, 0.2} {
		p := spurgear.DefaultParams()
		p.Thickness, p.Backlash = 2, j
		pitch := p.Module * float64(p.Teeth) / 2
		edge := math.Pi/(2*float64(p.Teeth)) - j/(2*pitch)

		angles := pitchCrossings(t, p)
		require.Len(t, angles, 2, "backlash %v", j)
		require.InDelta(t, edge, math.Max(angles[0], angles[1]), 2e-3, "backlash %v", j)
		require.InDelta(t, -edge, math.Min(angles[0], angles[1]), 2e-3, "backlash %v", j)

		gear, err := spurgear.Build(t.Context(), decad.New(), p)
		require.NoError(t, err)
		require.True(t, gear.IsSolid())
		volume, err := gear.Volume()
		require.NoError(t, err)
		volumes[j], bounds[j] = volume.Value.Base(), volume.Bound.Base()
	}
	require.Greater(t, volumes[0]-volumes[0.2], bounds[0]+bounds[0.2]+1)
}

func TestInvalidBacklashNamesBacklash(t *testing.T) {
	for name, j := range map[string]float64{
		"as thick as the module": 1,
		"negative thickening":    -1.5,
		"not a number":           math.NaN(),
		"infinite":               math.Inf(1),
	} {
		t.Run(name, func(t *testing.T) {
			p := spurgear.DefaultParams()
			p.Backlash = j
			doc := decad.New()
			_, err := spurgear.Build(t.Context(), doc, p)
			require.ErrorContains(t, err, "backlash")
			require.Empty(t, doc.Bodies())
		})
	}
}
