package spinningwheel_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// Keep the first real producer/consumer boundary separate from dimensional and clearance assertions.
func TestSpinningWheelNativeDefault(t *testing.T) {
	doc := spinningwheel.NewDocument()
	a, err := spinningwheel.Build(t.Context(), doc, spinningwheel.DefaultParams())
	if err != nil {
		operations := doc.OperationHistory().Operations()
		coils := 0
		for _, operation := range operations {
			if operation.Kind() == decad.OperationCoil {
				coils++
			}
		}
		t.Logf("Build failed after %d public operations, %d Coil operations, and %d live bodies",
			len(operations), coils, len(doc.Bodies()))
	}
	require.NoError(t, err)
	require.Equal(t, len(a.Parts), len(doc.Bodies()))
	requirePartHomes(t, a)
	for _, part := range a.Parts {
		mesh, err := part.Body.Tessellate(t.Context(), units.Millimeters(.12), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, "%s", part.Name)
		require.True(t, mesh.BoundaryVerified(), "%s", part.Name)
		require.True(t, mesh.VolumeVerified(), "%s", part.Name)
		require.Equal(t, 1, meshComponents(len(mesh.Vertices()), mesh.Triangles()), "%s must be one connected physical part", part.Name)
	}
	for i, s := range a.Sketches {
		r := s.Verify(t.Context())
		require.True(t, r.Status == sketch.FullyConstrained, "sketch %d status %s", i, r.Status)
		require.Zero(t, r.DOF, "sketch %d", i)
		require.Zero(t, len(r.Redundant), "sketch %d", i)
		require.Zero(t, len(r.Conflicts), "sketch %d", i)
		require.NoError(t, r.Check(), "sketch %d", i)
	}
	t.Logf("verified %d actual bodies and %d sketches", len(a.Parts), len(a.Sketches))
}
