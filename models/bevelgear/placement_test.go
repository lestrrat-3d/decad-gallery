package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// smallEightParams is the smallest verified straight member: 8/8, module 1, five samples.
func smallEightParams() bevelgear.Params {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.SpiralAngle = 0
	p.EnableBore = false
	return p
}

// sideFrame maps local X to +Y, local Y to +Z, and local Z to +X, with the apex at (10, -5, 7).
func sideFrame(t *testing.T) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	return &f
}

func TestPlacedMemberMatchesWorldMember(t *testing.T) {
	p := smallEightParams()
	world, err := bevelgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	worldVolume, err := world.Volume()
	require.NoError(t, err)
	worldBox, err := world.Bounds()
	require.NoError(t, err)

	p.Placement = sideFrame(t)
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	volume, err := gear.Volume()
	require.NoError(t, err)
	require.InDelta(t, worldVolume.Value.Base(), volume.Value.Base(),
		worldVolume.Bound.Base()+volume.Bound.Base()+0.02)
	box, err := gear.Bounds()
	require.NoError(t, err)
	tol := worldBox.Bound.Base() + box.Bound.Base() + 1e-8
	require.InDelta(t, worldBox.Min.Z+10, box.Min.X, tol)
	require.InDelta(t, worldBox.Max.Z+10, box.Max.X, tol)
	require.InDelta(t, worldBox.Min.X-5, box.Min.Y, tol)
	require.InDelta(t, worldBox.Max.X-5, box.Max.Y, tol)
	require.InDelta(t, worldBox.Min.Y+7, box.Min.Z, tol)
	require.InDelta(t, worldBox.Max.Y+7, box.Max.Z, tol)
}

func TestInvalidPlacementIsRejectedBeforeAddingBodies(t *testing.T) {
	p := smallEightParams()
	p.Placement = &r3.Frame{}
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "placement")
	_, err = bevelgear.BuildPair(t.Context(), doc, p)
	require.ErrorContains(t, err, "placement")
	require.Empty(t, doc.Bodies())
}

func TestPlacedPairMovesBothMembers(t *testing.T) {
	p := smallEightParams()
	member, err := bevelgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	memberVolume, err := member.Volume()
	require.NoError(t, err)
	memberBox, err := member.Bounds()
	require.NoError(t, err)

	p.Placement = sideFrame(t)
	doc := decad.New()
	pair, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	for _, gear := range []*decad.Body{pair.Driving, pair.Pinion} {
		require.True(t, gear.IsSolid())
		volume, err := gear.Volume()
		require.NoError(t, err)
		require.InDelta(t, memberVolume.Value.Base(), volume.Value.Base(),
			memberVolume.Bound.Base()+volume.Bound.Base()+0.02)
	}
	// The driving shaft runs along the frame's U axis (+Y) from the apex at y=-5. The driving
	// member is not phased, so its extent along that shaft matches the unplaced member.
	box, err := pair.Driving.Bounds()
	require.NoError(t, err)
	tol := memberBox.Bound.Base() + box.Bound.Base() + 1e-8
	require.InDelta(t, memberBox.Min.X-5, box.Min.Y, tol)
	require.InDelta(t, memberBox.Max.X-5, box.Max.Y, tol)
}
