package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralToothNineFaceLoft(t *testing.T) {
	for _, member := range []struct {
		name   string
		pinion bool
	}{
		{name: "driving"},
		{name: "pinion", pinion: true},
	} {
		t.Run(member.name, func(t *testing.T) {
			p := spiralDefaults()
			p.SpiralAngle = 35 * math.Pi / 180
			p.SpiralHand = "right"
			straight := p
			straight.SpiralAngle = 0
			doc := decad.New()
			tooth, err := BuildPointTooth(t.Context(), doc, straight)
			require.NoError(t, err)
			slabs, frame, err := sliceSpiralPointTooth(t.Context(), doc, p, tooth)
			require.NoError(t, err)
			trace, err := buildSpiralTrace(frame, p, member.pinion)
			require.NoError(t, err)
			twisted, err := twistSpiralSlabs(t.Context(), slabs, frame, trace)
			require.NoError(t, err)
			crowned, err := crownSpiralSlabs(t.Context(), twisted, frame, trace)
			require.NoError(t, err)
			faces, err := spiralLoftFaces(t.Context(), crowned, frame)
			require.NoError(t, err)
			require.Len(t, faces, 9)
			curved, err := decad.LoftFaces(t.Context(), doc, faces...)
			require.NoError(t, err)
			require.True(t, curved.IsSolid())
			mesh, err := curved.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}
