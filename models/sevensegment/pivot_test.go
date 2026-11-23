package sevensegment_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentPivotBlocks checks the 14 blocks on the placed assembly at digit 0: each stands at its rotor end on the
// rotor's axis, touches no rotor, and keeps its press clearance to the cradle plate and the top plate. The rotors are
// at their modeled turn, so the block sits in the real consuming assembly.
func TestSevenSegmentPivotBlocks(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	_, cradle := meshOf(t, parts.CradlePlate, 0.05)
	_, top := meshOf(t, parts.TopPlate, 0.05)
	for _, s := range sevensegment.Segments() {
		cx, cy, horizontal := segmentCentre(s)
		_, yellow := meshOf(t, parts.Flags[s].Yellow, 0.05)
		_, black := meshOf(t, parts.Flags[s].Black, 0.05)
		flags := []meshgap.Mesh{yellow, black}
		for end, sgn := range []float64{-1, 1} {
			b := parts.PivotBlocks[s][end]
			require.NotNil(t, b, "segment %s end %d", s, end)
			require.True(t, b.IsSolid())
			box, err := b.Bounds()
			require.NoError(t, err)
			// The block's middle plane stands on the rotor's axis: RotorAxisZ.
			require.InDelta(t, sevensegment.RotorAxisZ, (box.Min.Z+box.Max.Z)/2, 0.01, "segment %s end %d height", s, end)
			lo, hi := box.Min.Y, box.Max.Y
			if horizontal {
				lo, hi = box.Min.X, box.Max.X
			}
			// The pin tip and the block's outer face, from the rotor's middle, on the end's side.
			centre := cx
			if !horizontal {
				centre = cy
			}
			if sgn > 0 {
				require.InDelta(t, centre+sevensegment.PivotPinTip, lo, 0.01, "segment %s tip", s)
				require.InDelta(t, centre+sevensegment.PivotBlockFrom+sevensegment.PivotBlockLong, hi, 0.01, "segment %s outer face", s)
			} else {
				require.InDelta(t, centre-sevensegment.PivotBlockFrom-sevensegment.PivotBlockLong, lo, 0.01, "segment %s outer face", s)
				require.InDelta(t, centre-sevensegment.PivotPinTip, hi, 0.01, "segment %s tip", s)
			}
			_, bm := meshOf(t, b, 0.05)
			for i, flag := range flags {
				ok, err := bm.ExceedsClearance(flag, 0)
				require.NoError(t, err)
				require.True(t, ok, "segment %s end %d overlaps its rotor body %d", s, end, i)
			}
			// The press leaves 0.05 on the sides and the floor; the meshes are read at 0.05, so only a lower bound of 0 is proven
			// here (TestSevenSegmentPivotPocketFit measures the 0.05 on sections).
			for name, plate := range map[string]meshgap.Mesh{"cradle": cradle, "top plate": top} {
				ok, err := bm.ExceedsClearance(plate, 0)
				require.NoError(t, err)
				require.True(t, ok, "segment %s end %d overlaps the %s", s, end, name)
			}
		}
	}
	// All 14 are distinct bodies.
	seen := map[interface{}]struct{}{}
	for _, s := range sevensegment.Segments() {
		for _, b := range parts.PivotBlocks[s] {
			seen[b] = struct{}{}
		}
	}
	require.Len(t, seen, 14)
}
