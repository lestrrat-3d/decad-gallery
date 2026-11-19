package plaquestand_test

// CI: isolated-tests

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/stretchr/testify/require"
)

func TestPlaqueStandParamsDefaults(t *testing.T) {
	p := plaquestand.DefaultParams()
	require.Equal(t, plaquestand.DefaultPlaqueWidth, p.PlaqueWidth)
	require.Equal(t, plaquestand.DefaultPlaqueHeight, p.PlaqueHeight)
	require.Equal(t, plaquestand.DefaultPlaqueThickness, p.PlaqueThickness)
	require.Equal(t, plaquestand.DefaultRecline, p.Recline)
	require.Equal(t, plaquestand.DefaultLipHeight, p.LipHeight)
	require.Equal(t, plaquestand.DefaultSlotClearance, p.SlotClearance)
	require.Equal(t, 250.0, p.PlaqueWidth)
	require.Equal(t, 305.0, p.PlaqueHeight)
	require.Equal(t, 12.8, p.PlaqueThickness)
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
			require.InDelta(t, tc.span, dm.Span, tol)
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
			require.LessOrEqual(t, dm.Span, limit)
			require.LessOrEqual(t, dm.Depth, limit)
			require.LessOrEqual(t, dm.Height, limit)
		})
	}

	t.Run("default extras", func(t *testing.T) {
		dm, err := plaquestand.DefaultParams().Derive()
		require.NoError(t, err)
		require.InDelta(t, 13.9, dm.SlotWidth, tol)
		require.InDelta(t, 6.95, dm.SlotRadius, tol)
		require.InDelta(t, 57, dm.BackWall, tol)
		require.InDelta(t, 7.5, dm.LipWall, tol)
		require.InDelta(t, 14.95, dm.Seat.Z, tol)
		require.InDelta(t, 1.105571, dm.PlayAngle, tol)
		require.InDelta(t, 10.8, dm.ChannelWidth, tol)
		require.InDelta(t, 4, dm.ChannelHeight, tol)
		require.InDelta(t, 101, dm.ChannelBack, tol)
		require.InDelta(t, 90.2, dm.ChannelFront, tol)
		require.InDelta(t, 4, dm.ChannelFloor, tol)
		require.InDelta(t, 95.6, dm.RailCenter, tol)
		for k, x := range []float64{10, 190} {
			site := dm.Sites[k]
			require.InDelta(t, x, site.Head.X, tol)
			require.InDelta(t, 95.6, site.Head.Y, tol)
			require.InDelta(t, 8.4, site.Head.Z, tol)
			require.NoError(t, site.Validate())
			// The modeled tip stands 0.4 mm inside the pocket: clamped, the joint drops by JointFit and it ends flush.
			require.InDelta(t, 8, site.Tip(), tol)
			require.InDelta(t, 8.4, site.CavityFar(), tol)
		}
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
