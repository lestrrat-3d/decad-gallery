package hexnut_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

var (
	m2 = hexnut.Size{Thread: 2, AcrossFlats: 4, Thickness: 1.6, BearingDiameter: 3.48}
	m5 = hexnut.Size{Thread: 5, AcrossFlats: 8, Thickness: 4.7, BearingDiameter: 6.9}
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}
