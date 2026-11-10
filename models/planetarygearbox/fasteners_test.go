package planetarygearbox_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

// TestFastenersSitFlush checks that four M2 bolts and four nuts join each cover to the ring, that every bolt head
// ends level with its cover's outer face, and that every nut sits inside the ring, 1 mm from its end face.
func TestFastenersSitFlush(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
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
	// A nut sits in a closed cavity one millimeter inside the ring's end face, so the ring itself holds it.
	for i, nut := range box.Nuts {
		require.True(t, nut.IsSolid(), "nut %d", i)
		b := bounds(t, nut)
		require.InDelta(t, m2nut.Thickness, b.Max.Z-b.Min.Z, 2e-3, "nut %d thickness", i)
		if i < 4 {
			require.InDelta(t, -p.StageGap+1.2, b.Min.Z, 2e-3, "front nut %d is inside the ring", i)
			continue
		}
		require.InDelta(t, ringEnd-1.2, b.Max.Z, 2e-3, "rear nut %d is inside the ring", i)
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

// TestFastenersFitTheirPockets checks that no bolt or nut occupies material of the parts it joins: each bolt runs
// through its cover's hole and head pocket and the ring's hole, and each nut sits in a ring cavity.
func TestFastenersFitTheirPockets(t *testing.T) {
	// The overlap checks copy bodies into their document, so they use a gearbox of their own.
	box, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), planetarygearbox.DefaultParams())
	require.NoError(t, err)
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
}

func TestFastenerFitRules(t *testing.T) {
	for name, c := range map[string]struct {
		edit func(*planetarygearbox.Params)
		want string
	}{
		"hole narrower than the shank": {func(p *planetarygearbox.Params) { p.BoltHoleDiameter = 1.5 }, "shank"},
		"hole under the M2 clearance":  {func(p *planetarygearbox.Params) { p.BoltHoleDiameter = 2.5 }, "clearance hole"},
		"cover too thin for the head":  {func(p *planetarygearbox.Params) { p.CoverThickness = 2.5 }, "bolt head"},
	} {
		t.Run(name, func(t *testing.T) {
			p := planetarygearbox.DefaultParams()
			c.edit(&p)
			_, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
			require.ErrorContains(t, err, c.want)
		})
	}
}
