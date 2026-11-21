package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestButtonDetentPitchContinuity(t *testing.T) {
	const step = 1e-5
	for _, boundary := range []float64{0, 20, 67, 129, 205, 259, 260, 280, 354, 358, 360} {
		before, db := buttonDetentPitch(boundary - step)
		at, d := buttonDetentPitch(boundary)
		after, da := buttonDetentPitch(boundary + step)
		require.InDelta(t, before, after, 2e-6, "radius at %.0f", boundary)
		require.InDelta(t, db, da, 2e-5, "derivative at %.0f", boundary)
		require.InDelta(t, d, (at-before)/(step*math.Pi/180), 2e-5, "left derivative at %.0f", boundary)
		require.InDelta(t, d, (after-at)/(step*math.Pi/180), 2e-5, "right derivative at %.0f", boundary)
	}
	low, d := buttonDetentPitch(354)
	require.InDelta(t, 22, low, 1e-12)
	require.InDelta(t, 0, d, 1e-12)
	high, d := buttonDetentPitch(280)
	require.InDelta(t, 22+2.8*260*math.Pi/180, high, 1e-12)
	require.InDelta(t, 0, d, 1e-12)
}
