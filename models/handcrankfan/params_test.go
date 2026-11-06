package handcrankfan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require.NoError(t, handcrankfan.DefaultParams().Validate())
	for name, edit := range map[string]func(*handcrankfan.Params){
		"nan angle":      func(p *handcrankfan.Params) { p.CrankAngle = math.NaN() },
		"infinite angle": func(p *handcrankfan.Params) { p.CrankAngle = math.Inf(1) },
		"huge angle":     func(p *handcrankfan.Params) { p.CrankAngle = math.MaxFloat64 },
		"flat pitch":     func(p *handcrankfan.Params) { p.BladePitch = 0.1 },
		"steep pitch":    func(p *handcrankfan.Params) { p.BladePitch = 0.8 },
		"two blades":     func(p *handcrankfan.Params) { p.Blades = 2 },
		"five blades":    func(p *handcrankfan.Params) { p.Blades = 5 },
	} {
		p := handcrankfan.DefaultParams()
		edit(&p)
		require.Error(t, p.Validate(), name)
		doc := decad.New()
		_, err := handcrankfan.Build(t.Context(), doc, p)
		require.Error(t, err, name)
		require.Empty(t, doc.Bodies(), "%s leaves the document unchanged", name)
	}
}

func TestMotion(t *testing.T) {
	p := handcrankfan.DefaultParams()
	s, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	require.InDelta(t, -2*p.CrankAngle, s.FanAngle, 1e-12, "the fan turns twice per crank turn, the other way round")
	require.InDelta(t, 170, s.Apex.Z, 0)
	// 5π/4 points the arm down and toward the operator (-Y).
	require.InDelta(t, -math.Sqrt2/2, s.ArmDirection.Y, 1e-12)
	require.InDelta(t, -math.Sqrt2/2, s.ArmDirection.Z, 1e-12)
	s, err = handcrankfan.Motion(handcrankfan.Params{CrankAngle: 0, BladePitch: 0.5, Blades: 3})
	require.NoError(t, err)
	require.InDelta(t, 1, s.ArmDirection.Z, 1e-12, "angle 0 points the arm up")
}
