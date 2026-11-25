package meshgap

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestExceedsClearanceRowSkipKeepsEqualLowerBound pins the leaf-row skip in exceedsClearanceThreshold. The first
// triangle of mesh A has a row bound against the leaf box of mesh B that equals the threshold exactly, so only a
// strict comparison keeps it. The comparison must stay `lower > threshold`: `>=`, or a comparison with the
// squared limit, wrongly skips the row and reports the pair as clear.
func TestExceedsClearanceRowSkipKeepsEqualLowerBound(t *testing.T) {
	const (
		farX      = 54175.93691948187
		clearance = 1.0301886252127567e-06
	)
	x0 := 54175.93691745168
	near := x0 + (farX-x0)/2

	vertsA := []r3.Vec{
		{X: -11799.23218657791}, {X: x0}, {X: -11799.23218657791, Y: 1},
		{X: near, Y: 100}, {X: near, Y: 101}, {X: near - 1, Y: 100},
	}
	vertsB := []r3.Vec{{X: farX}, {X: farX, Y: 1}, {X: farX, Z: 1}}
	facesA := [][3]int{{0, 1, 2}, {3, 4, 5}}
	facesB := [][3]int{{0, 1, 2}}

	a, err := New(vertsA, facesA, 0)
	require.NoError(t, err)
	b, err := New(vertsB, facesB, 0)
	require.NoError(t, err)
	got, err := a.ExceedsClearance(b, clearance)
	require.NoError(t, err)

	refA, err := refNew(vertsA, facesA, 0)
	require.NoError(t, err)
	refB, err := refNew(vertsB, facesB, 0)
	require.NoError(t, err)
	want, err := refA.ExceedsClearance(refB, clearance)
	require.NoError(t, err)

	require.False(t, want, "reference ExceedsClearance changed; the pinned behavior moved")
	require.False(t, got, "ExceedsClearance reported clear; reference reports not proven clear")
}
