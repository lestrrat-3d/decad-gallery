package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentPivotBlockShape reads the block on its own: its bounds, its pin and its corners. It prints on its outer
// face with the pin up, so the print axis is -u (local -Z).
func TestSevenSegmentPivotBlockShape(t *testing.T) {
	_, block := sharedBody(t, "pivot block", (*run).pivotBlock)
	require.True(t, block.IsSolid())
	box, err := block.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -pivotBlockSide/2, box.Min.X, 0.01)
	require.InDelta(t, pivotBlockSide/2, box.Max.X, 0.01)
	require.InDelta(t, -pivotBlockHigh/2, box.Min.Y, 0.01)
	require.InDelta(t, pivotBlockHigh/2, box.Max.Y, 0.01)
	require.InDelta(t, pivotPinTip, box.Min.Z, 0.01, "pin tip")
	require.InDelta(t, pivotBlockFrom+pivotBlockLong, box.Max.Z, 0.01, "outer face")
	// 3.25 tall in print at Scale 1: 2.0 of block and 1.25 of pin.
	require.InDelta(t, pivotBlockLong+Scale*1.25, box.Max.Z-box.Min.Z, 0.01)
	// The pin stands on the block's rotor-side face, on its centre: the root cone is r 0.65 at u 12.5.
	pin := func(u float64) float64 {
		lo, hi := Scale*0.1, Scale*1.0
		for hi-lo > 0.002 {
			m := (lo + hi) / 2
			if inBody(t, block, r3.Vec{X: m, Z: u}) {
				lo = m
			} else {
				hi = m
			}
		}
		return (lo + hi) / 2
	}
	require.InDelta(t, pivotPinRadius, pin(pivotPinTip+Scale*0.75), 0.01, "pin radius")
	require.InDelta(t, pivotPinRadius+pivotRootLeg-0.1, pin(pivotBlockFrom-0.1), 0.01, "root cone, 45 degrees")
	require.InDelta(t, pivotPinRadius-pivotTipLeg+0.05, pin(pivotPinTip+0.05), 0.01, "tip chamfer")

	fs, err := cornercheck.Check(t.Context(), block, r3.Vec{Z: -1}, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}

// TestSevenSegmentPivotPocketFit measures the press of a block in its pocket on sections through the block's middle
// (u 13.5): 0.05 on both sides across the rotor and on the floor, and 0.05 under the roof once the seam closes (the top plate
// sits 0.25 lower than the cradle then). Segments g and b, both ends: the plates' pockets are one tool turned and moved, so
// these two cover both rotor directions and both ends.
func TestSevenSegmentPivotPocketFit(t *testing.T) {
	_, cradle := sharedBody(t, "cradle plate", (*run).cradlePlate)
	_, top := sharedBody(t, "top plate", (*run).flagTopPlate)
	for _, seg := range []Segment{SegG, SegB} {
		for end := range 2 {
			r := newTestRun(t)
			block := placedPivotBlock(t, r, seg, end)
			u := pivotBlockFrom + pivotBlockLong/2
			if end == 0 {
				u = -u
			}
			bl := pivotSection(t, block, seg, u)
			cl, tl := pivotSection(t, cradle, seg, u), pivotSection(t, top, seg, u)
			// Across the rotor, at the axis height of each plate's half: the block edge, then the pocket wall.
			across := func(loops [][]r3.Vec, z, sign float64) float64 {
				return scanEdge(loops, func(c float64) r3.Vec { return r3.Vec{X: sign * c, Y: z} }, 0, pivotBlockSide/2+0.5)
			}
			for _, sign := range []float64{-1, 1} {
				b := across(bl, 0, sign)
				require.InDelta(t, pivotBlockSide/2, b, 0.02, "%s end %d: block half width", seg, end)
				require.InDelta(t, PressFit, across(cl, -0.5, sign)-b, 0.03, "%s end %d: press across, cradle side %v", seg, end, sign)
				require.InDelta(t, PressFit, across(tl, 0.8, sign)-b, 0.03, "%s end %d: press across, top plate side %v", seg, end, sign)
			}
			// Along z at the middle: the block's bottom and top, the cradle's floor and the top plate's roof, all above the axis.
			up := func(loops [][]r3.Vec, from, to float64) float64 {
				return scanEdge(loops, func(c float64) r3.Vec { return r3.Vec{Y: c} }, from, to)
			}
			bottom, roof := up(bl, 0, -(pivotBlockHigh/2+0.5)), up(bl, 0, pivotBlockHigh/2+0.5)
			require.InDelta(t, -pivotBlockHigh/2, bottom, 0.02, "%s end %d: block bottom", seg, end)
			require.InDelta(t, pivotBlockHigh/2, roof, 0.02, "%s end %d: block top", seg, end)
			floor := up(cl, -0.5, -(pivotPocketLow + 0.5))
			require.InDelta(t, -pivotPocketLow, floor, 0.02, "%s end %d: cradle floor", seg, end)
			require.InDelta(t, 1.0, floor-(cradleFrom-RotorAxisZ), 0.02, "%s end %d: cradle under the pocket", seg, end)
			ceiling := up(tl, 0.5, pivotPocketHigh+0.5)
			require.InDelta(t, pivotPocketHigh, ceiling, 0.02, "%s end %d: top plate roof", seg, end)
			t.Logf("%s end %d: press floor %.3f; roof %.3f once the seam (%.2f) closes", seg, end, bottom-floor,
				ceiling-roof-Seam, Seam)
			require.InDelta(t, PressFit, bottom-floor, 0.03)
			require.InDelta(t, PressFit, ceiling-roof-Seam, 0.03)
		}
	}
}

// TestSevenSegmentPivotPinInBore checks the pin in its rotor's end bore on the real bodies: the Tight 0.20 round the pin,
// the pin's depth at the worst end play, and the floor.
func TestSevenSegmentPivotPinInBore(t *testing.T) {
	for _, seg := range []Segment{SegG, SegB} {
		f := sharedFlag(t, seg)
		r := newTestRun(t)
		for end := range 2 {
			block := placedPivotBlock(t, r, seg, end)
			sgn := float64(2*end - 1)
			// A section through the pin's middle: its disc lies inside the bore with 0.20 round it.
			pl := pivotSection(t, block, seg, sgn*(pivotPinTip+Scale*0.65))
			bl := pivotSection(t, f.Black, seg, sgn*(pivotPinTip+Scale*0.65))
			gap := loopsGap(pl, bl)
			t.Logf("%s end %d: pin to bore %.3f", seg, end, gap)
			require.GreaterOrEqual(t, gap, 0.2-0.04, "%s end %d", seg, end)
			require.InDelta(t, 0.2, gap, 0.03, "%s end %d", seg, end)
			// The pin stays inside the bore at the worst end play (the rotor 0.25 away from the block), 0.75 engaged, and its tip
			// stays 0.2 off the floor when the rotor is pushed 0.25 toward the block.
			floor := pivotBoreFloor
			require.GreaterOrEqual(t, (RotorHalfLength-rotorEndPlay)-pivotPinTip, 0.75-1e-9, "engagement")
			require.GreaterOrEqual(t, pivotPinTip-(floor+rotorEndPlay), 0.2-1e-9, "tip to floor at the other end play")
			// The bore's floor on the built black body: void at u floor+0.2, material at floor-0.2, along the axis.
			f0 := rotorFrame(seg)
			at := func(u float64) r3.Vec { return f0.Origin().Add(f0.N().Scale(sgn * u)) }
			require.False(t, inBody(t, f.Black, at(floor+0.2)), "bore")
			require.True(t, inBody(t, f.Black, at(floor-0.2)), "below the floor")
			// The pin's walls: 0.8 or more of black round the bore at the end face's flats, and the bore's mouth is 0.93.
			depth := pivotSection(t, f.Black, seg, sgn*(RotorHalfLength-0.05))
			var minx, maxx float64
			for _, l := range depth {
				for _, p := range l {
					minx, maxx = math.Min(minx, p.X), math.Max(maxx, p.X)
				}
			}
			require.Greater(t, maxx-minx, 2*pivotMouthR, "end face wider than the mouth")
		}
	}
}
