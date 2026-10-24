package cycloidal

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// The contour dips to the pin surface at the pin and rises to the lobe-tip path between pins.
func TestHousingContour(t *testing.T) {
	p := DefaultParams()
	d := resolve(p)
	points := housingContour(p, d)
	require.Len(t, points, contourPoints)
	radii := make([]float64, len(points))
	for i, q := range points {
		radii[i] = math.Hypot(q.x, q.y)
	}
	half := math.Pi / float64(p.PinCount)
	require.InDelta(t, -half, math.Atan2(points[0].y, points[0].x), 1e-12)
	require.InDelta(t, half, math.Atan2(points[len(points)-1].y, points[len(points)-1].x), 1e-12)
	require.InDelta(t, radii[0], radii[len(radii)-1], 0.02, "ends are mirror images")
	low, high := math.Inf(1), 0.0
	for _, r := range radii {
		low, high = math.Min(low, r), math.Max(high, r)
	}
	// The valley at the pin is the pin surface (R - Rr) and the peak is R - Rr + 2E, each within a bin.
	require.InDelta(t, d.pinCircleRadius-d.pinRadius, low, 0.1)
	require.InDelta(t, d.pinCircleRadius-d.pinRadius+2*p.Eccentricity, high, 0.1)
	require.Less(t, high, d.housingOuter/2-p.HousingWall+1e-9)
}
