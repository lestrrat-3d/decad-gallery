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
				math.Abs(v.X) <= flatsHalf+0.05 && v.Z >= turbinefan.SetScrewZ+m2fastener.NutCavityRadius/2-0.3
		},
	}
}

// optional returns the exemptions with each allowed to claim no finding.
func optional(exempt ...cornercheck.Exempt) []cornercheck.Exempt {
	for i := range exempt {
		exempt[i].Optional = true
	}
	return exempt
}

// setScrewSlotSideRounds is the 0.5 round where the set screw slot's straight sides meet its faces. A layer in the lower
// part of the slot (z up to 2.6) chords it into turns of 50.0 to 50.6 degrees on some hubs, just over the 50 degree limit,
// and under it on others, as the face levels fall differently.
func setScrewSlotSideRounds(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason:   "set screw slot side round chorded just over the limit",
		Optional: true,
		Near: func(v r3.Vec) bool {
			return -v.Y >= dm.BoreFlat+turbinefan.PressWall-0.05 && -v.Y <= dm.BoreFlat+turbinefan.PressWall+2.05 &&
				math.Abs(v.X) >= flatsHalf-0.6 && math.Abs(v.X) <= flatsHalf+0.05 && v.Z <= 2.6
		},
	}
}

// socketFloorRound is a blade socket's 0.5 floor round, which a layer off the socket's axis cuts at a slant and reads
// as a turn of about 58 degrees on the floor's outline, BladeSocketStart to BladeSocketStart + 0.9 from the shaft axis.
func socketFloorRound(sz turbinefan.Size) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "blade socket floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			r := math.Hypot(v.X, v.Y)
			return r >= sz.BladeSocketStart-0.05 && r <= sz.BladeSocketStart+0.9
		},
	}
}

// TestTurbineFanCorners applies "Corners and roots" to the hub, printed front face down with its axis +Z, and to each
// blade, printed on its stud end with its radial line vertical. The hub is read at the face levels MidHeights finds and at
// z = 2.5, 4, 5.5, 5.8, HubMid - 3 j, HubMid and HubMid + 3 j, which cut the slots, the counterbore and the sockets off
// their axes. Three exemptions
// are allowed; instructions.md records their measured turns.
func TestTurbineFanCorners(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		sz := dm.Size
		// The adapter holes' blind floor (z 6.75) adds mid heights 5.575 and 12.375, which cut the set screw slot's end
		// round and the socket floor rounds at a slant.
		// Which of these the face levels cut at a slant depends on the hub's length, so none is required to claim a finding
		// here; the listed heights below cut each of them, and there each must.
		fs, err := cornercheck.Check(t.Context(), m.Hub, r3.Vec{Z: 1},
			optional(counterboreRound(dm), setScrewSlotEndRound(dm), socketFloorRound(sz), setScrewSlotSideRounds(dm)))
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
		heights := []float64{2.5, 4, 5.5, 5.8, sz.HubMid - 3*sz.HubScale, sz.HubMid, sz.HubMid + 3*sz.HubScale}
		fs, err = cornercheck.CheckAt(t.Context(), m.Hub, r3.Vec{Z: 1}, heights,
			[]cornercheck.Exempt{counterboreRound(dm), setScrewSlotEndRound(dm), socketFloorRound(sz), setScrewSlotSideRounds(dm)})
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
// 64 to 90 degrees and the checker adds the stud's curvature. The loft is cut at radii read from the station table
// (the root's and the airfoil's, each scaled with its station), because it has no planar faces to give heights. The step
// where the stud meets the root face (RootRadius) is a corner between layers, which no section reads; instructions.md
// lists it.
func requireBladeCorners(t *testing.T, m *turbinefan.Model, k int) {
	t.Helper()
	sz := m.Dimensions.Size
	j, s := sz.HubScale, sz.Scale
	d := bladeDir(k)
	centre := r3.Vec{Z: sz.HubMid}
	body := m.Blades[k-1]
	// The bolt hole's radius is fixed, so the layers through it lie at its centre and 0.7 off it, not scaled.
	stud := []float64{7.0 * j, sz.BladeBoltAt, sz.BladeBoltAt + 0.7, 12.5 * j}
	fs, err := cornercheck.CheckAt(t.Context(), body, d, stud, []cornercheck.Exempt{
		// At 128 the four corners read as tips of 125 to 130 degrees. A stud of radius 4.05 (192, 224) meets the hole at
		// a wider angle and reads none, so the exemption need not claim a finding.
		{Reason: "bolt hole across the stud", Near: cornercheck.Disc(centre, d, sz.BladeStudRadius+0.1), Optional: true},
	})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
	loft := []float64{sz.RootRadius + 0.2*j, sz.RootRadius + 2.1*j}
	for _, r := range []float64{18, 22, 26, 30, 36, 39.2, 44, 52, 56, 60, 62, 63, 63.8} {
		loft = append(loft, r*s)
	}
	fs, err = cornercheck.CheckAt(t.Context(), body, d, loft, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}
