package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

type memberResult struct {
	volume float64
	bound  float64
	xmin   float64
	xmax   float64
	ymax   float64
}

func buildStraightMemberFor(t *testing.T, p bevelgear.Params) memberResult {
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

func optionBase() bevelgear.Params {
	return bevelgear.Params{Module: 1, Teeth: 8, MateTeeth: 8, ShaftAngle: math.Pi / 2,
		PressureAngle: 20 * math.Pi / 180, Samples: 5}
}

// Each option of the Fusion add-in builds a verified closed member, and moves the geometry the way the
// option says. Lengths are millimeters; the baseline is a Module 1 member with 8 teeth, whose default base
// height is 1.
func TestStraightOptionsBuildMembers(t *testing.T) {
	t.Parallel()
	var base memberResult
	t.Run("baseline", func(t *testing.T) {
		base = buildStraightMemberFor(t, optionBase())
		require.InDelta(t, 5, base.xmax, 1e-6, "the default base height 1 ends the blank at x = 5")
	})
	cases := []struct {
		name   string
		mod    func(p *bevelgear.Params)
		verify func(t *testing.T, got memberResult)
	}{
		{"base_height", func(p *bevelgear.Params) { p.BaseHeight = 2 }, func(t *testing.T, got memberResult) {
			require.InDelta(t, base.xmax+1, got.xmax, 1e-6, "a base height of 2 raises the heel end by 1")
			require.InDelta(t, base.xmin, got.xmin, 1e-6)
			require.Greater(t, got.volume, base.volume)
		}},
		{"toe_extension", func(p *bevelgear.Params) { p.ToeExtension = 20 }, func(t *testing.T, got memberResult) {
			require.Less(t, got.xmin, base.xmin, "a toe extension reaches closer to the apex")
			require.InDelta(t, base.xmax, got.xmax, 1e-6)
			require.Greater(t, got.volume, base.volume)
		}},
		{"tooth_spacing", func(p *bevelgear.Params) { p.ToothSpacing = 0.05 }, func(t *testing.T, got memberResult) {
			require.Less(t, got.ymax, base.ymax, "a spacing offset thins the tooth")
			require.Less(t, got.volume, base.volume)
		}},
		{"toe_radius", func(p *bevelgear.Params) { p.ToeRadius = 0.5 }, func(t *testing.T, got memberResult) {
			require.InDelta(t, base.xmax, got.xmax, 1e-6)
			require.Less(t, got.volume, base.volume)
		}},
		{"pressure_angle", func(p *bevelgear.Params) { p.PressureAngle = 14.5 * math.Pi / 180 }, func(t *testing.T, got memberResult) {
			require.Less(t, got.volume, base.volume)
		}},
		{"teeth_20_20", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth, p.Samples = 20, 20, 15 }, func(t *testing.T, got memberResult) {
			require.InDelta(t, base.ymax+6, got.ymax, 1e-3, "the pitch radius grows from 4 to 10")
		}},
	}
	t.Run("options", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				p := optionBase()
				tc.mod(&p)
				tc.verify(t, buildStraightMemberFor(t, p))
			})
		}
	})
}

func TestStraightBuildRejectsOtherShaftAngles(t *testing.T) {
	t.Parallel()
	p := optionBase()
	p.ShaftAngle = math.Pi / 3
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "90° shaft angle")
	require.Empty(t, doc.Bodies())
}

// The pair takes the driving member's options and the mate's options separately, and each member's
// shaft keeps its own base height. The pinion's shaft runs along +Y after placement.
func TestStraightPairMateOptions(t *testing.T) {
	t.Parallel()
	p := optionBase()
	p.BaseHeight, p.MateBaseHeight = 2, 2.5
	p.MateToeRadius = 0.5
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
	drivingBox, err := pair.Driving.Bounds()
	require.NoError(t, err)
	pinionBox, err := pair.Pinion.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 6, drivingBox.Max.X, 1e-6, "the driving base height 2 ends its shaft at x = 6")
	require.InDelta(t, 6.5, pinionBox.Max.Y, 1e-6, "the mate base height 2.5 ends the pinion shaft at y = 6.5")
}
