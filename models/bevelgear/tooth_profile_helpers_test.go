package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.Equal(t, sketch.FullyConstrained, report.Status)
	require.Zero(t, report.DOF)
	require.Empty(t, report.Redundant)
	require.Empty(t, report.Conflicts)
	require.NoError(t, report.Check())
}
