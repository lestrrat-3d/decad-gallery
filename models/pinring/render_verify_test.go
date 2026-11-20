package pinring_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pinring"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestDefaultRingVerifyAllAtRenderTolerance tessellates the default ring at 0.01 mm under VerifyAll. The renderer
// meshes it without verification at that tolerance.
func TestDefaultRingVerifyAllAtRenderTolerance(t *testing.T) {
	m, err := pinring.Build(t.Context(), decad.New(), pinring.DefaultParams())
	require.NoError(t, err)
	_, err = m.Body.Tessellate(t.Context(), units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
}
