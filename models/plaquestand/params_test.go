package plaquestand_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/stretchr/testify/require"
)

// expected holds one case's offline values from the design (docs: instructions.md "Representative cases").
type expected struct {
	name                       string
	params                     plaquestand.Params
	standWidth, backHeight     float64
	seatY, depth, height       float64
	tipFront, tipBack, tipSide float64
	area, length               float64
}

func cases() []expected {
	p := func(w, h, t, theta, lip, c float64) plaquestand.Params {
		return plaquestand.Params{
			PlaqueWidth: w, PlaqueHeight: h, PlaqueThickness: t, Recline: theta, LipHeight: lip, SlotClearance: c,
		}
	}
	return []expected{
		{"default", p(300, 250, 6, 15, 10, 0.55), 180, 50, 19.872139, 93, 58.253249,
			19.610016, 15.293816, 33.934962, 1136.317882, 320.343749},
		{"min", p(150, 120, 2, 10, 6, 0.55), 120, 30, 17.280750, 51, 38.349710,
			18.644768, 15.499808, 40.766229, 632.557129, 182.688124},
		{"max", p(400, 350, 15, 25, 20, 1.5), 200, 60, 29.192341, 160, 66.588125,
			27.346216, 15.206476, 29.031289, 1894.182043, 509.428323},
		{"forward", p(300, 350, 2, 10, 10, 0.55), 180, 60, 23.828646, 109, 67.893943,
			15.000000, 15.262544, 26.130591, 1296.557130, 366.688124},
		{"narrow-tall", p(150, 350, 15, 10, 6, 1.5), 120, 60, 31.809631, 124, 74.050264,
			15.000000, 15.154925, 17.328331, 1514.617026, 412.092989},
		{"short-steep", p(150, 120, 15, 25, 20, 1.5), 120, 30, 29.192341, 81, 40.220496,
			32.136555, 15.245825, 38.859428, 1082.182043, 291.428323},
	}
}

func TestPlaqueStandParamsDefaults(t *testing.T) {
	p := plaquestand.DefaultParams()
	require.Equal(t, plaquestand.DefaultPlaqueWidth, p.PlaqueWidth)
	require.Equal(t, plaquestand.DefaultPlaqueHeight, p.PlaqueHeight)
	require.Equal(t, plaquestand.DefaultPlaqueThickness, p.PlaqueThickness)
	require.Equal(t, plaquestand.DefaultRecline, p.Recline)
	require.Equal(t, plaquestand.DefaultLipHeight, p.LipHeight)
	require.Equal(t, plaquestand.DefaultSlotClearance, p.SlotClearance)
	require.Equal(t, 300.0, p.PlaqueWidth)
	require.Equal(t, 250.0, p.PlaqueHeight)
	require.Equal(t, 6.0, p.PlaqueThickness)
	require.Equal(t, 15.0, p.Recline)
	require.Equal(t, 10.0, p.LipHeight)
	require.Equal(t, 0.55, p.SlotClearance)
	require.Equal(t, plaquestand.LooseFit+plaquestand.FitMargin, plaquestand.MinSlotClearance)
	require.NoError(t, p.Validate())
}

func TestPlaqueStandDeriveCases(t *testing.T) {
	const tol = 1e-6
	limit := plaquestand.BedSize - 2*plaquestand.BedMargin
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.params.Derive()
			require.NoError(t, err)
			require.InDelta(t, tc.standWidth, dm.StandWidth, tol)
			require.InDelta(t, tc.backHeight, dm.BackHeight, tol)
			require.InDelta(t, tc.seatY, dm.Seat.Y, tol)
			require.InDelta(t, tc.depth, dm.Depth, tol)
			require.InDelta(t, tc.height, dm.Height, tol)
			require.InDelta(t, tc.tipFront, dm.TipFront, tol)
			require.InDelta(t, tc.tipBack, dm.TipBack, tol)
			require.InDelta(t, tc.tipSide, dm.TipSide, tol)
			require.InDelta(t, tc.area, dm.Outline.Area(), tol)
			require.InDelta(t, tc.length, dm.Outline.Length(), tol)

			require.GreaterOrEqual(t, dm.TipFront, plaquestand.MinTipAngle-1e-9)
			require.GreaterOrEqual(t, dm.TipBack, plaquestand.MinTipAngle-1e-9)
			require.GreaterOrEqual(t, dm.TipSide, plaquestand.MinTipAngle-1e-9)
			require.LessOrEqual(t, dm.StandWidth, limit)
			require.LessOrEqual(t, dm.Depth, limit)
			require.LessOrEqual(t, dm.Height, limit)
		})
	}

	t.Run("default extras", func(t *testing.T) {
		dm, err := plaquestand.DefaultParams().Derive()
		require.NoError(t, err)
		require.InDelta(t, 7.1, dm.SlotWidth, tol)
		require.InDelta(t, 3.55, dm.SlotRadius, tol)
		require.InDelta(t, 47, dm.BackWall, tol)
		require.InDelta(t, 7.5, dm.LipWall, tol)
		require.InDelta(t, 11.55, dm.Seat.Z, tol)
		require.InDelta(t, 1.340720, dm.PlayAngle, tol)
		require.InDelta(t, math.Sin(15*math.Pi/180), dm.Axis.Y, tol)
		require.InDelta(t, math.Cos(15*math.Pi/180), dm.Axis.Z, tol)
		require.InDelta(t, math.Cos(15*math.Pi/180), dm.Normal.Y, tol)
		require.InDelta(t, -math.Sin(15*math.Pi/180), dm.Normal.Z, tol)
	})
}

func TestPlaqueStandDeriveRejects(t *testing.T) {
	type field struct {
		name   string
		lo, hi float64
		set    func(*plaquestand.Params, float64)
	}
	fields := []field{
		{"PlaqueWidth", plaquestand.MinPlaqueWidth, plaquestand.MaxPlaqueWidth,
			func(p *plaquestand.Params, v float64) { p.PlaqueWidth = v }},
		{"PlaqueHeight", plaquestand.MinPlaqueHeight, plaquestand.MaxPlaqueHeight,
			func(p *plaquestand.Params, v float64) { p.PlaqueHeight = v }},
		{"PlaqueThickness", plaquestand.MinPlaqueThickness, plaquestand.MaxPlaqueThickness,
			func(p *plaquestand.Params, v float64) { p.PlaqueThickness = v }},
		{"Recline", plaquestand.MinRecline, plaquestand.MaxRecline,
			func(p *plaquestand.Params, v float64) { p.Recline = v }},
		{"LipHeight", plaquestand.MinLipHeight, plaquestand.MaxLipHeight,
			func(p *plaquestand.Params, v float64) { p.LipHeight = v }},
		{"SlotClearance", plaquestand.MinSlotClearance, plaquestand.MaxSlotClearance,
			func(p *plaquestand.Params, v float64) { p.SlotClearance = v }},
	}
	for _, f := range fields {
		for _, v := range []float64{f.lo - 0.01, f.hi + 0.01} {
			p := plaquestand.DefaultParams()
			f.set(&p, v)
			_, err := p.Derive()
			require.Error(t, err, f.name)
			require.True(t, strings.HasPrefix(err.Error(), "plaquestand: "), err.Error())
			require.Contains(t, err.Error(), f.name)
			require.EqualError(t, p.Validate(), err.Error())
		}
	}
	p := plaquestand.DefaultParams()
	p.PlaqueWidth = math.NaN()
	_, err := p.Derive()
	require.ErrorContains(t, err, "plaquestand: PlaqueWidth")
	p = plaquestand.DefaultParams()
	p.PlaqueHeight = math.Inf(1)
	_, err = p.Derive()
	require.ErrorContains(t, err, "plaquestand: PlaqueHeight")

	p = plaquestand.DefaultParams()
	p.PlaqueWidth = 149.99
	_, err = p.Derive()
	require.EqualError(t, err, "plaquestand: PlaqueWidth must be from 150 to 400 mm, got 149.99")
}
