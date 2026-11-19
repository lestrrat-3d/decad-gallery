package escapement

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// This file holds the escapement checker: a quasi-static simulation of the wheel and anchor outlines as closed
// polylines. The anchor swings by ψ = -A cos 2πt; at each step the wheel advances as far as the outlines allow
// without overlapping, and backs off if the anchor has moved into it.

func TestActionAnalytic(t *testing.T) {
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	c := &checker{g: &g, in: actionInput{teeth: analyticToothPolys(), anchor: analyticAnchor(&g, 0.005)}}
	res, err := c.run()
	require.NoError(t, err)
	checkAction(t, &g, res, 0.005)
	// Numerical verification of instructions.md.
	for _, d := range res.drops {
		require.InDelta(t, 1.388, d.size*TipRadius, 0.05)
	}
	require.InDelta(t, 0.613, res.others, 0.02)
	require.InDelta(t, 1.947, res.rootGap, 0.02)
	t.Logf("advance %v drops %v lock %v lift %v others %.3f root %.3f top %.3f recoil %g", res.periodAdvance,
		res.drops, res.lock, res.lift, res.others, res.rootGap, res.topGap, res.maxRecoil)
}

func TestActionAnalyticWide(t *testing.T) {
	p := DefaultParams()
	p.Amplitude = 6 * deg
	g, err := Layout(p)
	require.NoError(t, err)
	c := &checker{g: &g, in: actionInput{teeth: analyticToothPolys(), anchor: analyticAnchor(&g, 0.005)}}
	res, err := c.run()
	require.NoError(t, err)
	checkAction(t, &g, res, 0.005)
	// instructions.md: the pallet tops and arms stay 2.3 mm outside the tips and 1.2 mm clear of the root at A = 6°.
	require.InDelta(t, 2.34, res.topGap, 0.05)
	require.InDelta(t, 1.228, res.rootGap, 0.02)
}
