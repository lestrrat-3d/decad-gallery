package spiralspring_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSizeComparisonVerifyAll tessellates the default, small and large springs of the size comparison image at 0.05 mm
// under VerifyAll. The renderer meshes them without verification.
func TestSizeComparisonVerifyAll(t *testing.T) {
	small := spiralspring.DefaultParams()
	small.OuterDiameter, small.HubDiameter, small.Thickness, small.MinGap, small.Height = 40, 12, 1.5, 1.5, 5
	large := spiralspring.DefaultParams()
	large.OuterDiameter, large.HubDiameter, large.Thickness, large.MinGap, large.Height = 90, 16, 2, 2.5, 10
	for name, p := range map[string]spiralspring.Params{"default": spiralspring.DefaultParams(), "small": small, "large": large} {
		t.Run(name, func(t *testing.T) {
			m, err := spiralspring.Build(t.Context(), decad.New(), p)
			require.NoError(t, err)
			_, err = m.Body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
		})
	}
}
