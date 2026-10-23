package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const degrees = math.Pi / 180

// A spiral member takes any module, tooth counts and shaft angle. The volumes below are what the serial
// route reads for each member, and its bound is wide because each of the serial unions adds to it.
func TestSpiralInputsMembers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mod    func(p *bevelgear.Params)
		volume float64
	}{
		{"module_2", func(p *bevelgear.Params) { p.Module = 2 }, 8 * sourceOrderDefault.volume},
		{"module_0_5", func(p *bevelgear.Params) { p.Module = 0.5 }, sourceOrderDefault.volume / 8},
		{"teeth_31_25", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth = 31, 25 }, 3508.6},
		{"teeth_25_31", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth = 25, 31 }, 2325.9},
		{"teeth_40_20", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth = 40, 20 }, 5960.8},
		{"shaft_60", func(p *bevelgear.Params) { p.ShaftAngle = 60 * degrees }, 4459.3},
		{"shaft_120", func(p *bevelgear.Params) { p.ShaftAngle = 120 * degrees }, 3455.7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := serialSpiral()
			tc.mod(&p)
			doc := chainDocument()
			gear, err := bevelgear.Build(t.Context(), doc, p)
			require.NoError(t, err)
			got := verifySerialMember(t, doc, gear)
			require.InDelta(t, tc.volume, got.volume, 2*got.bound, "the volume bound of the serial unions is wide")
		})
	}
}

// The 20-tooth member of a 40/20 pair has a pitch cone so narrow that its flanks cross the end cones at a
// very shallow angle. It builds as a solid with the volume of the straight member less its spiral trim, but
// its mesh cannot be verified at 0.1 mm.
func TestSpiralInputsUnverifiedMember(t *testing.T) {
	t.Parallel()
	const straight = 1135.5
	p := serialSpiral()
	p.Teeth, p.MateTeeth = 20, 40
	doc := chainDocument()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	volume, err := gear.Volume()
	require.NoError(t, err)
	require.InDelta(t, straight, volume.Value.Base(), 0.05*straight)
	require.Less(t, volume.Bound.Base(), 0.5*volume.Value.Base())
}

// A pair of other modules, tooth counts and shaft angles builds both members with opposite hands.
func TestSpiralInputsPair(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mod  func(p *bevelgear.Params)
	}{
		{"teeth_31_25", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth = 31, 25 }},
		{"shaft_60", func(p *bevelgear.Params) { p.ShaftAngle = 60 * degrees }},
		{"module_2", func(p *bevelgear.Params) { p.Module = 2 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := serialSpiral()
			tc.mod(&p)
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
		})
	}
}
