package sevensegment_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentRestAngles proves, from Dimensions and the built loads, that a released knob leaves every cam on its
// dwell: over one step, with each cam turned against the ratchet wheel by the D-bores' play as the drive sets it
// (phaseRest), every angle where a cam is more than 0.05 off its dwell is one the detent drives the shaft out of, at
// friction 0.35 with a margin of 1.8 while the pawls drive back. At friction 0.45 the shaft can rest only where every cam
// is within 0.07 of its dwell (computed 0.061, while the pawls carry the shaft past the crest).
func TestSevenSegmentRestAngles(t *testing.T) {
	startDigit0()
	t.Parallel()
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	swing := phaseSwing()
	for _, c := range []struct{ mu, maxOff, minRelease float64 }{{0.35, 0.05, 1.8}, {0.45, 0.07, 1.0}} {
		off, release := phaseRest(t, dm, c.mu, swing)
		t.Logf("friction %.2f: worst cam offset at a rest %.3f, least release budget %.3f", c.mu, off, release)
		require.LessOrEqual(t, off, c.maxOff, "friction %.2f: a cam rests off its dwell", c.mu)
		require.GreaterOrEqual(t, release, c.minRelease, "friction %.2f: the pawls drive the shaft back without margin", c.mu)
	}
}
