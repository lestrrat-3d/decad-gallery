package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentPivotBlockShape reads the block on its own: its bounds, its stepped cone and its corners. It prints on its
// outer face with the cone up, so the print axis is -u (local -Z).
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
	require.InDelta(t, pivotOuter, box.Max.Z, 0.01, "outer face")
	// 5.575 tall in print: 1.807 of block and 3.768 of pin.
	require.InDelta(t, pivotOuter-pivotPinTip, box.Max.Z-box.Min.Z, 0.01, "height in print")
	// The cone stands on the block's rotor-side face, on its centre. radius reads its radius at u by bisection.
	radius := func(u float64) float64 {
		lo, hi := 0.0, 2.3
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
	require.InDelta(t, pivotContactR, radius(pivotContactU), 0.01, "cone at the bore's edge")
	require.InDelta(t, 2*pivotJunctionR, 2*radius(pivotJunctionU), 0.02, "diameter at the junction of frustum and cone")
	require.InDelta(t, pivotBaseR-0.01, radius(pivotBlockFrom-0.01), 0.01, "base")
	require.InDelta(t, pivotTipR, radius(pivotPinTip+pivotTipR), 0.01, "ball tip at its equator")
	require.InDelta(t, pivotJunctionR+0.5, radius(pivotJunctionU+0.5), 0.01, "frustum at 45 degrees")
	require.InDelta(t, pivotTangentR+(17.0-pivotTangentU)*pivotConeTan, radius(17.0), 0.01, "cone at 8 degrees")

	// Every face of the cone leans up in print: the least normal . up over the cone's triangles is sin 8 degrees.
	m := pivotMesh(t, block, 0.02)
	vs := m.Vertices()
	least := 1.0
	for _, tri := range m.Triangles() {
		a, b, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		if math.Max(a.Z, math.Max(b.Z, c.Z)) > pivotBlockFrom-1e-6 {
			continue // the block's own faces
		}
		n := b.Sub(a).Cross(c.Sub(a))
		if n.Len() < 1e-12 {
			continue
		}
		least = math.Min(least, n.Scale(1/n.Len()).Dot(r3.Vec{Z: -1}))
	}
	require.Greater(t, least, 0.9*pivotConeSin, "least normal . up on the cone's triangles: the cone has no overhang")

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
			u := pivotBlockFrom + pivotBlockLong/2 // inside the block, where the plates press it
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

// TestSevenSegmentPivotPinInBore measures the cone in its rotor's end bore on the real bodies, segments g and b at both ends:
// the least gap between the cone's and the bore's sections along the engaged length with the rotor 0.2 away from the block,
// at the modeled pose and 0.2 toward the block (the tip on its floor), the tip's distance from the floor, and the mouth.
func TestSevenSegmentPivotPinInBore(t *testing.T) {
	for _, seg := range []Segment{SegG, SegB} {
		f := sharedFlag(t, seg)
		r := newTestRun(t)
		for end := range 2 {
			block := placedPivotBlock(t, r, seg, end)
			sgn := float64(2*end - 1)
			// The gap is the least over sections every 0.02 along the engaged length, where the bore and the cone both exist.
			// With the rotor moved a toward the block (a section of the bore at u - a), it falls at the bore's edge.
			// Radially the gap is the modeled pivotClear, larger by the end play times tan 8 degrees when the rotor is away, and
			// pivotTight (0.2 normal to the cone) when the tip is on its floor.
			want := map[float64]float64{-pivotEndPlay: pivotClear + pivotEndPlay*pivotConeTan, 0: pivotClear, pivotEndPlay: pivotTight}
			for _, a := range []float64{-pivotEndPlay, 0, pivotEndPlay} {
				least := math.Inf(1)
				for u := pivotPinTip + 0.3; u <= RotorHalfLength-0.01+a; u += 0.02 {
					if u-a > RotorHalfLength-0.01 {
						continue
					}
					pl := pivotSection(t, block, seg, sgn*u)
					bl := pivotSection(t, f.Black, seg, sgn*(u-a))
					least = math.Min(least, loopsGap(pl, bl))
				}
				t.Logf("%s end %d, rotor %+.2f toward the block: least section gap %.4f", seg, end, a, least)
				// Each gap is at least the Tight 0.2 less the 0.04 the section gap's bound allows.
				require.GreaterOrEqual(t, least, 0.2-0.04, "%s end %d, rotor %+.2f", seg, end, a)
				require.InDelta(t, want[a], least, 0.03, "%s end %d, rotor %+.2f", seg, end, a)
			}
			// The tip stands pivotEndPlay off the floor at the modeled pose: void at the floor + 0.05, material under it.
			f0 := rotorFrame(seg)
			at := func(x, u float64) r3.Vec { return f0.Origin().Add(f0.U().Scale(x)).Add(f0.N().Scale(sgn * u)) }
			require.False(t, inBody(t, f.Black, at(0, pivotBoreFloor+0.05)), "bore")
			require.True(t, inBody(t, f.Black, at(0, pivotBoreFloor-0.05)), "below the floor")
			require.False(t, inBody(t, block, at(0, pivotPinTip-0.05)), "beyond the tip")
			require.True(t, inBody(t, block, at(0, pivotPinTip+0.05)), "the tip")
			require.InDelta(t, pivotEndPlay, pivotPinTip-pivotBoreFloor, 1e-9, "tip to floor")
			// The end face is wider than the mouth, so the mouth does not open the face's edge.
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
