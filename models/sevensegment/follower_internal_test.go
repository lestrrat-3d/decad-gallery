package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSevenSegmentFollowerDrive checks that no slider locks on its cam at the nominal friction, with the rigid-slider
// model of instructions.md "Cam follower drive".
func TestSevenSegmentFollowerDrive(t *testing.T) {
	require.InDelta(t, ShaftZ+OffRadius, sliderPinZ, 1e-9)
	dm, err := DefaultParams().Derive()
	require.NoError(t, err)

	nominal := followerSweep(dm, 0.35)
	harsh := followerSweep(dm, 0.5)
	lock := [2]float64{lockFriction(dm, 0), lockFriction(dm, 1)}
	t.Logf("D at 0.35: a/g/d %.3f, b-f %.3f", nominal.minRatio[0], nominal.minRatio[1])
	t.Logf("D at 0.5: a/g/d %.3f, b-f %.3f", harsh.minRatio[0], harsh.minRatio[1])
	t.Logf("lock friction: a/g/d %.3f, b-f %.3f", lock[0], lock[1])
	t.Logf("wall load at 0.35: a/g/d %.3f N, b-f %.3f N", nominal.maxWall[0], nominal.maxWall[1])
	press := nominal.maxTorque * math.Pi / 180 * PressTurn() / (ButtonStroke - RotorTravel)
	t.Logf("summed shaft torque at 0.35: %.1f N.mm (step %d to %d), press share %.1f N", nominal.maxTorque,
		nominal.step, nominal.step+1, press)

	require.GreaterOrEqual(t, nominal.minRatio[0], 1.40)
	require.GreaterOrEqual(t, nominal.minRatio[1], 1.17)
	require.GreaterOrEqual(t, lock[0], 0.42)
	require.GreaterOrEqual(t, lock[1], 0.38)
}
