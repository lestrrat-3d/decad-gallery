package sevensegment_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentBorePlay pins the D-bore play the determinacy tests use: each double-D bore lets its part turn 1.71
// degrees either way (a single-D bore of the same fit turns 3.51, because the part shifts away from its flat), so a cam
// can lead or lag the ratchet by 3.42.
func TestSevenSegmentBorePlay(t *testing.T) {
	cam := borePlay(sevensegment.ShaftCamRadius, sevensegment.ShaftCamFlat, sevensegment.CamBoreRadius, sevensegment.CamBoreFlat)
	t.Logf("cam bore play %.3f deg, phase swing %.3f deg", cam, phaseSwing())
	require.InDelta(t, 1.71, cam, 0.02)
	require.InDelta(t, 3.42, phaseSwing(), 0.04)
}
