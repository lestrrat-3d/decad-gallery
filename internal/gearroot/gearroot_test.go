package gearroot_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/gearroot"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestClearance(t *testing.T) {
	require.InDelta(t, 0.5, gearroot.Clearance(1.5), 1e-12)
	require.InDelta(t, 1.0, gearroot.Clearance(4), 1e-12)
}

func TestDefineRootAndTip(t *testing.T) {
	doc := decad.New()
	p := spurgear.DefaultParams()
	p.Module, p.Teeth, p.Thickness, p.BoreDiameter, p.Backlash = 1.5, 15, 6, 0, 0.15
	body, err := spurgear.Variant{Define: gearroot.Define(0.5, 0.02)}.Build(t.Context(), doc, p)
	require.NoError(t, err)

	plane, err := r3.NewFrame(r3.Vec{Z: 3}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	section, err := body.SectionOutline(t.Context(), plane, units.Millimeters(0.01))
	require.NoError(t, err)
	require.Len(t, section.Loops, 1)

	nearest, farthest := math.Inf(1), 0.0
	for _, loop := range section.Loops {
		for _, v := range loop {
			r := math.Hypot(v.X, v.Y)
			nearest, farthest = math.Min(nearest, r), math.Max(farthest, r)
		}
	}
	// Pitch radius 11.25: root 11.25 - 1.5 - 0.5 - 0.02 = 9.23, tip 11.25 + 1.5 - 0.02 = 12.73.
	require.InDelta(t, 9.23, nearest, 0.05)
	require.InDelta(t, 12.73, farthest, 0.05)
}
