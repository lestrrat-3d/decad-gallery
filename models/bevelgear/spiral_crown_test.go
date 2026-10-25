package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralToothRootAnchoredCrown(t *testing.T) {
	finished := make(map[string][]r3.Vec)
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
			anchors := make([]r3.Vec, len(twisted))
			for i, slab := range twisted {
				_, heel, err := slab.ExtremeFacesByCentroid(t.Context(), frame.coneVec)
				require.NoError(t, err, "twisted slab %d", i)
				anchors[i], err = spiralHeelRootAnchor(heel)
				require.NoError(t, err, "twisted slab %d", i)
			}
			crowned, err := crownSpiralSlabs(t.Context(), twisted, frame, trace)
			require.NoError(t, err)
			require.Len(t, crowned, 8)
			require.Len(t, doc.Bodies(), 8)
			require.Same(t, twisted[7], crowned[7], "outer heel slab stays full")
			faces, err := spiralLoftFaces(t.Context(), crowned, frame)
			require.NoError(t, err)
			require.Len(t, faces, 9)
			previous := math.Inf(-1)
			for i, face := range faces {
				center, err := face.Centroid()
				require.NoError(t, err, "loft section %d", i)
				distance := center.Value.Dot(frame.coneVec)
				require.Greater(t, distance, previous, "loft section %d order", i)
				previous = distance
			}
			centers := make([]r3.Vec, len(crowned))
			for i, slab := range crowned {
				_, heel, err := slab.ExtremeFacesByCentroid(t.Context(), frame.coneVec)
				require.NoError(t, err, "crowned slab %d", i)
				center, err := heel.Centroid()
				require.NoError(t, err, "crowned slab %d", i)
				centers[i] = center.Value
				root, err := spiralHeelRootAnchor(heel)
				require.NoError(t, err, "crowned slab %d", i)
				require.Less(t, root.Sub(anchors[i]).Len(), 0.01,
					"crowned slab %d root edge anchor moved", i)
				mesh, err := slab.Tessellate(t.Context(), units.Millimeters(0.1),
					decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, "crowned slab %d", i)
				require.True(t, mesh.BoundaryVerified(), "crowned slab %d", i)
				require.True(t, mesh.VolumeVerified(), "crowned slab %d", i)
			}
			finished[member.name] = centers
		})
	}
	require.Len(t, finished["driving"], 8)
	require.Len(t, finished["pinion"], 8)
	for i := range finished["driving"] {
		driving, pinion := finished["driving"][i], finished["pinion"][i]
		require.InDelta(t, driving.X, pinion.X, 0.02, "slab %d shaft coordinate", i)
		require.InDelta(t, driving.Y, pinion.Y, 0.02, "slab %d radial coordinate", i)
		require.InDelta(t, driving.Z, -pinion.Z, 0.02, "slab %d opposite spiral hands", i)
	}
}
