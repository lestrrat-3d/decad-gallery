package spinningwheel_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
)

// Start with the real producer and every real final solid before adding variants.
func TestSpinningWheelDefault(t *testing.T) {
	checkSpinningWheel(t, spinningwheel.DefaultParams(), true)
}

func TestSpinningWheelVariants(t *testing.T) {
	// requireLinkageFrameClearances poses the linkage at 72 angles and costs 150 to 290 s per
	// build, so it runs only where the linkage geometry differs. The linkage depends on Scale and
	// the brake travel, not on the ratio. The default build covers scale 1, small-seven covers scale
	// 0.8 and large-nine covers scale 1.2. TestSpinningWheelConnectorPoses and
	// TestSpinningWheelConnectorScales do not cover this: they check only the rebuilt connector
	// solid and its Sketches, never its clearance against the frame. Twelve and small-seventeen
	// repeat a scale already checked, so they skip it.
	for _, test := range []struct {
		name                        string
		linkage                     bool
		scale                       float64
		ratio                       int
		tension, brake, feed, angle float64
	}{
		{"small-seven", true, .8, 7, 0, 0, 0, math.Pi},
		{"large-nine", true, 1.2, 9, 12, 6, 200, -math.Pi / 2},
		{"twelve", false, 1, 12, 0, 6, 0, math.Pi / 7},
		{"small-seventeen", false, .8, 17, 12, 6, 200, math.Pi / 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := spinningwheel.DefaultParams()
			p.Scale, p.Ratio, p.DriveTension, p.BrakeTravel, p.YarnFeedPerTurn, p.WheelAngle = test.scale, test.ratio, test.tension, test.brake, test.feed, test.angle
			checkSpinningWheel(t, p, test.linkage)
		})
	}
}
