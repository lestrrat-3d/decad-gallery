package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBuildRejectsUnverifiedFullPatternCaseBeforeAddingBodies(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth = 30
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "source-default 31/31")
	require.Empty(t, doc.Bodies())
}

func TestSourceFourFourStraightMember(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 4, 4
	p.SpiralAngle = 0
	p.EnableBore = false
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 2, meshEuler(mesh), "the unbored member has one closed outer shell")
}

func TestSourceFourFourStraightMemberAutoBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 4, 4
	p.SpiralAngle = 0
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh), "the automatic bore pierces the member")
}

func TestSourceFourFourStraightPairAutoBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 4, 4
	p.SpiralAngle = 0
	doc := decad.New()
	pair, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	for name, gear := range map[string]*decad.Body{"driving": pair.Driving, "pinion": pair.Pinion} {
		require.True(t, gear.IsSolid(), name)
		mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		require.True(t, mesh.VolumeVerified(), name)
		require.Equal(t, 0, meshEuler(mesh), "%s has one through-hole", name)
	}
}

func TestSourceSixteenTwelveStraightPinion(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 12, 16
	p.SpiralAngle = 0
	p.EnableBore = false
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 2, meshEuler(mesh), "the unbored pinion has one closed outer shell")
}

func TestSourceSixteenTwelveStraightDriving(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 16, 12
	p.SpiralAngle = 0
	p.EnableBore = false
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 2, meshEuler(mesh), "the unbored driving gear has one closed outer shell")
}

func TestSourceSixteenTwelveStraightPair(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 16, 12
	p.SpiralAngle = 0
	p.EnableBore = false
	doc := decad.New()
	pair, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	for name, gear := range map[string]*decad.Body{"driving": pair.Driving, "pinion": pair.Pinion} {
		require.True(t, gear.IsSolid(), name)
		mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		require.True(t, mesh.VolumeVerified(), name)
		require.Equal(t, 2, meshEuler(mesh), "%s has one closed outer shell", name)
	}
}

func TestSourceSixteenTwelveStraightPinionAutoBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 12, 16
	p.SpiralAngle = 0
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh), "the automatic bore pierces the pinion")
}

func TestSourceSixteenTwelveStraightDrivingAutoBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 16, 12
	p.SpiralAngle = 0
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 0, meshEuler(mesh), "the automatic bore pierces the driving member")
}

func TestSourceSixteenTwelveStraightPairAutoBore(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module = 4
	p.Teeth, p.MateTeeth = 16, 12
	p.SpiralAngle = 0
	doc := decad.New()
	pair, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	for name, gear := range map[string]*decad.Body{"driving": pair.Driving, "pinion": pair.Pinion} {
		require.True(t, gear.IsSolid(), name)
		mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		require.True(t, mesh.VolumeVerified(), name)
		require.Equal(t, 0, meshEuler(mesh), "%s has one through-hole", name)
	}
}
