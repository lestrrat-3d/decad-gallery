package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanCorners applies "Corners and roots" to the hub, printed front face down with its axis +Z, and to each
// blade, read across its radial line and, as it prints, lying in its print pose on the bed (requireBladePrintCorners). The hub is read at the face levels MidHeights finds and at
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
	for _, c := range []struct {
		rotor, pitchDeg float64
		hand            turbinefan.Hand
		required        bool
	}{
		{128, 25, turbinefan.RightHand, false}, {192, 10, turbinefan.RightHand, false},
		{192, 25, turbinefan.RightHand, true}, {192, 40, turbinefan.RightHand, false},
		{224, 40, turbinefan.RightHand, false}, {192, 25, turbinefan.LeftHand, false},
	} {
		t.Run(fmt.Sprintf("print-%g-%g-%d", c.rotor, c.pitchDeg, c.hand), func(t *testing.T) {
			requireBladePrintCorners(t, c.rotor, c.pitchDeg*math.Pi/180, c.hand, c.required)
		})
	}
}
