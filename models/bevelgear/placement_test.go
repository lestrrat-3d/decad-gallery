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
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	p.SpiralAngle = 0
	p.EnableBore = false
	return p
}

// sideFrame is a target plane at y=-5 with its normal on +Y, the centre point at x=10, z=7, and the
// pinion leaning toward +Z.
func sideFrame(t *testing.T) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 0, 1), r3.NewVec(1, 0, 0))
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
	// The back face is the member's far station, so it lands flat on the plane y=-5 and the
	// gear stands toward +Y with its axis through x=10, z=7.
	require.InDelta(t, -5, box.Min.Y, tol)
	require.InDelta(t, -5+worldBox.Max.X-worldBox.Min.X, box.Max.Y, tol)
	require.InDelta(t, 10-worldBox.Max.Z, box.Min.X, tol)
	require.InDelta(t, 10-worldBox.Min.Z, box.Max.X, tol)
	require.InDelta(t, 7+worldBox.Min.Y, box.Min.Z, tol)
	require.InDelta(t, 7+worldBox.Max.Y, box.Max.Z, tol)
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
	// The driving gear stands on the plane y=-5 along its normal. Turning the member about its
	// shaft for its half-pitch phase keeps its extent along the shaft.
	box, err := pair.Driving.Bounds()
	require.NoError(t, err)
	tol := memberBox.Bound.Base() + box.Bound.Base() + 1e-8
	require.InDelta(t, -5, box.Min.Y, tol)
	require.InDelta(t, -5+memberBox.Max.X-memberBox.Min.X, box.Max.Y, tol)
	// At a 90° shaft angle the pinion shaft runs along U (+Z) from the apex.
	pinionBox, err := pair.Pinion.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 7, (box.Min.Z+box.Max.Z)/2, 0.01)
	require.Greater(t, (pinionBox.Min.Z+pinionBox.Max.Z)/2, 7+1.0)
}
