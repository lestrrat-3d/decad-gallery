package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// clampFloorRound is the clamp nut pocket's 0.6 floor round, which a layer cuts at a slant and reads as a turn of 51.6
// to 54.4 degrees (instructions.md, "Retained sharp corners and edges"). It exempts that band only.
func clampFloorRound(dm turbinefan.Dimensions) cornercheck.Exempt {
	floor := -turbinefan.TabHalfWidth + m2fastener.NutCavityHeight
	return cornercheck.Exempt{
		Reason: "clamp nut pocket floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			return v.X < floor-0.05 && v.X > floor-turbinefan.NutPocketRound-0.1 &&
				math.Hypot(v.Y-dm.ClampY, v.Z-turbinefan.HubMid) <= m2fastener.NutCavityRadius+0.05
		},
	}
}

// clampHeadRim is the clamp bolt's head pocket floor rim, which the 0.3 mm head clearance leaves no room to round.
func clampHeadRim(dm turbinefan.Dimensions) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "clamp head pocket floor rim",
		Near: func(v r3.Vec) bool {
			return math.Abs(v.X-(turbinefan.TabHalfWidth-m2fastener.HeadPocketDepth)) < 0.05 &&
				math.Hypot(v.Y-dm.ClampY, v.Z-turbinefan.HubMid) <= m2fastener.HeadPocketRadius+0.05
		},
	}
}

// slitRound is the slit's end round in the bore's floor, a full round of half the slit's width that the held mesh
// chords coarsely enough for cornercheck to read.
func slitRound(dm turbinefan.Dimensions) cornercheck.Exempt {
	half := turbinefan.SlitWidth / 2
	return cornercheck.Exempt{
		Reason: "slit end round in the bore's floor",
		Near: func(v r3.Vec) bool {
			return math.Hypot(v.X, v.Y-(-dm.BoreRadius+turbinefan.SlitInBore-half)) <= half+0.05 &&
				v.Z > turbinefan.HubLength-turbinefan.BoreFloor
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
			return r >= handcrankfan.BladeSocketStart-0.05 && r <= handcrankfan.BladeSocketStart+0.9
		},
	}
}

// TestTurbineFanCorners applies "Corners and roots" to the hub, printed front face down with its axis +Z, and to each
// blade, printed on its stud end with its radial line vertical. The hub's retained corners are listed in
// instructions.md. The hub's layers run at the face levels MidHeights finds, and again at z = 7, 10 and 13, which cut the
// sockets and the clamp's pockets off their axes.
func TestTurbineFanCorners(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		fs, err := cornercheck.Check(t.Context(), m.Hub, r3.Vec{Z: 1},
			[]cornercheck.Exempt{clampFloorRound(dm), clampHeadRim(dm), slitRound(dm)})
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
		fs, err = cornercheck.CheckAt(t.Context(), m.Hub, r3.Vec{Z: 1}, []float64{7, 10, 13},
			[]cornercheck.Exempt{clampHeadRim(dm), socketFloorRound()})
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
// no section reads; instructions.md lists it as inherited from handcrankfan.
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
