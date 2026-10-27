package screwgear_test

import (
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

var defaultWindows = sync.OnceValues(func() ([2]screwgear.WindowForTest, error) {
	return screwgear.WindowsForTest(screwgear.DefaultParams())
})

// TestWindowSearchAtTheDefaults pins the numbers the spec quotes for both default windows.
func TestWindowSearchAtTheDefaults(t *testing.T) {
	ws, err := defaultWindows()
	require.NoError(t, err)
	for i, w := range ws {
		require.Equal(t, "", w.Room, "window %d", i)
		require.Equal(t, 1.0, w.Lean)
		require.InDelta(t, -4.537, w.Lo, 5e-4)
		require.InDelta(t, 5.130, w.Hi, 5e-4)
		require.InDelta(t, -22.215, w.Bottom, 5e-4)
		require.InDelta(t, 22.808, w.Top, 5e-4)
		require.InDelta(t, -12.121, w.Left, 5e-4)
		require.InDelta(t, 12.172, w.Right, 5e-4)
		require.InDelta(t, 14.23, w.ZLimit, 5e-3) // the furthest the channels reach
		require.InDelta(t, 213.0, w.Area, 0.05)

		// The six corners in (t, z), in the order the clipping produces them.
		want := [][2]float64{
			{12.17, -7.04}, {-8.84, 13.97}, {-12.12, 10.69}, {-12.12, 7.58}, {8.84, -13.38}, {12.17, -10.04},
		}
		require.Len(t, w.Corners, len(want), "window %d", i)
		for j, c := range want {
			require.InDelta(t, c[0], w.Corners[j][0], 5e-3, "window %d corner %d t", i, j)
			require.InDelta(t, c[1], w.Corners[j][1], 5e-3, "window %d corner %d z", i, j)
		}

		// Long sides 6.84 mm apart across the band, ends 24.29 mm apart, reaching 13.97 mm.
		require.InDelta(t, 6.84, (w.Hi-w.Lo)/1.4142135623730951, 5e-3)
		require.InDelta(t, 24.29, w.Right-w.Left, 5e-3)
		top := 0.0
		for _, c := range w.Corners {
			top = max(top, c[1])
		}
		require.InDelta(t, 13.97, top, 5e-3)
		require.Less(t, top, w.ZLimit)
	}
	require.Equal(t, r3.NewVec(0, 1, 0), ws[0].Facing)
	require.Equal(t, r3.NewVec(0, -1, 0), ws[1].Facing)
}

// TestWindowSearchKeepsCollarWallToTheFlankingBores holds the band's sides to the wall: the
// band stays open, and a thicker wall narrows it.
func TestWindowSearchKeepsCollarWallToTheFlankingBores(t *testing.T) {
	p := screwgear.DefaultParams()
	p.CollarWall = 3.5
	ws, err := screwgear.WindowsForTest(p)
	require.NoError(t, err)
	base, err := defaultWindows()
	require.NoError(t, err)
	for i := range ws {
		require.Equal(t, "", ws[i].Room)
		// The band's measure z + lean*t grows by sqrt(2) per mm, so each side moves in by
		// sqrt(2) times the extra wall.
		require.InDelta(t, base[i].Hi-base[i].Lo-2*1.4142135623730951*0.5, ws[i].Hi-ws[i].Lo, 2e-3)
	}
}

// TestWindowSearchRoomRefusesTheBandWithNoWidth reaches the window that is not cut: with a
// wall this thick the flanking bores leave no band.
func TestWindowSearchRoomRefusesTheBandWithNoWidth(t *testing.T) {
	p := screwgear.DefaultParams()
	p.CollarWall = 7
	ws, err := screwgear.WindowsForTest(p)
	require.NoError(t, err)
	for _, w := range ws {
		require.Equal(t, "the flanking bores leave no band between them", w.Room)
	}
}
