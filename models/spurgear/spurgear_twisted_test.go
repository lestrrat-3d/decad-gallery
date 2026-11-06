package spurgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestTwistedProfileRootContacts(t *testing.T) {
	for _, tc := range []struct {
		name      string
		params    spurgear.Params
		tolerance float64
	}{
		{"ordinary", spurgear.Params{Module: 1, Teeth: 8, PressureAngle: 20 * math.Pi / 180, Thickness: 2, Samples: 5}, 1e-9},
		{"ordinary with backlash", spurgear.Params{Module: 1, Teeth: 8, PressureAngle: 20 * math.Pi / 180,
			Thickness: 2, Samples: 5, Backlash: 0.05}, 1e-9},
		{"embedded", spurgear.Params{Module: 0.5, Teeth: 24, PressureAngle: 30 * math.Pi / 180, Thickness: 2, Samples: 5}, 1e-8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.params
			root := (p.Module*float64(p.Teeth) - 2.5*p.Module) / 2
			const twist = 0.2
			var contacts [2][][2]float64
			for i, angle := range []float64{0, twist} {
				world := sketch.NewWorld()
				s, profile, roots, err := spurgear.BuildTwistedProfile(t.Context(), world, world.XY(), p, angle)
				require.NoError(t, err)
				require.True(t, profile.Valid)
				require.Empty(t, profile.Holes)
				report := s.Verify(t.Context())
				require.Equal(t, sketch.FullyConstrained, report.Status)
				require.Zero(t, report.DOF)
				require.Empty(t, report.Redundant)
				require.Empty(t, report.Conflicts)
				require.Len(t, roots, 2*p.Teeth)
				for _, r := range roots {
					require.InDelta(t, root, math.Hypot(r[0], r[1]), 1e-7)
				}
				contacts[i] = roots
			}
			sin, cos := math.Sincos(twist)
			for i, r := range contacts[0] {
				require.InDelta(t, r[0]*cos-r[1]*sin, contacts[1][i][0], tc.tolerance, "contact %d x", i)
				require.InDelta(t, r[0]*sin+r[1]*cos, contacts[1][i][1], tc.tolerance, "contact %d y", i)
			}
		})
	}
}

func TestTwistedProfileRejectsBadInputs(t *testing.T) {
	p := spurgear.DefaultParams()
	world := sketch.NewWorld()
	_, _, _, err := spurgear.BuildTwistedProfile(t.Context(), world, world.XY(), p, math.NaN())
	require.Error(t, err)
	_, _, _, err = spurgear.BuildTwistedProfile(t.Context(), world, nil, p, 0)
	require.Error(t, err)
	_, _, _, err = spurgear.BuildTwistedProfile(t.Context(), nil, world.XY(), p, 0)
	require.Error(t, err)
	p.Module = 0
	_, _, _, err = spurgear.BuildTwistedProfile(t.Context(), world, world.XY(), p, 0)
	require.Error(t, err)
}

func TestDefineParametersWritesTheSpurTable(t *testing.T) {
	doc := decad.New()
	p := spurgear.DefaultParams()
	require.NoError(t, spurgear.DefineParameters(doc.Params(), p))
	for name, want := range map[string]float64{
		"module": p.Module, "backlash": p.Backlash, "pitch_radius": p.Module * float64(p.Teeth) / 2,
	} {
		value, err := doc.Param(name)
		require.NoError(t, err, name)
		require.InDelta(t, want, value.Base(), 1e-12, name)
	}
	teeth, err := doc.ParamInt("teeth")
	require.NoError(t, err)
	require.Equal(t, p.Teeth, teeth)
}
