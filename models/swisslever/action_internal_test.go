package swisslever

// CI: isolated-tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file holds the wheel and stone checker of instructions.md "Geometry checks": a quasi-static simulation of the
// tooth and stone outlines as closed polygons. The lever angle follows the dwell law psi = clamp(-1.25 a cos 2 pi t,
// -a, a); at each step the wheel turns the smallest angle that frees an overlap (a stone pushing a tooth from behind
// turns the wheel forward, a drawn locking face pushes it back) and then advances as far as the outlines allow.

func TestActionAnalytic(t *testing.T) {
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	c := newChecker(&g, analyticTeeth(&g), analyticStones(&g), 2000)
	res, err := c.run()
	require.NoError(t, err)
	requireAction(t, &g, res, 1e-6*deg, 1.8, 1.0)
	// The drops are the closed-form drops (1.890 and 1.920 mm) plus one step of the simulation.
	want := closedDrops(t, &g)
	for _, d := range res.drops {
		require.InDelta(t, want[d.psi > 0], d.size, 0.04, "drop at psi %v", d.psi/deg)
		require.Less(t, d.landing, 1e-6, "the toe lands on a locking face")
	}
	require.Less(t, res.maxPhi, 1e-6*deg, "closed form")
	require.InDelta(t, 1.031, res.others, 0.02, "E1")
	require.InDelta(t, 2.310, res.rootGap, 0.01, "E2")
	require.InDelta(t, 1.452, res.topGap, 0.01, "E3")
}

func TestErrorStudy(t *testing.T) {
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	for _, e := range []printError{
		{0, 0, 0}, {-0.2, 0, 0}, {0.2, 0, 0}, {0, 0.2, 0}, {0, -0.2, 0}, {0, 0, 0.15}, {0, 0, -0.15},
		{0.2, 0, -0.15}, {0.2, 0.2, -0.15}, {-0.2, 0.2, -0.15}, {-0.2, -0.2, 0.15}, {0.2, -0.2, 0.15}, {-0.2, 0, 0.15},
	} {
		t.Run(fmt.Sprintf("dr %v dw %v dd %v", e.dr, e.dw, e.dd), func(t *testing.T) {
			res, err := errorCase(&g, e, 700).run()
			require.NoError(t, err)
			requireAction(t, &g, res, 0.05*deg, 1.55, 0.4)
		})
	}
}
