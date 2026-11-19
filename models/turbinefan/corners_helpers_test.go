package turbinefan_test

import (
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

// requireBladeCorners cuts the placed blade parallel to its print bed.
func requireBladeCorners(t *testing.T, m *turbinefan.Model, k int) {
	t.Helper()
	sz := m.Dimensions.Size
	d := bladeDir(k)
	normal := r3.Vec{X: -d.Y, Y: d.X}
	r := sz.BladeStudRadius
	heights := []float64{-r + 0.2, -2 * r / 3, -r / 3, 0, r / 3, 2 * r / 3, r - 0.01, r - 0.001}
	fs, err := cornercheck.CheckAt(t.Context(), m.Blades[k-1], normal, heights, nil)
	require.NoError(t, err)
	for _, f := range fs {
		h := f.Height
		if f.Kind != cornercheck.Tip || math.Abs(f.At.Dot(d)-(sz.BladeStudStart+0.3)) > 1e-8 ||
			(h != r-0.01 && h != r-0.001) {
			t.Fatalf("unexpected blade corner: %s", cornercheck.Report([]cornercheck.Finding{f}, 1))
		}
		// The narrow conical cap can combine two permitted turns in one tip window.
		b2 := r*r - h*h
		require.Greater(t, h, r-0.3)
		require.GreaterOrEqual(t, h, 0.4)
		require.GreaterOrEqual(t, b2, 0.0)
		require.Less(t, 2*math.Sqrt(b2*(1+b2/(r*r))), cornercheck.TipWindow)
		require.LessOrEqual(t, math.Abs(f.At.Z-sz.BladeSocketZ()), math.Sqrt(b2)+1e-7)
	}
}
