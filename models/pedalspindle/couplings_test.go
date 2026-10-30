package pedalspindle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPedalSpindlePositiveCouplings(t *testing.T) {
	doc := decad.New()
	a, err := pedalspindle.Build(t.Context(), doc, pedalspindle.DefaultParams())
	require.NoError(t, err)
	for _, joint := range []struct {
		part, shaft            string
		center                 r3.Vec
		y, radius, flat, angle float64
	}{
		{"input-gear", "input-shaft", a.State.Input, -36, 4.1, 3.1, math.Pi / 3},
		{"intermediate-pinion", "intermediate-shaft", a.State.Intermediate, -36, 4.1, 3.1, a.State.IntermediateAngle},
		{"intermediate-gear", "intermediate-shaft", a.State.Intermediate, -66, 4.1, 3.1, a.State.IntermediateAngle},
		{"output-pinion", "spindle", a.State.Output, -66, 2.6, 1.85, a.State.SpindleAngle},
		{"crank", "input-shaft", a.State.Input, -109, 4.1, 3.1, math.Pi / 3},
		{"flywheel-hub", "input-shaft", a.State.Input, 44, 4.1, 3.1, math.Pi / 3},
	} {
		body, shaft := partByName(t, a, joint.part), partByName(t, a, joint.shaft)
		requireDBore(t, doc.OperationHistory(), body, joint.center, joint.y, joint.angle, joint.radius, joint.flat)
		plane, err := r3.NewFrame(r3.NewVec(0, joint.y, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
		require.NoError(t, err)
		for _, delta := range []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
			gearCopy, err := decad.Copy(t.Context(), body, decad.WithCopyRotation(joint.center, r3.NewVec(0, -1, 0), units.Radians(delta)))
			require.NoError(t, err)
			shaftCopy, err := decad.Copy(t.Context(), shaft, decad.WithCopyRotation(joint.center, r3.NewVec(0, -1, 0), units.Radians(delta)))
			require.NoError(t, err)
			gover, err := gearCopy.SectionOutline(t.Context(), plane, units.Millimeters(.01))
			require.NoError(t, err)
			sover, err := shaftCopy.SectionOutline(t.Context(), plane, units.Millimeters(.01))
			require.NoError(t, err)
			gap, err := gover.Clearance(sover)
			require.NoError(t, err)
			require.False(t, gap.Overlaps, "%s rigid phase %.2f", joint.part, delta)
			require.Greater(t, gap.Lower().Base(), 0.0, "%s certified assembly clearance", joint.part)
			require.NoError(t, decad.Remove(gearCopy))
			require.NoError(t, decad.Remove(shaftCopy))
		}
		wrong, err := decad.Copy(t.Context(), shaft, decad.WithCopyRotation(joint.center, r3.NewVec(0, -1, 0), units.Degrees(90)))
		require.NoError(t, err)
		gearOutline, err := body.SectionOutline(t.Context(), plane, units.Millimeters(.01))
		require.NoError(t, err)
		shaftOutline, err := wrong.SectionOutline(t.Context(), plane, units.Millimeters(.01))
		require.NoError(t, err)
		gap, err := gearOutline.Clearance(shaftOutline)
		require.NoError(t, err)
		require.True(t, gap.Overlaps, "%s must prevent free relative shaft rotation", joint.part)
		require.NoError(t, decad.Remove(wrong))
	}
}

func recordedCut(t *testing.T, history *decad.OperationHistory, body *decad.Body) *decad.Operation {
	t.Helper()
	for range 12 {
		op, err := history.Producer(body)
		require.NoError(t, err)
		if op.Kind() == decad.OperationCut {
			return op
		}
		require.True(t, op.Kind() == decad.OperationCopy || op.Kind() == decad.OperationTransform,
			"expected a placed/copied Cut, got %v", op.Kind())
		inputs := op.Inputs()
		require.Equal(t, 1, len(inputs))
		body = inputs[0].Body
	}
	t.Fatal("operation chain exceeds expected coupling depth")
	return nil
}

func cutInput(t *testing.T, op *decad.Operation, role string) *decad.Body {
	t.Helper()
	for _, input := range op.Inputs() {
		if input.Role == role {
			return input.Body
		}
	}
	t.Fatalf("Cut lacks %s input", role)
	return nil
}

// Read the actual recorded primitive inputs exactly. The derived D tool and
// final gear remain bounded geometry; their output contours are checked too.
func requireDBore(t *testing.T, history *decad.OperationHistory, body *decad.Body, center r3.Vec, y, angle, radius, flat float64) {
	t.Helper()
	tool := cutInput(t, recordedCut(t, history, body), "tool")
	op := recordedCut(t, history, tool)
	circular, cap := cutInput(t, op, "target"), cutInput(t, op, "tool")
	bounds, err := tool.Bounds()
	require.NoError(t, err)
	z := (bounds.Min.Z + bounds.Max.Z) / 2
	frame, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	_, profiles, err := circular.Section(t.Context(), frame)
	require.NoError(t, err)
	require.Equal(t, 1, len(profiles))
	require.True(t, profiles[0].Valid)
	require.Equal(t, 1, len(profiles[0].Outer))
	circle, ok := profiles[0].Outer[0].Entity.(*sketch.Circle)
	require.True(t, ok, "actual D-tool target must be a whole circle")
	require.InDelta(t, radius, circle.R(), 1e-6, "recorded cutter radius")
	require.InDelta(t, 0, math.Hypot(circle.Center.X(), circle.Center.Y()), 1e-6, "recorded cutter center")
	_, profiles, err = cap.Section(t.Context(), frame)
	require.NoError(t, err)
	require.Equal(t, 1, len(profiles))
	require.True(t, profiles[0].Valid)
	require.Equal(t, 4, len(profiles[0].Outer))
	minX, maxX, minY, maxY := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, edge := range profiles[0].Outer {
		line, ok := edge.Entity.(*sketch.Line)
		require.True(t, ok, "actual cap cutter must be rectangular")
		for _, point := range []*sketch.Point{line.Start, line.End} {
			minX, maxX = math.Min(minX, point.X()), math.Max(maxX, point.X())
			minY, maxY = math.Min(minY, point.Y()), math.Max(maxY, point.Y())
		}
	}
	require.InDelta(t, flat, minX, 1e-6, "recorded cutter flat")
	require.InDelta(t, radius+1, maxX, 1e-6)
	require.InDelta(t, -radius-1, minY, 1e-6)
	require.InDelta(t, radius+1, maxY, 1e-6)
	requireDContour(t, tool, frame, r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0), radius, flat)
	u, v := r3.NewVec(math.Cos(angle), 0, math.Sin(angle)), r3.NewVec(-math.Sin(angle), 0, math.Cos(angle))
	plane, err := r3.NewFrame(r3.NewVec(center.X, y, center.Z), u, v)
	require.NoError(t, err)
	requireDContour(t, body, plane, r3.NewVec(center.X, y, center.Z), u, v, radius, flat)
}

func requireDContour(t *testing.T, body *decad.Body, plane r3.Frame, center, u, v r3.Vec, radius, flat float64) {
	t.Helper()
	outline, err := body.SectionOutline(t.Context(), plane, units.Millimeters(.01))
	require.NoError(t, err)
	require.Less(t, outline.OffSurface.Base(), .01, "contour error must fit the dimensional check")
	found := 0
	for _, loop := range outline.Loops {
		minX, maxX, maxRadius := math.Inf(1), math.Inf(-1), 0.0
		for _, point := range loop {
			delta := point.Sub(center)
			x, z := delta.Dot(u), delta.Dot(v)
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			maxRadius = math.Max(maxRadius, math.Hypot(x, z))
		}
		if maxRadius > radius+.02 {
			continue
		}
		found++
		require.InDelta(t, -radius, minX, .01, "D contour circular side")
		require.InDelta(t, flat, maxX, .01, "D contour flat")
		require.InDelta(t, radius, maxRadius, .01, "D contour radius")
	}
	require.Equal(t, 1, found, "one actual D contour")
}

func requireRoundJournal(t *testing.T, history *decad.OperationHistory, body *decad.Body, center r3.Vec, y, radius, sourceZ float64) {
	t.Helper()
	tool := cutInput(t, recordedCut(t, history, body), "tool")
	requireToolMissesDepth(t, history, tool, sourceZ, 0)
	plane, err := r3.NewFrame(r3.NewVec(0, y, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	section, err := body.SectionOutline(t.Context(), plane, units.Millimeters(.01))
	require.NoError(t, err)
	require.Less(t, section.OffSurface.Base(), .01)
	require.Equal(t, 1, len(section.Loops))
	for _, point := range section.Loops[0] {
		require.InDelta(t, radius, math.Hypot(point.X-center.X, point.Z-center.Z), .01, "actual journal stays round")
	}
	require.InDelta(t, math.Pi*radius*radius, section.PolygonArea().Base(), .15, "full circular journal area")
}

func requireToolMissesDepth(t *testing.T, history *decad.OperationHistory, tool *decad.Body, z float64, depth int) {
	t.Helper()
	require.Less(t, depth, 4, "flat-tool union nesting")
	box, err := tool.Bounds()
	require.NoError(t, err)
	if z < box.Min.Z-box.Bound.Base() || z > box.Max.Z+box.Bound.Base() {
		return
	}
	op, err := history.Producer(tool)
	require.NoError(t, err)
	require.Equal(t, decad.OperationUnion, op.Kind(), "a flat tool must not intersect the bearing journal")
	inputs := op.Inputs()
	require.GreaterOrEqual(t, len(inputs), 2)
	for _, input := range inputs {
		requireToolMissesDepth(t, history, input.Body, z, depth+1)
	}
}
