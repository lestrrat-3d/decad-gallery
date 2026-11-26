package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentPivotConstants checks the pivot's derived constants against their definitions and the values the design
// measured: the trigonometry of the 8 degree cone, the wedge factor, the block's face and the junction.
func TestSevenSegmentPivotConstants(t *testing.T) {
	a := 8 * math.Pi / 180
	require.InDelta(t, math.Sin(a), pivotConeSin, 1e-12)
	require.InDelta(t, math.Cos(a), pivotConeCos, 1e-12)
	require.InDelta(t, math.Tan(a), pivotConeTan, 1e-12)
	// The friction radius factor of the cone's contact at friction 0.5, against a straight journal.
	const mu = 0.5
	require.InDelta(t, math.Sqrt(1+mu*mu)/math.Sqrt(pivotConeCos*pivotConeCos+mu*mu), pivotWedge, 1e-6)

	require.InDelta(t, 0.2301, pivotClear, 0.0001)
	require.InDelta(t, 0.6699, pivotContactR, 0.0001)
	require.InDelta(t, 18.0334, pivotContactU, 0.0001)
	require.InDelta(t, 1.2416, pivotMouthR, 0.0001)
	require.InDelta(t, 0.2464, pivotJournalPlay, 0.0001)
	require.InDelta(t, 19.8182, pivotBlockFrom, 0.0001)
	require.InDelta(t, 1.8068, pivotBlockLong, 0.0001)
	require.InDelta(t, 18.5625, pivotJunctionU, 0.0001)
	require.InDelta(t, 0.7443, pivotJunctionR, 0.0001)
	// The junction lies on the cone's line and the frustum is the 45 degree line from the base on the block's face.
	require.InDelta(t, pivotTangentR+(pivotJunctionU-pivotTangentU)*pivotConeTan, pivotJunctionR, 1e-9)
	require.InDelta(t, pivotBaseR, pivotBlockFrom-pivotFrustumC, 1e-9)
	// The cone is tangent to the ball: the ball's centre is pivotTipR from the cone's line.
	cone := func(u float64) float64 { return pivotTangentR + (u-pivotTangentU)*pivotConeTan }
	centre := pivotPinTip + pivotTipR
	require.InDelta(t, pivotTipR, (cone(centre))*pivotConeCos, 1e-9)
	// The bore's edge lies inside the end face, and the block's face stands clear of the rotor's end face.
	require.Less(t, pivotContactU, RotorHalfLength)
	require.Greater(t, pivotBlockFrom, RotorHalfLength+pivotEndPlay)
	require.Less(t, pivotBlockFrom, pivotWall+pivotEndPlay+2)
}

// TestSevenSegmentPivotSketches requires every sketch of a pivot block and of both rotor types (their bores and roofs)
// to be fully constrained: DOF 0, no redundant constraints, no conflicts.
func TestSevenSegmentPivotSketches(t *testing.T) {
	r := newTestRun(t)
	_, err := r.pivotBlock()
	require.NoError(t, err)
	for _, seg := range []Segment{SegG, SegB} {
		_, err := r.flag(seg)
		require.NoError(t, err)
	}
	require.NotEmpty(t, r.sketches)
	for i, s := range r.sketches {
		report := s.Verify(t.Context())
		require.Equal(t, sketch.FullyConstrained, report.Status, "sketch %d", i)
		require.Equal(t, 0, report.DOF, "sketch %d", i)
		require.Empty(t, report.Redundant, "sketch %d", i)
		require.Empty(t, report.Conflicts, "sketch %d", i)
	}
}

// TestSevenSegmentPivotFrustumClearance measures the frustum and the block's face against both rotor bodies, segments g and
// b at both ends, with the rotor pushed pivotEndPlay toward the block and dropped by pivotTight in each of 8 directions: the
// gap stays 0.2 or more (the Tight minimum). The block's whole body at the modeled pose keeps 0.2 as well.
func TestSevenSegmentPivotFrustumClearance(t *testing.T) {
	for _, seg := range []Segment{SegG, SegB} {
		f := sharedFlag(t, seg)
		r := newTestRun(t)
		black, yellow := pivotMesh(t, f.Black, 0.02), pivotMesh(t, f.Yellow, 0.02)
		for end := range 2 {
			block := placedPivotBlock(t, r, seg, end)
			sgn := float64(2*end - 1)
			fr := rotorFrame(seg)
			o := fr.Origin()
			bm := pivotMesh(t, block, 0.02)
			// The block's triangles from just short of the junction outward: the frustum and the face.
			frustum := func(c r3.Vec) bool { return c.Sub(o).Dot(fr.N())*sgn >= pivotJunctionU-0.02 }
			worst := math.Inf(1)
			for k := range 8 {
				th := 2 * math.Pi * float64(k) / 8
				dir := fr.U().Scale(math.Cos(th)).Add(fr.V().Scale(math.Sin(th)))
				shift := fr.N().Scale(sgn * pivotEndPlay).Add(dir.Scale(pivotTight))
				worst = math.Min(worst, pivotMeshGap(t, bm, frustum, black, shift))
				worst = math.Min(worst, pivotMeshGap(t, bm, frustum, yellow, shift))
			}
			t.Logf("%s end %d: frustum to the rotor, pushed and dropped: %.4f", seg, end, worst)
			require.GreaterOrEqual(t, worst, 0.2-1e-3, "%s end %d: frustum to the rotor", seg, end)
			modeled := math.Min(pivotMeshGap(t, bm, nil, black, r3.Vec{}), pivotMeshGap(t, bm, nil, yellow, r3.Vec{}))
			require.GreaterOrEqual(t, modeled, 0.2-1e-3, "%s end %d: block to the rotor at the modeled pose", seg, end)
		}
	}
}

// TestSevenSegmentPivotFloorWall measures the black material under each bore floor along the axis at local x 0.6, where
// type B's second pocket comes closest: 0.8 or more stays.
func TestSevenSegmentPivotFloorWall(t *testing.T) {
	for _, seg := range []Segment{SegG, SegB} {
		f := sharedFlag(t, seg)
		fr := rotorFrame(seg)
		for end := range 2 {
			sgn := float64(2*end - 1)
			at := func(u float64) r3.Vec { return fr.Origin().Add(fr.U().Scale(0.6)).Add(fr.N().Scale(sgn * u)) }
			low := pivotBoreFloor
			for u := pivotBoreFloor - 0.01; u > pivotBoreFloor-3; u -= 0.01 {
				if !inBody(t, f.Black, at(u)) {
					break
				}
				low = u
			}
			t.Logf("%s end %d: material under the floor, %.2f thick", seg, end, pivotBoreFloor-low)
			require.GreaterOrEqual(t, pivotBoreFloor-low, 0.8, "%s end %d: wall under the floor", seg, end)
		}
	}
}
