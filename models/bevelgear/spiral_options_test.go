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
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
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

// A spiral cutter radius other than the mean cone distance changes the tooth curve, not the blank.
func TestSpiralCutterRadiusBuildsMember(t *testing.T) {
	t.Parallel()
	spiral := func() bevelgear.Params {
		p := bevelgear.DefaultParams()
		p.EnableBore = false
		return p
	}
	var base memberResult
	t.Run("default", func(t *testing.T) {
		base = buildSpiralMemberFor(t, spiral())
	})
	t.Run("cutter_30", func(t *testing.T) {
		p := spiral()
		p.CutterRadius = 30
		got := buildSpiralMemberFor(t, p)
		require.InDelta(t, base.xmax, got.xmax, 1e-6)
		require.InDelta(t, base.xmin, got.xmin, 0.01, "the toe edge shifts by the trace, not by a cone")
		require.NotEqual(t, base.volume, got.volume, "a different cutter circle moves the flanks")
	})
}

// The pair mirrors the identical pinion for the driving gear, so one cutter radius serves both members.
func TestSpiralPairCutterRadius(t *testing.T) {
	t.Parallel()
	p := bevelgear.DefaultParams()
	p.EnableBore = false
	p.CutterRadius = 30
	doc := decad.New()
	pair, err := bevelgear.BuildPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	for name, body := range map[string]*decad.Body{"driving": pair.Driving, "pinion": pair.Pinion} {
		require.True(t, body.IsSolid(), name)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		require.True(t, mesh.VolumeVerified(), name)
	}
}

// Left hand, face width and base height change the tooth trace and the blank. Each member and pair builds a verified
// closed shell, and a narrower face moves the toe end toward the heel.
func TestSpiralHandWidthAndBaseHeight(t *testing.T) {
	t.Parallel()
	spiral := func() bevelgear.Params {
		p := bevelgear.DefaultParams()
		p.EnableBore = false
		return p
	}
	var base memberResult
	t.Run("default", func(t *testing.T) {
		base = buildSpiralMemberFor(t, spiral())
	})
	t.Run("members", func(t *testing.T) {
		t.Run("left_hand", func(t *testing.T) {
			t.Parallel()
			p := spiral()
			p.SpiralHand = "left"
			got := buildSpiralMemberFor(t, p)
			require.InDelta(t, base.xmin, got.xmin, 0.01)
			require.InDelta(t, base.xmax, got.xmax, 1e-6)
		})
		t.Run("base_height_2", func(t *testing.T) {
			t.Parallel()
			p := spiral()
			p.BaseHeight = 2
			got := buildSpiralMemberFor(t, p)
			require.InDelta(t, base.xmax-(31.0/8-2), got.xmax, 1e-6,
				"a base height of 2 instead of the default 31/8 lowers the heel end by 31/8 - 2")
			require.Less(t, got.volume, base.volume)
		})
		t.Run("face_width_6", func(t *testing.T) {
			t.Parallel()
			p := spiral()
			p.FaceWidth = 6
			got := buildSpiralMemberFor(t, p)
			require.InDelta(t, base.xmax, got.xmax, 1e-6, "the heel stays put")
			require.Greater(t, got.xmin, base.xmin, "a narrower face ends the toe closer to the heel")
			require.Less(t, got.volume, base.volume)
		})
	})
}

func TestSpiralPairHandWidthAndBaseHeight(t *testing.T) {
	t.Parallel()
	for name, mod := range map[string]func(p *bevelgear.Params){
		"left_hand":    func(p *bevelgear.Params) { p.SpiralHand = "left" },
		"face_width_6": func(p *bevelgear.Params) { p.FaceWidth = 6 },
		"base_heights": func(p *bevelgear.Params) { p.BaseHeight, p.MateBaseHeight = 2, 2.5 },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := bevelgear.DefaultParams()
			p.EnableBore = false
			mod(&p)
			doc := decad.New()
			pair, err := bevelgear.BuildPair(t.Context(), doc, p)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 2)
			for member, body := range map[string]*decad.Body{"driving": pair.Driving, "pinion": pair.Pinion} {
				require.True(t, body.IsSolid(), member)
				mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, member)
				require.True(t, mesh.BoundaryVerified(), member)
				require.True(t, mesh.VolumeVerified(), member)
			}
		})
	}
}
