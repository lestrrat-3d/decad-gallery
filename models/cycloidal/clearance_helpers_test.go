package cycloidal_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// sectionGap returns the proven lower bound on the gap between a and b in the plane z = height above the drive
// axis frame, including both sections' mesh bounds.
func sectionGap(t *testing.T, a, b *decad.Body, height float64) float64 {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, height), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	tolMM := 0.01
	for _, body := range []*decad.Body{a, b} {
		bounds, err := body.Bounds()
		require.NoError(t, err)
		tolMM = math.Max(tolMM, bounds.Bound.Base()+0.001)
	}
	tol := units.Millimeters(tolMM)
	first, err := a.SectionOutline(t.Context(), plane, tol)
	require.NoError(t, err)
	second, err := b.SectionOutline(t.Context(), plane, tol)
	require.NoError(t, err)
	gap, err := first.Clearance(second)
	require.NoError(t, err)
	require.False(t, gap.Overlaps)
	return gap.Lower().Base()
}

// axialGap returns the proven lower bound on the distance from the top of low to the bottom of high.
func axialGap(t *testing.T, low, high *decad.Body) float64 {
	t.Helper()
	lo, err := low.Bounds()
	require.NoError(t, err)
	hi, err := high.Bounds()
	require.NoError(t, err)
	return hi.Min.Z - lo.Max.Z - lo.Bound.Base() - hi.Bound.Base()
}
