package sevensegment_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentRestAngles proves, from Dimensions alone, that the camshaft cannot rest on a digit-less angle: at every
// angle psi past a digit's rest angle, either every segment sits within 0.05 mm of its dwell height all through the
// 1.5 degree timing allowance, or the detent drives the shaft (lift of the pawl above zero, drive at least 1.8 times
// the opposing torque of the flag mechanism, below the crest band). Where the shaft can rest, the heights are those of the nearest digit.
func TestSevenSegmentRestAngles(t *testing.T) {
	startDigit0()
	t.Parallel()
	dm, err := sevensegment.DefaultParams().Derive()
	require.NoError(t, err)
	restSteps, drivenSteps := 0, 0
	for psi := -1.4; psi < 34.6; psi += 0.05 {
		dwell := true
		for _, e := range []float64{-sevensegment.TimingAllowance, 0, sevensegment.TimingAllowance} {
			digit := int(math.Round((psi + e) / 36))
			for _, s := range sevensegment.Segments() {
				for k := range 10 {
					got := dm.Lift(s, 36*float64(k)+psi+e)
					// The digit this angle belongs to for digit index k is k + digit.
					lit, _ := sevensegment.Lit((k+digit+10)%10, s)
					w := 0.0
					if lit {
						w = sevensegment.Travel
					}
					if math.Abs(got-w) > 0.05 {
						dwell = false
					}
				}
			}
		}
		if dwell {
			restSteps++
			continue
		}
		delta, _ := dm.PawlDeflection(psi)
		require.Greater(t, delta, 0.0, "angle %.2f is neither a dwell nor driven: no pawl lift", psi)
		require.Less(t, psi, dm.CrestAngle-1.0, "angle %.2f is in the crest band", psi)
		require.GreaterOrEqual(t, worstFlagBudget(t, dm, psi), 1.8, "angle %.2f is not driven back with margin", psi)
		drivenSteps++
	}
	require.Greater(t, restSteps, 0)
	require.Greater(t, drivenSteps, 0)
}
