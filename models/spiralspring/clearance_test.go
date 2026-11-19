package spiralspring_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSpiralSpringClearance(t *testing.T) {
	t.Run("pair-helper", func(t *testing.T) {
		never := func(p, q segment) bool { return false }
		air := []segment{
			{a: vec2{0, 0}, b: vec2{1, 0}, n: vec2{0, 1}},
			{a: vec2{0, 0.5}, b: vec2{1, 0.5}, n: vec2{0, -1}},
		}
		minAir, minMaterial := pairLimits(air, 2, never)
		require.InDelta(t, 0.5, minAir, 1e-12)
		require.True(t, math.IsInf(minMaterial, 1))
		wall := []segment{
			{a: vec2{0, 0}, b: vec2{1, 0}, n: vec2{0, -1}},
			{a: vec2{0, 0.5}, b: vec2{1, 0.5}, n: vec2{0, 1}},
		}
		minAir, minMaterial = pairLimits(wall, 2, never)
		require.True(t, math.IsInf(minAir, 1))
		require.InDelta(t, 0.5, minMaterial, 1e-12)
	})

	for _, c := range []struct {
		name string
		edit func(p *spiralspring.Params)
	}{
		{"default", func(*spiralspring.Params) {}},
		{"thickness-floor", func(p *spiralspring.Params) {
			p.OuterDiameter, p.Thickness, p.MinGap, p.Height = 30, 1.5, 1, 4
		}},
		{"large", func(p *spiralspring.Params) {
			p.OuterDiameter, p.HubDiameter, p.Thickness, p.MinGap, p.Height = 120, 30, 3, 3, 8
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := spiralspring.DefaultParams()
			c.edit(&p)
			m, err := spiralspring.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			dm := m.Dimensions
			mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(meshTolerance))
			require.NoError(t, err)
			segs := sliceSegments(mesh, p.Height/2)
			require.NotEmpty(t, segs)

			inHole := func(s segment) bool {
				mid := s.midpoint()
				return mid.len() <= dm.BoreRadius+0.01 ||
					mid.distTo(vec2{dm.EyeCenterX, dm.EyeCenterY}) <= dm.EyeHoleDiameter/2+0.01
			}
			// The taper narrows the gap between the hub and half turn 0 to g - 2e at the spoke, so that pair is
			// measured on its own.
			// The hub side includes the C4 fillet, which leaves the hub circle by up to r_f.
			onHub := func(s segment) bool {
				r := s.midpoint().len()
				return r > dm.HubRadius-sliceMargin && r < dm.HubRadius+dm.FilletRadius+sliceMargin
			}
			onFirst := func(s segment) bool {
				d := s.midpoint().distTo(vec2{dm.CenterOffset + dm.TaperOffset, 0})
				// The C3 fillet adds material inside the edge's circle, by up to r_f.
				edge := dm.FirstInnerRadius - dm.TaperOffset
				return d > edge-dm.FilletRadius-sliceMargin && d < edge+sliceMargin
			}
			hubPair := func(p, q segment) bool { return onHub(p) && onFirst(q) || onFirst(p) && onHub(q) }
			skip := func(p, q segment) bool { return inHole(p) && inHole(q) || hubPair(p, q) }
			reach := 1.5 * math.Max(dm.Gap, p.Thickness)
			minAir, minMaterial := pairLimits(segs, reach, skip)
			minHub, _ := pairLimits(segs, reach, func(p, q segment) bool { return !hubPair(p, q) })
			require.GreaterOrEqual(t, minHub, dm.Gap-2*dm.TaperOffset-sliceMargin, "hub to half turn 0")
			require.LessOrEqual(t, minHub, dm.Gap+sliceMargin, "hub to half turn 0")
			msg := fmt.Sprintf("minAir %.4f (gap %.4f), minMaterial %.4f (thickness %.4f)", minAir, dm.Gap, minMaterial, p.Thickness)
			require.GreaterOrEqual(t, minAir, dm.Gap-sliceMargin, msg)
			require.LessOrEqual(t, minAir, dm.Gap+sliceMargin, msg)
			require.GreaterOrEqual(t, minMaterial, p.Thickness-sliceMargin, msg)
			require.LessOrEqual(t, minMaterial, p.Thickness+sliceMargin, msg)
		})
	}
}
