package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

// requireFastenersFlush checks the bolts and nuts of the built gearbox box against its covers and ring.
func requireFastenersFlush(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	require.Len(t, box.Bolts, 8)
	require.Len(t, box.Nuts, 8)

	ringEnd := bounds(t, box.Ring).Max.Z
	frontFace := -p.StageGap - p.CoverThickness
	rearFace := ringEnd + p.CoverThickness
	for i, bolt := range box.Bolts {
		require.True(t, bolt.IsSolid(), "bolt %d", i)
		b := bounds(t, bolt)
		if i < 4 {
			require.InDelta(t, frontFace, b.Min.Z, 2e-3, "front bolt %d head is flush", i)
			require.InDelta(t, frontFace+m2bolt.HeadHeight+6, b.Max.Z, 2e-3, "front bolt %d tip", i)
			continue
		}
		require.InDelta(t, rearFace, b.Max.Z, 2e-3, "rear bolt %d head is flush", i)
		require.InDelta(t, rearFace-m2bolt.HeadHeight-6, b.Min.Z, 2e-3, "rear bolt %d tip", i)
	}
	// A nut sits in a closed cavity 1.8 mm inside the ring's end face, 0.2 mm off the cavity's floor, so the ring
	// itself holds it.
	for i, nut := range box.Nuts {
		require.True(t, nut.IsSolid(), "nut %d", i)
		b := bounds(t, nut)
		require.InDelta(t, m2nut.Thickness, b.Max.Z-b.Min.Z, 2e-3, "nut %d thickness", i)
		if i < 4 {
			require.InDelta(t, -p.StageGap+2.0, b.Min.Z, 2e-3, "front nut %d is inside the ring", i)
			continue
		}
		require.InDelta(t, ringEnd-2.0, b.Max.Z, 2e-3, "rear nut %d is inside the ring", i)
	}
	requireEarFasteners(t, box, p)
}

// requireEarFasteners checks the bolt and the nut of each mounting ear against the ring's ends: the head ends level
// with the ear's rear face, the tip stays inside the ear, the bolt is a standard length, and the nut sits in its
// cavity, 0.2 mm off the floor, wholly between the head and the tip, so the bolt runs through it.
func requireEarFasteners(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	require.Len(t, box.EarBolts, p.EarCount)
	require.Len(t, box.EarNuts, p.EarCount)
	ring := bounds(t, box.Ring)
	length := ring.Max.Z - ring.Min.Z
	bolt := earBoltLength(length)
	require.Contains(t, m2bolt.StandardLengths(), bolt)
	outer := ringOuterRadius(p)
	center := outer + p.EarBoreDiameter/2 + earWall
	for i := range p.EarCount {
		b := bounds(t, box.EarBolts[i])
		require.True(t, box.EarBolts[i].IsSolid(), "ear bolt %d", i+1)
		require.InDelta(t, ring.Max.Z, b.Max.Z, 2e-3, "ear bolt %d head is flush with the ear's rear face", i+1)
		require.InDelta(t, ring.Max.Z-m2bolt.HeadHeight-bolt, b.Min.Z, 2e-3, "ear bolt %d tip", i+1)
		require.GreaterOrEqual(t, b.Min.Z, ring.Min.Z-1e-6, "ear bolt %d tip stays inside the ear", i+1)
		angle := math.Pi/2 + 2*math.Pi*float64(i)/float64(p.EarCount)
		require.InDelta(t, center*math.Cos(angle), (b.Min.X+b.Max.X)/2, 2e-3, "ear bolt %d on the bore", i+1)
		require.InDelta(t, center*math.Sin(angle), (b.Min.Y+b.Max.Y)/2, 2e-3, "ear bolt %d on the bore", i+1)

		n := bounds(t, box.EarNuts[i])
		require.True(t, box.EarNuts[i].IsSolid(), "ear nut %d", i+1)
		require.InDelta(t, ring.Min.Z+earNutFloor(length)+0.2, n.Min.Z, 2e-3, "ear nut %d rests in its cavity", i+1)
		require.GreaterOrEqual(t, n.Min.Z, b.Min.Z-1e-6, "ear nut %d lies past the bolt's tip", i+1)
		require.LessOrEqual(t, n.Max.Z, ring.Max.Z-m2bolt.HeadHeight-1, "ear nut %d keeps 1 mm under the head pocket", i+1)
		require.GreaterOrEqual(t, n.Min.Z-0.2, ring.Min.Z+1.8-1e-6, "ear nut %d keeps its 1.8 mm roof", i+1)
	}
}

// overlapVolume returns the volume two bodies share, zero when they only touch or are apart. It works on copies,
// because a boolean consumes its operands.
func overlapVolume(t *testing.T, a, b *decad.Body) float64 {
	t.Helper()
	ca, err := decad.Copy(t.Context(), a)
	require.NoError(t, err)
	cb, err := decad.Copy(t.Context(), b)
	require.NoError(t, err)
	both, err := decad.Intersect(t.Context(), ca, cb)
	if err != nil {
		return 0
	}
	volume, err := both.Volume()
	require.NoError(t, err)
	return volume.Value.Base()
}

// requireFastenersClear checks that no bolt or nut of box overlaps the parts it passes through. It copies bodies into
// the document, so call it last.
func requireFastenersClear(t *testing.T, box planetarygearbox.Gearbox) {
	t.Helper()
	for i, bolt := range box.Bolts {
		cover := box.FrontCover
		if i >= 4 {
			cover = box.RearCover
		}
		require.Less(t, overlapVolume(t, bolt, cover), 1e-3, "bolt %d in its cover", i)
		require.Less(t, overlapVolume(t, bolt, box.Ring), 1e-3, "bolt %d in the ring", i)
	}
	for i, nut := range box.Nuts {
		require.Less(t, overlapVolume(t, nut, box.Ring), 1e-3, "nut %d in the ring", i)
	}
	for i := range box.EarBolts {
		require.Less(t, overlapVolume(t, box.EarBolts[i], box.Ring), 1e-3, "ear bolt %d in the ring", i+1)
		require.Less(t, overlapVolume(t, box.EarNuts[i], box.Ring), 1e-3, "ear nut %d in the ring", i+1)
	}
}
