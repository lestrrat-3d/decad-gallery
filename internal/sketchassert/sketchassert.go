// Package sketchassert checks the gallery's Sketch constraint requirement in tests.
package sketchassert

import (
	"testing"

	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// FullyConstrained requires s to report FullyConstrained with zero degrees of freedom, no redundant
// constraints, and no conflicts. Failures report counts only, never the sketch contents.
func FullyConstrained(t testing.TB, name string, s *sketch.Sketch) {
	t.Helper()
	r := s.Verify(t.Context(), sketch.WithoutConditioning())
	require.Equal(t, sketch.FullyConstrained, r.Status, "%s: status", name)
	require.Zero(t, r.DOF, "%s: degrees of freedom", name)
	require.Zero(t, len(r.Redundant), "%s: redundant constraints", name)
	require.Zero(t, len(r.Conflicts), "%s: conflicting constraints", name)
}
