package handcrankfan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// The class minima of the model-pipeline rules and the fastener and gear-root bounds. The tests do not read the
// model's own nominal values: they prove the placed parts meet these.
const (
	tight       = 0.2
	regular     = 0.35
	loose       = 0.5
	fastenerGap = 0.05
	gearRoot    = 0.5
)

// sectionTolerance is the tolerance of the gear sections. fitTolerance is the tolerance of the fit sections: the
// faceted housing pieces cannot be sectioned below a mesh bound of 0.0146, so 0.01 is refused for them. The bevel
// gears are faceted with a mesh bound of 0.103, so a section that includes one is taken at bevelTolerance and its
// proven gap is charged that bound.
const (
	sectionTolerance = 0.01
	fitTolerance     = 0.02
	bevelTolerance   = 0.11
)

// frameAt returns the plane through origin spanned by u and v.
func frameAt(t *testing.T, origin, u, v r3.Vec) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(origin, u, v)
	require.NoError(t, err)
	return f
}

// sectionGap returns the proven lower bound on the gap between a and b in the plane, including both sections'
// mesh bounds. The sections must not overlap.
func sectionGap(t *testing.T, a, b *decad.Body, plane r3.Frame) float64 {
	t.Helper()
	return sectionGapAt(t, a, b, plane, fitTolerance)
}

// sectionGapAt is sectionGap with the section tolerance named, for pairs that include a bevel gear.
func sectionGapAt(t *testing.T, a, b *decad.Body, plane r3.Frame, tolerance float64) float64 {
	t.Helper()
	first, err := a.SectionOutline(t.Context(), plane, units.Millimeters(tolerance))
	require.NoError(t, err)
	second, err := b.SectionOutline(t.Context(), plane, units.Millimeters(tolerance))
	require.NoError(t, err)
	gap, err := first.Clearance(second)
	require.NoError(t, err)
	require.False(t, gap.Overlaps)
	return gap.Lower().Base()
}

// axialGap returns the proven lower bound on the distance from the top of lo to the bottom of hi along axis 0 (X),
// 1 (Y) or 2 (Z).
func axialGap(t *testing.T, lo, hi *decad.Body, axis int) float64 {
	t.Helper()
	a, err := lo.Bounds()
	require.NoError(t, err)
	b, err := hi.Bounds()
	require.NoError(t, err)
	component := func(v r3.Vec) float64 { return [3]float64{v.X, v.Y, v.Z}[axis] }
	return component(b.Min) - component(a.Max) - a.Bound.Base() - b.Bound.Base()
}

// loopDistances returns the nearest and the farthest distance in the (Y, Z) plane from the axis point to the loop.
// The nearest is measured to each segment, not to each vertex.
func loopDistances(loop []r3.Vec, axis r3.Vec) (nearest, farthest float64) {
	nearest = math.Inf(1)
	for i, p := range loop {
		next := loop[(i+1)%len(loop)]
		x0, y0 := p.Y-axis.Y, p.Z-axis.Z
		dx, dy := next.Y-p.Y, next.Z-p.Z
		length := dx*dx + dy*dy
		fraction := 0.0
		if length > 0 {
			fraction = math.Max(0, math.Min(1, -(x0*dx+y0*dy)/length))
		}
		nearest = math.Min(nearest, math.Hypot(x0+fraction*dx, y0+fraction*dy))
		farthest = math.Max(farthest, math.Hypot(x0, y0))
	}
	return nearest, farthest
}

// outerOutline keeps only the loop of the section that reaches farthest from axis. The spline bore's loop never
// enters the root or reach computation.
func outerOutline(t *testing.T, body *decad.Body, plane r3.Frame, axis r3.Vec) decad.SectionOutline {
	t.Helper()
	section, err := body.SectionOutline(t.Context(), plane, units.Millimeters(sectionTolerance))
	require.NoError(t, err)
	require.NotEmpty(t, section.Loops)
	best, reach := 0, -1.0
	for i, loop := range section.Loops {
		if _, far := loopDistances(loop, axis); far > reach {
			best, reach = i, far
		}
	}
	section.Loops = [][]r3.Vec{section.Loops[best]}
	return section
}

// requireSpurMesh turns the spur pair through one tooth pitch in four steps and measures at each step the tooth tip
// to the mate's root against the gear rule, and the pair's other fits that change with the angle.
func requireSpurMesh(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	state, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	require.InDelta(t, 40, state.Apex.Sub(state.Crank).Len(), 1e-9, "the spur pair's centre distance")
	plane := frameAt(t, r3.Vec{X: -6.2}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	pitch := 2 * math.Pi / 20
	for k := range 5 {
		theta := float64(k) * pitch / 4
		crank, err := decad.Copy(t.Context(), parts["crank-gear"].Body,
			decad.WithCopyRotation(state.Crank, r3.Vec{X: -1}, units.Radians(theta)))
		require.NoError(t, err)
		drive, err := decad.Copy(t.Context(), parts["drive-gear"].Body,
			decad.WithCopyRotation(state.Apex, r3.Vec{X: 1}, units.Radians(theta)))
		require.NoError(t, err)

		crankOutline := outerOutline(t, crank, plane, state.Crank)
		driveOutline := outerOutline(t, drive, plane, state.Apex)
		// For each ordering, the tip of A reaches toward B's axis and B's root is its nearest point to that axis.
		for _, order := range []struct {
			name         string
			a, b         decad.SectionOutline
			axisB        r3.Vec
			rootBExpects float64
		}{
			{"crank tip to drive root", crankOutline, driveOutline, state.Apex, 17.48},
			{"drive tip to crank root", driveOutline, crankOutline, state.Crank, 17.48},
		} {
			rootB, _ := loopDistances(order.b.Loops[0], order.axisB)
			require.InDelta(t, order.rootBExpects, rootB, 0.05, "sample %d: the root loop is the gear's outer loop", k)
			rootB += order.b.OffSurface.Base()
			reachA, _ := loopDistances(order.a.Loops[0], order.axisB)
			reachA -= order.a.OffSurface.Base()
			require.GreaterOrEqual(t, reachA-rootB, gearRoot, "sample %d at %.4f rad: %s", k, theta, order.name)
		}
		gap, err := crankOutline.Clearance(driveOutline)
		require.NoError(t, err)
		require.False(t, gap.Overlaps, "sample %d: the spur teeth share material", k)

		// The fits that change with the angle: gear to housing, and gear to the bevel pinion's toe.
		require.GreaterOrEqual(t, sectionGap(t, crank, parts["gearbox-base"].Body, plane), loose,
			"sample %d, L2 crank-gear / gearbox-base", k)
		require.GreaterOrEqual(t, sectionGap(t, drive, parts["gearbox-base"].Body, plane), loose,
			"sample %d, L2 drive-gear / gearbox-base", k)
		require.GreaterOrEqual(t, sectionGap(t, drive, parts["gearbox-fan-wall"].Body, plane), loose,
			"sample %d, L2 drive-gear / gearbox-fan-wall", k)
		require.GreaterOrEqual(t, sectionGapAt(t, crank, parts["pinion"].Body, plane, bevelTolerance), loose,
			"sample %d, L3 crank-gear / pinion", k)
		require.GreaterOrEqual(t, sectionGapAt(t, drive, parts["pinion"].Body, plane, bevelTolerance), loose,
			"sample %d, L3 drive-gear / pinion", k)

		require.NoError(t, decad.Remove(crank))
		require.NoError(t, decad.Remove(drive))
	}
}
