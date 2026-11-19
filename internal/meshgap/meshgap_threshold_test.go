package meshgap

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestClearanceThresholdUpperFollowsExceedsRoundingOrder(t *testing.T) {
	boundA := 46095.381876896776
	boundB := 46610.127105692416
	clearance := 62320.35242257626

	margin := boundA + boundB + 1e-6
	limit := clearance + margin
	reorderedLimit := clearance + boundA + boundB + 1e-6
	require.NotEqual(t, math.Float64bits(limit), math.Float64bits(reorderedLimit))

	upperMargin := roundUp(boundA + boundB)
	upperMargin = roundUp(upperMargin + 1e-6)
	upperLimit := roundUp(clearance + upperMargin)
	want := roundUp(upperLimit * upperLimit)
	got, ok := clearanceThresholdUpper(clearance, boundA, boundB)
	require.True(t, ok)
	require.Equal(t, math.Float64bits(want), math.Float64bits(got))
	require.Greater(t, got, limit*limit)
}

func TestExceedsClearancePrunesSeparatedMultiTriangleTree(t *testing.T) {
	build := func(points []r3.Vec) Mesh {
		t.Helper()
		faces := make([][3]int, len(points))
		for i := range faces {
			faces[i] = [3]int{i, i, i}
		}
		m, err := New(points, faces, 0)
		require.NoError(t, err)
		return m
	}
	leftPoints := []r3.Vec{{}}
	rightPoints := []r3.Vec{{X: 2e-6}}
	for i := range 8 {
		x := 100 + 2*float64(i)
		leftPoints = append(leftPoints, r3.Vec{X: x, Y: 100, Z: 100})
		rightPoints = append(rightPoints, r3.Vec{X: x, Y: 110, Z: 100})
	}
	left, right := build(leftPoints), build(rightPoints)
	require.Len(t, left.triangles, 9)
	require.Len(t, right.triangles, 9)
	require.Zero(t, squaredBoxGap(triangle{lo: left.lo, hi: left.hi}, triangle{lo: right.lo, hi: right.hi}))
	threshold, ok := clearanceThresholdUpper(0, left.bound, right.bound)
	require.True(t, ok)
	leftCross, valid := lowerBoxGap(
		triangle{lo: left.nodes[left.nodes[0].left].lo, hi: left.nodes[left.nodes[0].left].hi},
		triangle{lo: right.nodes[right.nodes[0].right].lo, hi: right.nodes[right.nodes[0].right].hi},
	)
	require.True(t, valid)
	require.Greater(t, leftCross, threshold)

	for _, pair := range [][2]Mesh{{left, right}, {right, left}} {
		_, want, wantErr := pair[0].Exceeds(pair[1], 0)
		got, gotErr := pair[0].ExceedsClearance(pair[1], 0)
		require.NoError(t, wantErr)
		require.NoError(t, gotErr)
		require.True(t, want)
		require.Equal(t, want, got)
	}
}
