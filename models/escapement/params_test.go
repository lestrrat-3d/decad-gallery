package escapement_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	const deg = math.Pi / 180
	require.NoError(t, escapement.DefaultParams().Validate())
	require.Equal(t, 0.25, escapement.DefaultParams().Phase)

	cases := []struct {
		field string
		edit  func(*escapement.Params)
	}{
		{"PendulumLength", func(p *escapement.Params) { p.PendulumLength = 119 }},
		{"PendulumLength", func(p *escapement.Params) { p.PendulumLength = 251 }},
		{"PendulumLength", func(p *escapement.Params) { p.PendulumLength = math.NaN() }},
		{"Amplitude", func(p *escapement.Params) { p.Amplitude = 2.5 * deg }},
		{"Amplitude", func(p *escapement.Params) { p.Amplitude = 6.5 * deg }},
		{"Amplitude", func(p *escapement.Params) { p.Amplitude = math.Inf(1) }},
		{"LockAngle", func(p *escapement.Params) { p.LockAngle = 1 * deg }},
		{"LockAngle", func(p *escapement.Params) { p.LockAngle = 1.6 * deg }},
		{"LiftAngle", func(p *escapement.Params) { p.LiftAngle = 1.9 * deg }},
		{"LiftAngle", func(p *escapement.Params) { p.LiftAngle = 3.1 * deg }},
		{"Phase", func(p *escapement.Params) { p.Phase = -0.1 }},
		{"Phase", func(p *escapement.Params) { p.Phase = 1 }},
		{"Phase", func(p *escapement.Params) { p.Phase = math.NaN() }},
		{"Backlash", func(p *escapement.Params) { p.Backlash = -0.01 }},
		{"Backlash", func(p *escapement.Params) { p.Backlash = 0.31 }},
		{"Placement", func(p *escapement.Params) { p.Placement = &r3.Frame{} }},
	}
	for _, c := range cases {
		p := escapement.DefaultParams()
		c.edit(&p)
		err := p.Validate()
		require.Error(t, err, c.field)
		require.Contains(t, err.Error(), c.field)
	}

	edges := []func(*escapement.Params){
		func(p *escapement.Params) { p.PendulumLength = 120 },
		func(p *escapement.Params) { p.PendulumLength = 250 },
		func(p *escapement.Params) { p.Amplitude = 6 * deg },
		func(p *escapement.Params) { p.Amplitude = (p.LiftAngle+p.LockAngle)/2 + deg },
		func(p *escapement.Params) { p.LockAngle = 1.25 * deg; p.LiftAngle = 3 * deg },
		func(p *escapement.Params) { p.Phase = 0.999 },
		func(p *escapement.Params) { p.Backlash = 0 },
		func(p *escapement.Params) { p.Backlash = 0.3 },
	}
	for i, edit := range edges {
		p := escapement.DefaultParams()
		edit(&p)
		require.NoError(t, p.Validate(), "edge %d", i)
	}
}
