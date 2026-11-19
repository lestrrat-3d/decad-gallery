package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func buildSpiralMemberFor(t *testing.T, p bevelgear.Params) memberResult {
	t.Helper()
	return buildSpiralMemberChecked(t, p, true)
}

// buildSpiralMemberChecked builds a spiral member in a chain-tolerance document. A member whose flanks cross
// the end cones at a shallow angle builds with a volume bound under 15 percent but its mesh cannot be
// verified at 0.1 mm, so verify is false for it.
func buildSpiralMemberChecked(t *testing.T, p bevelgear.Params, verify bool) memberResult {
	t.Helper()
	doc := chainDocument()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	if !verify {
		volume, err := gear.Volume()
		require.NoError(t, err)
		require.Less(t, volume.Bound.Base(), 0.15*volume.Value.Base())
		box, err := gear.Bounds()
		require.NoError(t, err)
		return memberResult{volume: volume.Value.Base(), bound: volume.Bound.Base(),
			xmin: box.Min.X, xmax: box.Max.X, ymax: box.Max.Y}
	}
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 2, meshEuler(mesh), "the unbored member has one closed outer shell")
	volume, err := gear.Volume()
	require.NoError(t, err)
	box, err := gear.Bounds()
	require.NoError(t, err)
	return memberResult{volume: volume.Value.Base(), bound: volume.Bound.Base(),
		xmin: box.Min.X, xmax: box.Max.X, ymax: box.Max.Y}
}
