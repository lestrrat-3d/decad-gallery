package spurgear_test

import (
	"testing"

	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

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
