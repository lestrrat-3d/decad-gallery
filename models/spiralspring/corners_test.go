package spiralspring_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSpiralSpringCorners applies "Corners and roots" to the spring in its print orientation, axis +Z. No corner is
// exempt: the D bore's corners turn 35 degrees.
func TestSpiralSpringCorners(t *testing.T) {
	for _, c := range springCases {
		if c.name == "max-turns" {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			m, err := spiralspring.Build(t.Context(), decad.New(), c.params())
			require.NoError(t, err)
			fs, err := cornercheck.Check(t.Context(), m.Body, r3.Vec{Z: 1}, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}

// TestSpiralSpringRootThickness checks the taper: where the strip leaves the spoke it is t + 2e thick.
func TestSpiralSpringRootThickness(t *testing.T) {
	for _, c := range springCases {
		if c.name == "max-turns" {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, err := spiralspring.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			dm := m.Dimensions
			mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(meshTolerance))
			require.NoError(t, err)
			segs := sliceSegments(mesh, p.Height/2)

			b := vec2{dm.CenterOffset, 0}
			b0 := vec2{dm.CenterOffset + dm.TaperOffset, 0}
			outerR := dm.FirstInnerRadius + p.Thickness
			innerR := dm.FirstInnerRadius - dm.TaperOffset
			wantAt := func(phi float64) float64 { return p.Thickness + dm.TaperOffset*(1-math.Cos(phi)) }
			var inner []segment
			for _, s := range segs {
				if math.Abs(s.midpoint().distTo(b0)-innerR) < sliceMargin && s.midpoint()[1] < -dm.SpokeWidth-dm.FilletRadius {
					inner = append(inner, s)
				}
			}
			require.NotEmpty(t, inner)
			checked := 0
			for _, s := range segs {
				mid := s.midpoint()
				if math.Abs(mid.distTo(b)-outerR) > sliceMargin || mid[1] > -(dm.SpokeWidth+dm.FilletRadius+0.8) {
					continue
				}
				phi := math.Atan2(mid[1]-b[1], mid[0]-b[0])
				if phi < 0 {
					phi += 2 * math.Pi
				}
				if wantAt(phi) < p.Thickness+2*dm.TaperOffset-0.02 {
					continue
				}
				dist := math.Inf(1)
				for _, in := range inner {
					dist = math.Min(dist, pointSegmentDistance(mid, in))
				}
				require.GreaterOrEqual(t, dist, p.Thickness+2*dm.TaperOffset-0.02-sliceMargin, "phi %.1f", phi*180/math.Pi)
				require.LessOrEqual(t, dist, p.Thickness+2*dm.TaperOffset+sliceMargin, "phi %.1f", phi*180/math.Pi)
				checked++
			}
			require.Positive(t, checked)
		})
	}
}
