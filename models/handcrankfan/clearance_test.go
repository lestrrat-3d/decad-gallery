package handcrankfan_test

import (
	"fmt"
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
	return sectionGapMixed(t, a, tolerance, b, tolerance, plane)
}

// sectionGapMixed takes each body's section at its own tolerance, so a bevel gear's coarse mesh does not widen the
// other body's bound.
func sectionGapMixed(t *testing.T, a *decad.Body, ta float64, b *decad.Body, tb float64, plane r3.Frame) float64 {
	t.Helper()
	first, err := a.SectionOutline(t.Context(), plane, units.Millimeters(ta))
	require.NoError(t, err)
	second, err := b.SectionOutline(t.Context(), plane, units.Millimeters(tb))
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

// fitCheck collects every fit that falls short of its class, so one run names all of them.
type fitCheck struct {
	t        *testing.T
	failures []string
}

// atLeast records one fit: got is the proven lower bound and want the class minimum.
func (c *fitCheck) atLeast(id, what string, got, want float64) {
	c.t.Helper()
	c.t.Logf("%s %s: lower bound %.4f, class minimum %.2f", id, what, got, want)
	if got < want {
		c.failures = append(c.failures, fmt.Sprintf("%s %s: lower bound %.4f is below %.2f", id, what, got, want))
	}
}

func (c *fitCheck) done() {
	c.t.Helper()
	require.Empty(c.t, c.failures)
}

// shaftLine is a shaft's axis in the build frame: a point on it, its direction, the two directions across it that
// the spokes are measured from, and the motion that poses it.
type shaftLine struct {
	point, dir, e1, e2 r3.Vec
	motion             string
}

var (
	crankLine = shaftLine{r3.Vec{Z: 130}, r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}, "crank"}
	driveLine = shaftLine{r3.Vec{Z: 170}, r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}, "drive"}
	fanLine   = shaftLine{r3.Vec{Z: 170}, r3.Vec{Y: -1}, r3.Vec{X: 1}, r3.Vec{Z: 1}, "fan"}
)

// axial returns the plane through the shaft's axis that holds the posed direction d30, which lies between two
// grooves, so the plane meets only the round shaft.
func (l shaftLine) axial(t *testing.T, p handcrankfan.Params) r3.Frame {
	t.Helper()
	d30 := l.e1.Scale(math.Cos(math.Pi / 6)).Add(l.e2.Scale(math.Sin(math.Pi / 6)))
	origin, err := handcrankfan.PosePoint(p, l.motion, l.point)
	require.NoError(t, err)
	u, err := handcrankfan.PoseVector(p, l.motion, l.dir)
	require.NoError(t, err)
	v, err := handcrankfan.PoseVector(p, l.motion, d30)
	require.NoError(t, err)
	return frameAt(t, origin, u, v)
}

// atX is the plane across the X axis at x, atY the one across Y at y.
func atX(t *testing.T, x float64) r3.Frame {
	return frameAt(t, r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
}

func atY(t *testing.T, y float64) r3.Frame {
	return frameAt(t, r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: 1})
}

// atZ is the plane across Z at z, with U = X and V = Y.
func atZ(t *testing.T, z float64) r3.Frame {
	return frameAt(t, r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
}

// requireClearances measures every distinct fit of the default fan that the angle does not change, on the placed
// parts, and requires the proven lower bound to meet the class minimum. The bevel gears are faceted with a mesh bound
// of 0.103, so a section that includes one is taken at bevelTolerance. instructions.md lists the rows.
func requireClearances(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	c := &fitCheck{t: t}
	body := func(name string) *decad.Body { return parts[name].Body }
	section := func(id, a, b string, plane r3.Frame, class float64) {
		t.Helper()
		c.atLeast(id, a+" / "+b, sectionGap(t, body(a), body(b), plane), class)
	}
	exact := func(id, a, b string, plane r3.Frame, class float64) {
		t.Helper()
		c.atLeast(id, a+" / "+b, sectionGapAt(t, body(a), body(b), plane, sectionTolerance), class)
	}
	axial := func(id, lo, hi string, axis int, class float64) {
		t.Helper()
		c.atLeast(id, lo+" / "+hi+" axial", axialGap(t, body(lo), body(hi), axis), class)
	}

	// T1: the housing pieces across their seams, in the floor's section at Z 103, where every seam shows.
	for _, piece := range []string{"gearbox-base", "gearbox-post", "gearbox-crank-wall"} {
		section("T1", "gearbox-fan-wall", piece, atZ(t, 103), tight)
	}
	section("T1", "gearbox-post", "gearbox-base", atZ(t, 103), tight)
	section("T1", "gearbox-base", "gearbox-crank-wall", atZ(t, 103), tight)
	// T2, T3: the grip under the floor and beside the lower ears.
	axial("T2", "handle", "gearbox-base", 2, tight)
	for _, piece := range []string{"gearbox-fan-wall", "gearbox-post", "gearbox-crank-wall"} {
		for _, z := range []float64{100, 101.0} {
			section(fmt.Sprintf("T3 Z=%g", z), "handle", piece, atZ(t, z), tight)
		}
	}
	// T4: each bearing in its seat, outside diameter and face to floor.
	for _, site := range [...]struct {
		bearing, holder string
		line            shaftLine
	}{
		{"bearing-crank-post", "gearbox-post", crankLine}, {"bearing-drive-post", "gearbox-post", driveLine},
		{"bearing-crank-wall", "gearbox-crank-wall", crankLine}, {"bearing-drive-wall", "gearbox-crank-wall", driveLine},
		{"bearing-crank-block", "gearbox-crank-wall", crankLine},
		{"bearing-fan-wall", "gearbox-fan-wall", fanLine}, {"bearing-fan-block", "gearbox-fan-wall", fanLine},
	} {
		section("T4", site.bearing, site.holder, site.line.axial(t, p), tight)
	}
	// T5: each shaft in its bearings, and the collar to the bearing face.
	for _, site := range [...]struct {
		shaft, bearing string
		line           shaftLine
	}{
		{"crank-shaft", "bearing-crank-post", crankLine}, {"crank-shaft", "bearing-crank-wall", crankLine},
		{"crank-shaft", "bearing-crank-block", crankLine},
		{"drive-shaft", "bearing-drive-post", driveLine}, {"drive-shaft", "bearing-drive-wall", driveLine},
		{"fan-shaft", "bearing-fan-wall", fanLine}, {"fan-shaft", "bearing-fan-block", fanLine},
	} {
		exact("T5", site.shaft, site.bearing, site.line.axial(t, p), tight)
	}
	// T6: the splines.
	exact("T6", "crank-shaft", "crank-gear", atX(t, -6.2), tight)
	exact("T6", "drive-shaft", "drive-gear", atX(t, -6.2), tight)
	// T6b: the arm's splined bore, across and along the shaft's end.
	exact("T6b", "crank-shaft", "crank-arm", atX(t, 52), tight)
	exact("T6b", "crank-shaft", "crank-arm", crankLine.axial(t, p), tight)
	// T6c: the hub's spline and the collar face.
	exact("T6c", "fan-shaft", "fan-hub", atY(t, -86), tight)
	exact("T6c", "fan-shaft", "fan-hub", fanLine.axial(t, p), tight)
	// T6d: the bevel splines.
	bevel := func(id, gear, other string, plane r3.Frame, class float64) {
		t.Helper()
		c.atLeast(id, gear+" / "+other, sectionGapMixed(t, body(gear), bevelTolerance, body(other), fitTolerance, plane), class)
	}
	bevel("T6d", "driving-gear", "drive-shaft", atX(t, 19), tight)
	bevel("T6d", "pinion", "fan-shaft", atY(t, -29.5), tight)
	// T7: the pin's stub in the arm's pocket, across it and along the pin.
	pinOrigin, err := handcrankfan.PosePoint(p, "crank", r3.Vec{Z: 130 + 55})
	require.NoError(t, err)
	pinUp, err := handcrankfan.PoseVector(p, "crank", r3.Vec{Y: 1})
	require.NoError(t, err)
	pinPlane := frameAt(t, pinOrigin, r3.Vec{X: 1}, pinUp)
	exact("T7", "crank-pin", "crank-arm", atX(t, 55.5), tight)
	exact("T7", "crank-pin", "crank-arm", pinPlane, tight)
	// T8: the two gears on the drive shaft.
	axial("T8", "drive-gear", "driving-gear", 0, tight)

	// R1: each shaft through the housing.
	for _, pair := range [...][2]string{
		{"crank-shaft", "gearbox-post"}, {"crank-shaft", "gearbox-crank-wall"},
		{"drive-shaft", "gearbox-post"}, {"drive-shaft", "gearbox-crank-wall"}, {"fan-shaft", "gearbox-fan-wall"},
	} {
		line := crankLine
		switch pair[0] {
		case "drive-shaft":
			line = driveLine
		case "fan-shaft":
			line = fanLine
		}
		section("R1", pair[0], pair[1], line.axial(t, p), regular)
	}
	// R2, R4: a bevel gear's back face to the wall and the bearing.
	bevel("R2", "driving-gear", "gearbox-crank-wall", driveLine.axial(t, p), regular)
	bevel("R2", "pinion", "gearbox-fan-wall", fanLine.axial(t, p), regular)
	bevel("R4", "driving-gear", "bearing-drive-wall", driveLine.axial(t, p), regular)
	bevel("R4", "pinion", "bearing-fan-wall", fanLine.axial(t, p), regular)
	// R3: the driving gear to the back wall, taken at its heel, 0.1 inside the back face, where its radius is largest.
	bevel("R3", "driving-gear", "gearbox-base", atX(t, 20.9), regular)
	// R5: each blade's stud in the hub's socket, the stud end and the root to the rim.
	for k := 1; k <= p.Blades; k++ {
		section("R5", fmt.Sprintf("fan-blade-%d", k), "fan-hub", atY(t, -86), regular)
	}
	// R6, R7: the handle on the pin and against the arm.
	exact("R6", "crank-grip", "crank-pin", pinPlane, regular)
	axial("R7", "crank-arm", "crank-grip", 0, regular)

	// L1, L3, L4, L5: the gears' distance to the post and to the bevel gear, the arm to the block, the hub to the block.
	for _, gear := range []string{"crank-gear", "drive-gear"} {
		axial("L1", "gearbox-post", gear, 0, loose)
	}
	axial("L3", "crank-gear", "driving-gear", 0, loose)
	axial("L4", "gearbox-crank-wall", "crank-arm", 0, loose)
	axial("L5", "fan-hub", "gearbox-fan-wall", 1, loose)
	c.done()
}
