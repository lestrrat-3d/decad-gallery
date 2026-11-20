package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file holds the pin-in-fork and guard checker of instructions.md "Geometry checks" (TestForkAction): the
// sections of the real Lever, Roller Table and Safety Roller, posed by LeverAt and the balance rotation, are compared
// as polygons at each sampled balance angle.

// TestForkAction sweeps the balance over -60 to 60 degrees on both swings and checks, on sections of the real bodies,
// that the pin never overlaps the lever by more than the section bound, touches a corner round on [θ1, θ3] and
// [-θ3, -θ1] only, and that the fork and guard clearances F1, F2, F4 and G1 to G3 hold.
func TestForkAction(t *testing.T) {
	b := partBuilder(t)
	g := &b.g
	f := newForkBodies(t, b)
	bound := 0.02 + f.off

	// F1: the pin to each corner round with the lever centred, and F2: the pin to the notch bottom.
	_, _, pin0, _, _ := f.pose(g, 0, 0)
	lf0, _, _, _, _ := f.pose(g, 0, 0)
	require.InDelta(t, SlotShake, loopsGap(pin0, lf0), bound, "F1 shake")
	q := Pt{g.B.X, g.B.Y - PinCentreRadius}
	bottom, ok := rayDown(lf0, q.X, q.Y)
	require.True(t, ok)
	require.GreaterOrEqual(t, q.Y-bottom-PinRadius, 0.75-bound, "F2 notch bottom")

	// F4: at each banking the round farther from the pin path stands 1.30 or more from it.
	for _, sign := range []float64{-1, 1} {
		lever, _, _, _, _ := f.pose(g, sign*g.Banking, 0)
		path := make([]outline, 1)
		for i := 0; i < 720; i++ {
			a := float64(i) * 0.5 * deg
			path[0] = append(path[0], Pt{g.B.X, g.B.Y}.Add(Polar(PinCentreRadius, a)))
		}
		gap := 0.0
		for _, kc := range []Pt{g.KR, g.KL} {
			c := kc.Rot(-sign * g.Banking).Add(Pt{g.P.X, g.P.Y}) // Layout gives K in the lever frame about P
			near := math.Inf(1)
			for _, l := range lever {
				for _, v := range l {
					if v.Sub(c).Len() > g.CornerRadius+bound {
						continue
					}
					for _, p := range path {
						near = math.Min(near, segDistLoop(v, p))
					}
				}
			}
			gap = math.Max(gap, near-PinRadius)
		}
		require.GreaterOrEqual(t, gap, 1.30-bound, "F4 at banking %+.0f", sign)
	}

	step := forkStep
	for _, plus := range []bool{true, false} {
		lo, hi := g.Theta1, g.Theta3
		if !plus {
			lo, hi = -g.Theta3, -g.Theta1
		}
		for theta := -forkSweep; theta <= forkSweep; theta += step {
			psi, _, err := g.LeverAt(theta, plus)
			require.NoError(t, err)
			lvF, lvS, pin, rolF, rolS := f.pose(g, psi, theta)
			require.LessOrEqual(t, loopsDepth(pin, lvF), bound, "pin in the lever at θ %.3f°", theta/deg)
			gap := loopsGap(pin, lvF)
			switch {
			case theta >= lo+forkMargin && theta <= hi-forkMargin:
				require.LessOrEqual(t, gap, bound, "pin touches a round at θ %.3f°", theta/deg)
			case theta < lo-forkMargin || theta > hi+forkMargin:
				require.Greater(t, gap, bound, "pin is free at θ %.3f°", theta/deg)
			}
			// G2 and G3: the dart in the crescent and the horn tips round the safety roller, over the swing.
			require.False(t, loopsGap(lvS, rolS) < 0.25-bound, "G2 at θ %.3f°", theta/deg)
			require.GreaterOrEqual(t, loopsGap(lvF, rolF), 1.9-bound, "G3 at θ %.3f°", theta/deg)
			// G1: the dart to the safety roller at the banking.
			if math.Abs(math.Abs(psi)-g.Banking) < 1e-9 {
				require.GreaterOrEqual(t, loopsGap(lvS, rolS), 0.95-bound, "G1 at θ %.3f°", theta/deg)
			}
		}
	}
}

// TestForkHelpers checks the sweep's polygon helpers on a square and a disc.
func TestForkHelpers(t *testing.T) {
	square := []outline{{{0, 0}, {10, 0}, {10, 10}, {0, 10}}}
	disc := outline{}
	for i := 0; i < 360; i++ {
		disc = append(disc, Pt{14, 5}.Add(Polar(2, float64(i)*deg)))
	}
	require.InDelta(t, 2.0, loopsGap(square, []outline{disc}), 1e-3)
	require.Zero(t, loopsDepth([]outline{disc}, square))
	moved := movedBy([]outline{disc}, Pt{-5, 0})
	require.InDelta(t, 3.0, loopsDepth(moved, square), 1e-3)
	y, ok := rayDown(square, 5, 20)
	require.True(t, ok)
	require.InDelta(t, 10.0, y, 1e-12)
	y, ok = rayDown(square, 5, 5)
	require.True(t, ok)
	require.InDelta(t, 0.0, y, 1e-12)
	_, ok = rayDown(square, 12, 20)
	require.False(t, ok)
}
