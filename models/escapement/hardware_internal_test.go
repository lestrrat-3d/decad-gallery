package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// requireNoOverlap checks that a and b share no volume by their section outlines at each z.
func requireNoOverlap(t *testing.T, a, b *decad.Body, zs ...float64) {
	t.Helper()
	for _, z := range zs {
		oa, ob := sectionAt(t, a, z), sectionAt(t, b, z)
		c, err := oa.Clearance(ob)
		require.NoError(t, err)
		require.False(t, c.Overlaps, "overlap at z %v", z)
	}
}

func TestFasteners(t *testing.T) {
	b := testBuilder(t)
	back, err := b.buildBackPlate()
	require.NoError(t, err)
	front, err := b.buildFrontPlate()
	require.NoError(t, err)
	h, err := b.buildHardware()
	require.NoError(t, err)
	require.Len(t, h.Bolts, 4)
	require.Len(t, h.Nuts, 4)

	for i, p := range pillars {
		bolt, nut := h.Bolts[i], h.Nuts[i]
		bb, err := bolt.Bounds()
		require.NoError(t, err)
		// The head's top is flush with the front plate's face at z 34; the tip is inside the pillar hole (z 22).
		require.InDelta(t, headTop, bb.Max.Z, 2e-3, "head top (Bounds is conservative by about 2e-4)")
		require.InDelta(t, headTop-m2Tip, bb.Min.Z, 2e-3, "bolt tip")
		require.Greater(t, bb.Min.Z, 21.0, "the tip stays inside the pillar hole, which ends at z 21")
		require.InDelta(t, p.X, (bb.Min.X+bb.Max.X)/2, 2e-3)
		// The head sits in its pocket: its radius stays inside the pocket's 2.2.
		require.Less(t, math.Max(bb.Max.X-p.X, bb.Max.Y-p.Y), 2.2)
		nb, err := nut.Bounds()
		require.NoError(t, err)
		// The nut lies inside the cavity z [24, 26], radius 2.51.
		require.GreaterOrEqual(t, nb.Min.Z, 24.0)
		require.LessOrEqual(t, nb.Max.Z, 26.0)
		for _, c := range []float64{nb.Min.X - p.X, nb.Max.X - p.X, nb.Min.Y - p.Y, nb.Max.Y - p.Y} {
			require.LessOrEqual(t, math.Abs(c), 2.51)
		}
		// No bolt or nut shares volume with the plates it passes through (hole 1.3, cavity 2.51, pocket 2.2).
		requireNoOverlap(t, bolt, back, 22.5, 23, 26.5, 27.5)
		requireNoOverlap(t, bolt, front, 29, 31, 33)
		requireNoOverlap(t, nut, back, 24.5, 25, 25.5)
		// The nut sits in a cavity closed on every side except the bolt hole: a roof above and a pillar below.
		require.True(t, inside(t, back, p.X+2.0, p.Y, 26.5), "roof over the nut")
		require.True(t, inside(t, back, p.X+2.0, p.Y, 23.5), "floor under the nut")
		// 1 mm or more of material round the cavity and under the head pocket: the pillar wall is 5 - 2.51.
		require.GreaterOrEqual(t, pillarRadius-2.51, 1.0)
		require.InDelta(t, 4.0, frontHigh-frontLow-2.0, 1e-9, "plate under the head pocket")
	}
}
