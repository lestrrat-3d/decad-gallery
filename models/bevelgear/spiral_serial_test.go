package bevelgear_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// The serial route builds the members the source-order route cannot: tooth spacing, a toe extension, a
// toe radius and another tooth count. Its volume bound is wide because each of the serial unions adds to it.
func TestSpiralSerialRouteBuildsMembers(t *testing.T) {
	t.Parallel()
	t.Run("default_serial", func(t *testing.T) {
		t.Parallel()
		p := serialSpiral()
		geometry := p
		geometry.SpiralAngle = 0
		geometry.BoreDiameter = 0
		doc := chainDocument()
		gear, used, err := bevelgear.BuildSpiralMemberSerialForTest(t.Context(), doc, p, geometry, false)
		require.NoError(t, err)
		t.Logf("default overshoot %v modules", used)
		got := verifySerialMember(t, doc, gear)
		require.InDelta(t, sourceOrderDefault.volume, got.volume, got.bound+sourceOrderDefault.bound,
			"same member as the source-order route")
		require.InDelta(t, sourceOrderDefault.xmin, got.xmin, 0.01)
		require.InDelta(t, sourceOrderDefault.xmax, got.xmax, 1e-6)
	})
	cases := []struct {
		name   string
		mod    func(p *bevelgear.Params)
		volume float64
	}{
		{"tooth_spacing", func(p *bevelgear.Params) { p.ToothSpacing = 0.05 }, 3870.0},
		{"toe_extension", func(p *bevelgear.Params) { p.ToeExtension = 20 }, 4448.8},
		{"toe_radius", func(p *bevelgear.Params) { p.ToeRadius = 2 }, 3749.9},
		{"teeth_25_25", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth = 25, 25 }, 2022.7},
	}
	t.Run("options", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				p := serialSpiral()
				tc.mod(&p)
				doc := chainDocument()
				gear, err := bevelgear.Build(t.Context(), doc, p)
				require.NoError(t, err)
				got := verifySerialMember(t, doc, gear)
				require.InDelta(t, tc.volume, got.volume, got.bound, "the volume bound of the serial unions is wide")
				if tc.name == "toe_extension" {
					require.Less(t, got.xmin, sourceOrderDefault.xmin, "a toe extension reaches closer to the apex")
					require.InDelta(t, sourceOrderDefault.xmax, got.xmax, 1e-6)
				}
			})
		}
	})
}

// The pair mirrors the identical pinion for the driving gear, so one serial member serves both.
func TestSpiralSerialPair(t *testing.T) {
	t.Parallel()
	p := serialSpiral()
	p.Teeth, p.MateTeeth = 25, 25
	doc := chainDocument()
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

// A shallow pressure or spiral angle makes the flanks cross the end cones at a very shallow angle, and
// the proven bound on the held mesh grows as one over the sine of that angle. These members build as a
// solid with a volume bound under 15 percent, but their mesh cannot be verified at 0.1 mm.
func TestSpiralSerialRouteBuildsUnverifiedMembers(t *testing.T) {
	t.Parallel()
	for name, mod := range map[string]func(p *bevelgear.Params){
		"pressure_angle_14_5": func(p *bevelgear.Params) { p.PressureAngle = 14.5 * math.Pi / 180 },
		"spiral_angle_20":     func(p *bevelgear.Params) { p.SpiralAngle = 20 * math.Pi / 180 },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := serialSpiral()
			mod(&p)
			doc := chainDocument()
			gear, err := bevelgear.Build(t.Context(), doc, p)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1)
			require.True(t, gear.IsSolid())
			volume, err := gear.Volume()
			require.NoError(t, err)
			require.Greater(t, volume.Value.Base(), 3000.0)
			require.Less(t, volume.Bound.Base(), 0.15*volume.Value.Base())
		})
	}
}
