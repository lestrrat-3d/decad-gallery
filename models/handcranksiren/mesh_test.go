package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// sectionTolerance is the tolerance of the gear sections.
const sectionTolerance = 0.01

// gearRoot is the tip-to-root minimum of the Gear train clearances rule at module 1.5: max(0.25·1.5, 0.5).
const gearRoot = 0.5

// loopDistances returns the nearest and the farthest distance in the (X, Z) plane from the axis point to the loop. The
// nearest is measured to each segment, not to each vertex.
func loopDistances(loop []r3.Vec, axis r3.Vec) (nearest, farthest float64) {
	nearest = math.Inf(1)
	for i, p := range loop {
		next := loop[(i+1)%len(loop)]
		x0, y0 := p.X-axis.X, p.Z-axis.Z
		dx, dy := next.X-p.X, next.Z-p.Z
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

// outerOutline keeps only the loop of the section that reaches farthest from axis. The D-bore's loop never enters the
// root or reach computation.
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

// meshStage is one meshing pair: the driver turns about -Y and the driven gear about +Y, by the driver's angle times
// its tooth count over the driven gear's.
type meshStage struct {
	name                   string
	driver, driven         string
	driverZ, drivenZ       float64
	driverAxis, drivenAt   func(handcranksiren.State) r3.Vec
	midY, distance         float64
	driverRoot, drivenRoot float64
}

var meshStages = []meshStage{
	{"stage 1", "crank-gear", "intermediate-pinion", 50, 16,
		func(s handcranksiren.State) r3.Vec { return s.Crank }, func(s handcranksiren.State) r3.Vec { return s.Intermediate },
		23.6, 49.5, 35.48, 9.98},
	{"stage 2", "intermediate-gear", "rotor-pinion", 48, 15,
		func(s handcranksiren.State) r3.Vec { return s.Intermediate }, func(s handcranksiren.State) r3.Vec { return s.Rotor },
		15.6, 47.25, 33.98, 9.23},
}

// meshSamples turns each stage through one driver tooth pitch in four steps and calls check with the turned gears,
// their section outlines and the sample number.
func meshSamples(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part,
	check func(stage meshStage, driver, driven *decad.Body, driverOutline, drivenOutline decad.SectionOutline, plane r3.Frame, axisA, axisB r3.Vec, k int)) {
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	for _, stage := range meshStages {
		axisA, axisB := stage.driverAxis(state), stage.drivenAt(state)
		require.InDelta(t, stage.distance, math.Hypot(axisA.X-axisB.X, axisA.Z-axisB.Z), 1e-9, "%s centre distance", stage.name)
		plane := frameAt(t, r3.Vec{Y: stage.midY}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		pitch := 2 * math.Pi / stage.driverZ
		for k := range 5 {
			theta := float64(k) * pitch / 4
			driver, err := decad.Copy(t.Context(), parts[stage.driver].Body,
				decad.WithCopyRotation(axisA, r3.Vec{Y: -1}, units.Radians(theta)))
			require.NoError(t, err)
			driven, err := decad.Copy(t.Context(), parts[stage.driven].Body,
				decad.WithCopyRotation(axisB, r3.Vec{Y: 1}, units.Radians(theta*stage.driverZ/stage.drivenZ)))
			require.NoError(t, err)
			driverOutline := outerOutline(t, driver, plane, axisA)
			drivenOutline := outerOutline(t, driven, plane, axisB)
			check(stage, driver, driven, driverOutline, drivenOutline, plane, axisA, axisB, k)
			require.NoError(t, decad.Remove(driver))
			require.NoError(t, decad.Remove(driven))
		}
	}
}

// requireMesh checks both gear pairs at five angles across one tooth pitch: the tip of each gear clears its mate's root
// by the rule, each root loop sits at the gear's root radius, the outlines share no material, and both gears clear the
// frame.
func requireMesh(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	meshSamples(t, p, parts, func(stage meshStage, driver, driven *decad.Body, a, b decad.SectionOutline, plane r3.Frame, axisA, axisB r3.Vec, k int) {
		for _, order := range []struct {
			name        string
			a, b        decad.SectionOutline
			axisB       r3.Vec
			rootExpects float64
		}{
			{stage.name + " driver tip to driven root", a, b, axisB, stage.drivenRoot},
			{stage.name + " driven tip to driver root", b, a, axisA, stage.driverRoot},
		} {
			root, _ := loopDistances(order.b.Loops[0], order.axisB)
			require.InDelta(t, order.rootExpects, root, 0.05, "sample %d: the root loop is the gear's outer loop", k)
			root += order.b.OffSurface.Base()
			reach, _ := loopDistances(order.a.Loops[0], order.axisB)
			reach -= order.a.OffSurface.Base()
			require.GreaterOrEqual(t, reach-root, gearRoot, "sample %d: %s", k, order.name)
		}
		gap, err := a.Clearance(b)
		require.NoError(t, err)
		require.False(t, gap.Overlaps, "%s sample %d: the teeth share material", stage.name, k)
		frame := parts["gearbox-frame"].Body
		require.GreaterOrEqual(t, sectionGap(t, driver, frame, plane), loose, "%s sample %d, L2 %s / gearbox-frame", stage.name, k, stage.driver)
		require.GreaterOrEqual(t, sectionGap(t, driven, frame, plane), loose, "%s sample %d, L2 %s / gearbox-frame", stage.name, k, stage.driven)
	})
}
