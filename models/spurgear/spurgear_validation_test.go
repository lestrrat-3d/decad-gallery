package spurgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestBuildRejectsInvalidParamsBeforeAddingBodies(t *testing.T) {
	cases := []struct {
		name   string
		change func(*spurgear.Params)
	}{
		{"zero module", func(p *spurgear.Params) { p.Module = 0 }},
		{"two teeth", func(p *spurgear.Params) { p.Teeth = 2 }},
		{"two samples", func(p *spurgear.Params) { p.Samples = 2 }},
		{"zero thickness", func(p *spurgear.Params) { p.Thickness = 0 }},
		{"zero pressure angle", func(p *spurgear.Params) { p.PressureAngle = 0 }},
		{"right angle pressure", func(p *spurgear.Params) { p.PressureAngle = math.Pi / 2 }},
		{"negative bore", func(p *spurgear.Params) { p.BoreDiameter = -1 }},
		{"root-reaching bore", func(p *spurgear.Params) { p.BoreDiameter = 14.5 }},
		{"negative chamfer", func(p *spurgear.Params) { p.CapChamfer = -0.1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := spurgear.DefaultParams()
			tc.change(&p)
			doc := decad.New()
			_, err := spurgear.Build(t.Context(), doc, p)
			require.Error(t, err)
			require.Empty(t, doc.Bodies())
		})
	}
}

func TestProfileAndFlankAPIsRejectInvalidInputs(t *testing.T) {
	p := spurgear.DefaultParams()
	world := sketch.NewWorld()
	_, err := spurgear.Build(t.Context(), nil, p)
	require.Error(t, err)
	_, _, err = spurgear.BuildProfile(t.Context(), world, nil, p)
	require.Error(t, err)
	_, _, err = spurgear.BuildToothSection(t.Context(), world, world.XY(), p, math.NaN())
	require.Error(t, err)

	p.Module, p.Teeth, p.PressureAngle = 0.5, 30, 30*math.Pi/180
	_, _, err = spurgear.BuildToothSection(t.Context(), world, world.XY(), p, 0)
	require.ErrorContains(t, err, "embedded")

	_, _, err = spurgear.InvoluteFlanks(4, 3.7, 5, math.NaN(), 5, 0)
	require.Error(t, err)
	_, _, err = spurgear.InvoluteFlanks(4, 4, 5, 8, 5, 0)
	require.Error(t, err)
	_, _, err = spurgear.InvoluteFlanks(4, 3.7, 5, 8, 2, 0)
	require.Error(t, err)
}
