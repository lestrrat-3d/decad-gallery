package screwgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSleeveChannelSeparationAtTheDefaults(t *testing.T) {
	sep, which := screwgear.ChannelSeparationForTest(screwgear.DefaultParams())
	// 5.310 mm across the gap between the two -R bores.
	require.InDelta(t, 5.310, sep, 5e-4)
	require.Contains(t, which, "gear A -R")
	require.Contains(t, which, "gear B -R")
}

func TestSleeveChannelSeparationShrinksWithTheWall(t *testing.T) {
	p := screwgear.DefaultParams()
	base, _ := screwgear.ChannelSeparationForTest(p)
	p.Clearance = 0.55
	wide, _ := screwgear.ChannelSeparationForTest(p)
	require.Less(t, wide, base)
}

func TestSleeveWindowFacingsFollowTheCrossingAngle(t *testing.T) {
	p := screwgear.DefaultParams()
	require.Equal(t, [2]r3.Vec{r3.NewVec(0, 1, 0), r3.NewVec(0, -1, 0)}, screwgear.WindowFacingsForTest(p))
	p.CrossAngle = deg(90)
	require.Equal(t, [2]r3.Vec{r3.NewVec(0, 1, 0), r3.NewVec(0, -1, 0)}, screwgear.WindowFacingsForTest(p))
	p.CrossAngle = deg(100)
	require.Equal(t, [2]r3.Vec{r3.NewVec(1, 0, 0), r3.NewVec(-1, 0, 0)}, screwgear.WindowFacingsForTest(p))
}
