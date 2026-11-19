package swisslever_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require.NoError(t, swisslever.DefaultParams().Validate())
	require.Equal(t, 0.25, swisslever.DefaultParams().Phase)
	require.InDelta(t, 180*deg, swisslever.DefaultParams().Amplitude, 1e-12)

	cases := []struct {
		field string
		edit  func(*swisslever.Params)
	}{
		{"Amplitude", func(p *swisslever.Params) { p.Amplitude = 44 * deg }},
		{"Amplitude", func(p *swisslever.Params) { p.Amplitude = 271 * deg }},
		{"Amplitude", func(p *swisslever.Params) { p.Amplitude = math.NaN() }},
		{"Amplitude", func(p *swisslever.Params) { p.Amplitude = 0 }},
		{"Phase", func(p *swisslever.Params) { p.Phase = -0.1 }},
		{"Phase", func(p *swisslever.Params) { p.Phase = 1 }},
		{"Phase", func(p *swisslever.Params) { p.Phase = math.Inf(1) }},
		{"Placement", func(p *swisslever.Params) { p.Placement = &r3.Frame{} }},
	}
	for _, c := range cases {
		p := swisslever.DefaultParams()
		c.edit(&p)
		err := p.Validate()
		require.Error(t, err, c.field)
		require.Contains(t, err.Error(), c.field)
		_, err = swisslever.Layout(p)
		require.Error(t, err, "Layout validates first")
	}

	edges := []func(*swisslever.Params){
		func(p *swisslever.Params) { p.Amplitude = 45 * deg },
		func(p *swisslever.Params) { p.Amplitude = 270 * deg },
		func(p *swisslever.Params) { p.Phase = 0 },
		func(p *swisslever.Params) { p.Phase = 0.999 },
	}
	for i, edit := range edges {
		p := swisslever.DefaultParams()
		edit(&p)
		require.NoError(t, p.Validate(), "edge %d", i)
	}
}
