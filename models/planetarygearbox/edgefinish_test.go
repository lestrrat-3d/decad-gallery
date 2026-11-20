package planetarygearbox_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

// TestEdgeFinishing checks Surface edge finishing on the ring, covers and carrier: a 45° chamfer of EdgeChamfer
// along the outer boundary of each large flat face.
func TestEdgeFinishing(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	require.InDelta(t, 0.5, p.EdgeChamfer, 0)
	require.GreaterOrEqual(t, p.EdgeChamfer, planetarygearbox.MinEdgeChamfer)
	_, box := defaultGearbox(t)
	requireEdgeFinishing(t, box, p)
}

// TestEdgeChamferLimits checks that a chamfer below the edge finishing minimum is refused.
func TestEdgeChamferLimits(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	p.EdgeChamfer = planetarygearbox.MinEdgeChamfer - 0.01
	require.ErrorContains(t, p.Validate(), "edge chamfer")
	p.EdgeChamfer = 1.01
	require.ErrorContains(t, p.Validate(), "edge chamfer")
	p.EdgeChamfer = 1
	require.NoError(t, p.Validate())
}
