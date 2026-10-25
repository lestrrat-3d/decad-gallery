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

// contourConfigs are drives the contour tests sweep.
var contourConfigs = map[string]func(p *Params){
	"default":       func(p *Params) {},
	"twelve pins":   func(p *Params) { p.PinCount = 12 },
	"odd pins":      func(p *Params) { p.PinCount = 13 },
	"big eccentric": func(p *Params) { p.Eccentricity = 2 },
	"clearance":     func(p *Params) { p.DiskClearance = 0.6 },
}

// The contour from one pitch's share of the cam turn, with folded segments, stays within the profile's chord
// error of the contour from a much finer sweep over the whole turn, which is what the rotor profile's 2000
// points limit: the old whole-turn sweep at 720 cam angles is no closer. That error is 0.002 mm at the default
// drive, against a 0.3 mm clearance.
func TestHousingContourStaysNearFineSweep(t *testing.T) {
	for name, edit := range contourConfigs {
		t.Run(name, func(t *testing.T) {
			p := DefaultParams()
			edit(&p)
			require.Empty(t, Problems(p))
			d := resolve(p)
			fine := housingContourSweep(p, d, 1440, 8000)
			worst := func(a []point) float64 {
				w := 0.0
				for i := range fine {
					w = math.Max(w, math.Hypot(a[i].x-fine[i].x, a[i].y-fine[i].y))
				}
				return w
			}
			got, old := worst(housingContour(p, d)), worst(housingContourFull(p, d))
			t.Logf("folded %.3g mm, whole turn at 720 %.3g mm", got, old)
			require.Less(t, got, 0.01)
			require.Less(t, got, old+0.001)
		})
	}
}
