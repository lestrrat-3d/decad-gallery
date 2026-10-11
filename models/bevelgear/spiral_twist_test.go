package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralToothHeelFaceTwist(t *testing.T) {
	finished := make(map[string][]r3.Vec)
	for _, member := range []struct {
		name   string
		pinion bool
	}{
		{name: "driving"},
		{name: "pinion", pinion: true},
	} {
		t.Run(member.name, func(t *testing.T) {
			p := DefaultParams()
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
			before := make([]r3.Vec, len(slabs))
			for i, slab := range slabs {
				_, heel, err := slab.ExtremeFacesByCentroid(t.Context(), frame.coneVec)
				require.NoError(t, err, "slab %d", i)
				center, err := heel.Centroid()
				require.NoError(t, err, "slab %d", i)
				before[i] = center.Value
			}
			twisted, err := twistSpiralSlabs(t.Context(), slabs, frame, trace)
			require.NoError(t, err)
			require.Len(t, twisted, 8)
			require.Len(t, doc.Bodies(), 8)
			centers := make([]r3.Vec, len(twisted))
			for i, slab := range twisted {
				_, heel, err := slab.ExtremeFacesByCentroid(t.Context(), frame.coneVec)
				require.NoError(t, err, "twisted slab %d", i)
				center, err := heel.Centroid()
				require.NoError(t, err, "twisted slab %d", i)
				centers[i] = center.Value
				angle := -trace.handSign * trace.total *
					(frame.mean - before[i].Dot(frame.coneVec)) / frame.span
				turn, err := r3.RotationAround(r3.Vec{}, r3.NewVec(1, 0, 0), units.Radians(angle))
				require.NoError(t, err)
				expected := turn.Apply(before[i])
				require.LessOrEqual(t, center.Value.Sub(expected).Len(),
					center.Bound.Base()+0.02, "twisted slab %d heel center", i)
				mesh, err := slab.Tessellate(t.Context(), units.Millimeters(0.1),
					decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, "twisted slab %d", i)
				require.True(t, mesh.BoundaryVerified(), "twisted slab %d", i)
				require.True(t, mesh.VolumeVerified(), "twisted slab %d", i)
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
