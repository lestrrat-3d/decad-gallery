package cubepuzzle_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestFastenersAreFlushAndNutsSeated(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	for i, axis := range []r3.Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: 1}, {Z: -1}} {
		// The +Z fastener is made with the source disk before the other five.
		boltIndex := []int{1, 2, 3, 4, 0, 5}[i]
		disk, err := a.Disks[i].Bounds()
		require.NoError(t, err)
		bolt, err := a.Bolts[boltIndex].Bounds()
		require.NoError(t, err)
		nut, err := a.Nuts[boltIndex].Bounds()
		require.NoError(t, err)
		maxAlong := func(minimum, maximum r3.Vec) float64 {
			return math.Max(minimum.Dot(axis), maximum.Dot(axis))
		}
		minAlong := func(minimum, maximum r3.Vec) float64 {
			return math.Min(minimum.Dot(axis), maximum.Dot(axis))
		}
		require.InDelta(t, maxAlong(disk.Min, disk.Max), maxAlong(bolt.Min, bolt.Max),
			disk.Bound.Base()+bolt.Bound.Base()+1e-6, "head %d is not flush", i)
		require.GreaterOrEqual(t, minAlong(nut.Min, nut.Max)-nut.Bound.Base(), 8.5, "nut %d inner end", i)
		require.LessOrEqual(t, maxAlong(nut.Min, nut.Max)+nut.Bound.Base(), 10.5, "nut %d outer end", i)
		require.Greater(t, minAlong(bolt.Min, bolt.Max)-bolt.Bound.Base(), 0.0, "bolt %d tip", i)
	}
}
