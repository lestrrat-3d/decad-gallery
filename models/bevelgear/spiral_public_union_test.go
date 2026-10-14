package bevelgear

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralSerialPublicUnion(t *testing.T) {
	testSourceDefaultSpiralPublicUnion(t, true)
}

func TestSourceDefaultSpiralVariadicPublicUnion(t *testing.T) {
	testSourceDefaultSpiralPublicUnion(t, false)
}

func testSourceDefaultSpiralPublicUnion(t *testing.T, serial bool) {
	t.Helper()
	p := DefaultParams()
	geometry := p
	geometry.SpiralAngle = 0
	geometry.BoreDiameter = 0
	doc := decad.New()

	blank, err := BuildBlank(t.Context(), doc, geometry)
	require.NoError(t, err)
	blankVolume, err := blank.Volume()
	require.NoError(t, err)
	curved, err := buildSpiralTooth(t.Context(), doc, p, false)
	require.NoError(t, err)
	trimmed, err := trimSpiralTooth(t.Context(), doc, p, curved)
	require.NoError(t, err)
	pattern, err := patternSpiralTooth(t.Context(), trimmed, p.Teeth)
	require.NoError(t, err)
	require.Len(t, pattern, 31)
	require.Len(t, doc.Bodies(), 32)

	joined := blank
	if serial {
		for i, tooth := range pattern {
			joined, err = decad.Union(t.Context(), joined, tooth)
			require.NoError(t, err, "join curved tooth %d", i+1)
		}
	} else {
		// The source's Combine-Join consumes the whole circular pattern in one call.
		joined, err = decad.Union(t.Context(), blank, pattern...)
		require.NoError(t, err)
	}

	require.Len(t, doc.Bodies(), 1)
	require.True(t, joined.IsSolid())
	joinedVolume, err := joined.Volume()
	require.NoError(t, err)
	require.Greater(t, joinedVolume.Value.Base()-blankVolume.Value.Base(),
		joinedVolume.Bound.Base()+blankVolume.Bound.Base(), "the joined teeth must add certified volume")
	mesh, err := joined.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 2, spiralMeshEuler(mesh), "the unbored gear must have one closed outer shell")
}
