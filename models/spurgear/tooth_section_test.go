package spurgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestToothSectionSignedRotation(t *testing.T) {
	p := spurgear.DefaultParams()
	p.Teeth, p.Samples, p.Thickness = 8, 5, 2
	world := sketch.NewWorld()
	frame, err := r3.NewFrame(r3.NewVec(0, 0, p.Thickness), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	upper, err := world.CreatePlaneFromFrame(frame)
	require.NoError(t, err)
	lowerSketch, lower, err := spurgear.BuildToothSection(t.Context(), world, world.XY(), p, 0)
	require.NoError(t, err)
	requireSectionSketchFullyConstrained(t, lowerSketch)
	require.Len(t, lower.Outer, 6)
	lx, ly := sectionCentroid(lower)
	require.Greater(t, lx, 0.0)
	for _, angle := range []float64{14.5 * math.Pi / 180, -14.5 * math.Pi / 180} {
		topSketch, top, err := spurgear.BuildToothSection(t.Context(), world, upper, p, angle)
		require.NoError(t, err)
		requireSectionSketchFullyConstrained(t, topSketch)
		require.Len(t, top.Outer, 6)
		x, y := sectionCentroid(top)
		require.InDelta(t, lx*math.Cos(angle)-ly*math.Sin(angle), x, 1e-6)
		require.InDelta(t, lx*math.Sin(angle)+ly*math.Cos(angle), y, 1e-6)
	}
}

func requireSectionSketchFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Zero(t, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.NoError(t, report.Check())
}

func sectionCentroid(profile *sketch.Profile) (float64, float64) {
	var x, y float64
	count := 0
	for _, edge := range profile.Outer {
		for _, point := range edge.Polyline {
			x += point[0]
			y += point[1]
			count++
		}
	}
	return x / float64(count), y / float64(count)
}
