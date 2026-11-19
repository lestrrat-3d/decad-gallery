package spinningwheel_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/stretchr/testify/require"
)

// The goal requires the connector to be rebuilt at 0.01 mm in 72 poses. This test covers that
// rebuild, its verification and its Sketches on their own. The pose is rigid, so the volume
// must stay constant within the held bounds across all 72 poses.
func TestSpinningWheelConnectorPoses(t *testing.T) {
	const poses = 72
	volumes := make([]float64, poses)
	bounds := make([]float64, poses)
	// Each pose is a parallel subtest with its own document. At most four run at once, which bounds
	// peak memory. The group subtest returns only after every pose has finished.
	slots := make(chan struct{}, 4)
	t.Run("poses", func(t *testing.T) {
		for i := range poses {
			t.Run(fmt.Sprintf("%d", i*5), func(t *testing.T) {
				t.Parallel()
				slots <- struct{}{}
				defer func() { <-slots }()
				p := spinningwheel.DefaultParams()
				p.WheelAngle = float64(i) * math.Pi / 36
				volumes[i], bounds[i] = requireConnectorPose(t, p)
			})
		}
	})
	if t.Failed() {
		return
	}
	for i := 1; i < poses; i++ {
		require.InDelta(t, volumes[0], volumes[i], bounds[0]+bounds[i]+1e-9, "connector volume at %d degrees", i*5)
	}
}

// Every scale and ratio changes the connector's size and pose, so its Sketches are
// verified on each branch too.
func TestSpinningWheelConnectorScales(t *testing.T) {
	for _, scale := range []float64{.8, 1, 1.2} {
		for _, ratio := range []int{7, 17} {
			p := spinningwheel.DefaultParams()
			p.Scale, p.Ratio = scale, ratio
			for _, angle := range []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
				p.WheelAngle = angle
				requireConnectorPose(t, p)
			}
		}
	}
}
