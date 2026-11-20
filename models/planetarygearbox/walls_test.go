package planetarygearbox_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

// TestSunBoreWallDimensions requires, for every kit, the sun's bore diameter and root radius to leave 1.6 mm of wall.
// It builds nothing, so it covers the kits that TestKitGearboxes skips.
func TestSunBoreWallDimensions(t *testing.T) {
	for _, kit := range planetarygearbox.SupportedCombinations() {
		t.Run(kit.Name, func(t *testing.T) {
			p, err := planetarygearbox.DefaultParamsFor(kit.Name)
			require.NoError(t, err)
			require.Empty(t, planetarygearbox.Problems(p))
			// The root circle of the sun is the pitch radius less the module and the root clearance (0.5 mm at module 1).
			root := p.Module*float64(p.SunTeeth)/2 - p.Module - math.Max(0.25*p.Module, 0.5)
			require.GreaterOrEqual(t, root-planetarygearbox.SunBoreDiameter(p)/2, minPressWall-1e-9)
		})
	}
}
