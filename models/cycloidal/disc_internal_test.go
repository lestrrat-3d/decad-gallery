package cycloidal

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBuildDefaultDisc(t *testing.T) {
	doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
	r, err := begin(t.Context(), doc, DefaultParams())
	require.NoError(t, err)
	body, err := r.buildDisc(t.Context(), 0)
	require.NoError(t, err)
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	volume, err := body.Volume()
	require.NoError(t, err)
	box, err := body.Bounds()
	require.NoError(t, err)
	t.Logf("volume %.1f ± %.2f, bounds %+v", volume.Value.Base(), volume.Bound.Base(), box)
	// The rotor reaches the peak of the housing contour, R - Rr + 2E - c, from its own center E away.
	d := r.d
	require.Less(t, math.Max(box.Max.X, -box.Min.X), d.pinCircleRadius)
	require.InDelta(t, r.p.DiscThickness, box.Max.Z-box.Min.Z, 1e-6)
}
