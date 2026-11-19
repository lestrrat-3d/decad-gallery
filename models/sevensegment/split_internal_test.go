package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// atY exempts the corners of the shaft at world Y y, within 0.06 mm.
func atY(reason string, y float64) cornercheck.Exempt {
	return cornercheck.Exempt{Reason: reason, Optional: true, Near: func(p r3.Vec) bool { return math.Abs(p.Y-y) < 0.06 }}
}

// TestSevenSegmentSplitShaft applies "Split shafts" to the shaft posed at digits 0 and 2 (theta 0 and 72): the split
// plane holds the axis and the flats' normal, which turns with the camshaft, and the halves meet the rule's checks. The
// square steps and flat ends that face a bearing or a D chord are the only exempt corners.
func TestSevenSegmentSplitShaft(t *testing.T) {
	for _, digit := range []int{0, 2} {
		r, shaft := sharedBodyFor(t, Params{Digit: digit}, "shaft", (*run).shaft)
		shaft, err := decad.Transform(t.Context(), shaft, r.camshaftTurn())
		require.NoError(t, err)
		point, axis := r3.Vec{Z: ShaftZ}, r3.Vec{Y: 1}
		rep, err := splitcheck.Check(t.Context(), splitcheck.Shaft{Body: shaft, Point: point, Axis: axis}, 0.01)
		require.NoError(t, err)
		require.Empty(t, rep.Violations)
		// The flats face +Z at theta 0; the split plane's normal is perpendicular to both the axis and the flats' normal.
		turn := r.camshaftTurn()
		want := turn.ApplyDir(r3.Vec{X: 1})
		require.InDelta(t, 1.0, math.Abs(rep.Normal.Dot(want)), 1e-3, "digit %d", digit)
		heights, err := cornercheck.SplitHeights(point, rep.Normal, 2.5)
		require.NoError(t, err)
		fs, err := cornercheck.CheckAt(t.Context(), shaft, rep.Normal, heights, []cornercheck.Exempt{
			atY("front step facing the front bearing", -49.25), atY("rear collar's rear face facing the rear bearing", 42.75),
			atY("cam flat's square end under cam a's D chord", 39.2), atY("stub flat's square end under the collar ring", -57.85),
		})
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
	}
}
