package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// counterboreRound is the set screw counterbore's 0.5 floor round, which a layer off the screw's line cuts at a slant and
// reads as a turn over 45 degrees.
func counterboreRound(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "counterbore floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			r := math.Hypot(v.X, v.Z-turbinefan.SetScrewZ)
			return math.Abs(-v.Y-dm.CounterboreFloor) <= turbinefan.CounterboreRound+0.1 &&
				r >= m2fastener.HeadPocketRadius-turbinefan.CounterboreRound-0.05 && r <= m2fastener.HeadPocketRadius+0.05
		},
	}
}

// setScrewSlotEndRound is the set screw slot's rounded sloped end, which a layer cuts at a slant.
func setScrewSlotEndRound(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "set screw slot end round cut at a slant",
		Near: func(v r3.Vec) bool {
			return -v.Y >= dm.BoreFlat+turbinefan.PressWall-0.05 && -v.Y <= dm.BoreFlat+turbinefan.PressWall+2.05 &&
				math.Abs(v.X) <= flatsHalf+0.05 && v.Z >= turbinefan.SetScrewZ+m2fastener.NutCavityRadius/2-0.1
		},
	}
}

// socketFloorRound is a blade socket's 0.5 floor round, which a layer off the socket's axis cuts at a slant and reads
// as a turn of about 58 degrees on the floor's outline, 5.7 to 6.6 from the shaft axis.
func socketFloorRound() cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "blade socket floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			r := math.Hypot(v.X, v.Y)
			return r >= turbinefan.BladeSocketStart-0.05 && r <= turbinefan.BladeSocketStart+0.9
		},
	}
}

// TestTurbineFanCorners applies "Corners and roots" to the hub, printed front face down with its axis +Z, and to each
// blade, printed on its stud end with its radial line vertical. The hub is read at the face levels MidHeights finds and at
// z = 2.5, 4, 5.5, 5.8, 7, 10 and 13, which cut the slots, the counterbore and the sockets off their axes. Three exemptions
// are allowed, each only for the call whose findings it matches; instructions.md records their measured turns.
func TestTurbineFanCorners(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		fs, err := cornercheck.Check(t.Context(), m.Hub, r3.Vec{Z: 1}, []cornercheck.Exempt{counterboreRound(dm)})
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
		fs, err = cornercheck.CheckAt(t.Context(), m.Hub, r3.Vec{Z: 1}, []float64{2.5, 4, 5.5, 5.8, 7, 10, 13},
			[]cornercheck.Exempt{counterboreRound(dm), setScrewSlotEndRound(dm), socketFloorRound()})
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
	})
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		for k := 1; k <= turbinefan.BladeCount; k++ {
			t.Run(fmt.Sprintf("blade-%d", k), func(t *testing.T) { requireBladeCorners(t, m, k) })
		}
	})
}

// requireBladeCorners checks blade k along its radial line d, in the hub's mid plane. The stud is cut at the radii where
// its bolt hole crosses it; the hole's four corners (r 9.5 and 10.2) are the one exemption, where the material angle is
// 64 to 90 degrees and the checker adds the stud's curvature. The loft is cut at the listed radii, because it has no
// planar faces to give heights. The step where the stud meets the root face (r 14.4) is a corner between layers, which
// no section reads; instructions.md lists it.
func requireBladeCorners(t *testing.T, m *turbinefan.Model, k int) {
	t.Helper()
	d := bladeDir(k)
	centre := r3.Vec{Z: turbinefan.HubMid}
	body := m.Blades[k-1]
	stud := []float64{7.0, 9.5, 10.2, 12.5}
	fs, err := cornercheck.CheckAt(t.Context(), body, d, stud, []cornercheck.Exempt{
		{Reason: "bolt hole across the stud", Near: cornercheck.Disc(centre, d, 3.1)},
	})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
	fs, err = cornercheck.CheckAt(t.Context(), body, d,
		[]float64{14.6, 16.5, 18, 22, 26, 30, 36, 39.2, 44, 52, 56, 60, 62, 63, 63.8}, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}
