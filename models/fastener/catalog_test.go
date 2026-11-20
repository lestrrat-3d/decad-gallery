package fastener_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/stretchr/testify/require"
)

func mustLookup(t *testing.T, name string) fastener.Preset {
	t.Helper()
	p, ok := fastener.Lookup(name)
	require.True(t, ok, name)
	return p
}

func TestCatalogNames(t *testing.T) {
	require.Equal(t, []string{"M2", "M5"}, fastener.Names())
	names := fastener.Names()
	names[0] = "changed"
	require.Equal(t, []string{"M2", "M5"}, fastener.Names())
}

func TestCatalogLookup(t *testing.T) {
	for _, name := range []string{"M2", "M5"} {
		p, ok := fastener.Lookup(name)
		require.True(t, ok, name)
		require.Equal(t, name, p.Name)
	}
	for _, name := range []string{"M3", "m2", ""} {
		p, ok := fastener.Lookup(name)
		require.False(t, ok, name)
		require.Equal(t, fastener.Preset{}, p, name)
	}

	p := mustLookup(t, "M2")
	p.Bolt.StandardLengths[0] = 999
	require.Equal(t, 6.0, mustLookup(t, "M2").Bolt.StandardLengths[0])
}

func TestCatalogValues(t *testing.T) {
	require.Equal(t, fastener.Preset{
		Name: "M2",
		Bolt: fastener.BoltSize{
			Head:            fastener.HeadSocketCap,
			ShankDiameter:   2.0,
			HeadHeight:      2.0,
			TipChamfer:      0.25,
			StandardLengths: []float64{6, 10, 16, 18, 20, 25},
			SocketCap: fastener.SocketCapHead{
				HeadDiameter:         3.8,
				SocketAcrossFlats:    1.5,
				SocketDepth:          1.0,
				HeadChamferWidth:     0.2,
				HeadChamferAngle:     30.0,
				UnderheadRadius:      0.1,
				DrillPointAngle:      118.0,
				DrillPointBaseRadius: 0.7,
				PilotDepth:           0.1,
			},
		},
		Nut: fastener.NutSize{BoreDiameter: 2.0, AcrossFlats: 4.0, Thickness: 1.6, BearingDiameter: 3.48},
	}, mustLookup(t, "M2"))
	require.Equal(t, fastener.Preset{
		Name: "M5",
		Bolt: fastener.BoltSize{
			Head:            fastener.HeadHex,
			ShankDiameter:   5.0,
			HeadHeight:      3.5,
			TipChamfer:      0.5,
			StandardLengths: []float64{16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80},
			Hex:             fastener.HexHead{AcrossFlats: 8.0, BearingDiameter: 6.88},
		},
		Nut: fastener.NutSize{BoreDiameter: 5.0, AcrossFlats: 8.0, Thickness: 4.7, BearingDiameter: 6.9},
	}, mustLookup(t, "M5"))
}

func TestCatalogPresetsValid(t *testing.T) {
	for _, name := range fastener.Names() {
		require.NoError(t, mustLookup(t, name).Validate(), name)
	}
}

func TestAcrossCorners(t *testing.T) {
	require.InDelta(t, 4.0/math.Cos(math.Pi/6), fastener.NutSize{AcrossFlats: 4}.AcrossCorners(), 1e-12)
	require.InDelta(t, 8.0/math.Cos(math.Pi/6), fastener.HexHead{AcrossFlats: 8}.AcrossCorners(), 1e-12)
}

func TestPresetValidateRejects(t *testing.T) {
	cases := []struct {
		name   string
		base   string
		mutate func(p *fastener.Preset)
	}{
		{"empty name", "M2", func(p *fastener.Preset) { p.Name = "" }},
		{"zero shank", "M2", func(p *fastener.Preset) { p.Bolt.ShankDiameter = 0 }},
		{"NaN head height", "M2", func(p *fastener.Preset) { p.Bolt.HeadHeight = math.NaN() }},
		{"infinite tip chamfer", "M2", func(p *fastener.Preset) { p.Bolt.TipChamfer = math.Inf(1) }},
		{"tip chamfer too large", "M2", func(p *fastener.Preset) { p.Bolt.TipChamfer = 1 }},
		{"no lengths", "M2", func(p *fastener.Preset) { p.Bolt.StandardLengths = nil }},
		{"unsorted lengths", "M2", func(p *fastener.Preset) { p.Bolt.StandardLengths = []float64{10, 6} }},
		{"duplicate lengths", "M2", func(p *fastener.Preset) { p.Bolt.StandardLengths = []float64{6, 6} }},
		{"length below tip chamfer", "M2", func(p *fastener.Preset) { p.Bolt.StandardLengths = []float64{0.2, 6} }},
		{"infinite length", "M2", func(p *fastener.Preset) { p.Bolt.StandardLengths = []float64{6, math.Inf(1)} }},
		{"unknown head style", "M2", func(p *fastener.Preset) { p.Bolt.Head = 0 }},
		{"hex data on socket cap", "M2", func(p *fastener.Preset) { p.Bolt.Hex.AcrossFlats = 8 }},
		{"zero head diameter", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.HeadDiameter = 0 }},
		{"head not wider than shank", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.HeadDiameter = 2.1 }},
		{"chamfer angle 90", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.HeadChamferAngle = 90 }},
		{"chamfer taller than head", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.HeadChamferWidth = 3 }},
		{"socket outside top face", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.SocketAcrossFlats = 3.5 }},
		{"pilot wider than socket", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.DrillPointBaseRadius = 0.8 }},
		{"drill point angle 180", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.DrillPointAngle = 180 }},
		{"apex below underside", "M2", func(p *fastener.Preset) { p.Bolt.SocketCap.SocketDepth = 1.5 }},
		{"socket data on hex head", "M5", func(p *fastener.Preset) { p.Bolt.SocketCap.HeadDiameter = 3.8 }},
		{"zero hex flats", "M5", func(p *fastener.Preset) { p.Bolt.Hex.AcrossFlats = 0 }},
		{"bearing below shank", "M5", func(p *fastener.Preset) { p.Bolt.Hex.BearingDiameter = 4.9 }},
		{"bearing above flats", "M5", func(p *fastener.Preset) { p.Bolt.Hex.BearingDiameter = 8.1 }},
		{"head not above overlap", "M5", func(p *fastener.Preset) { p.Bolt.HeadHeight = 1 }},
		{"nut thickness zero", "M5", func(p *fastener.Preset) { p.Nut.Thickness = 0 }},
		{"nut bearing above flats", "M5", func(p *fastener.Preset) { p.Nut.BearingDiameter = 8.5 }},
		{"nut bore differs from shank", "M5", func(p *fastener.Preset) { p.Nut.BoreDiameter = 4.5 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := mustLookup(t, tc.base)
			require.NoError(t, p.Validate())
			tc.mutate(&p)
			err := p.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), "fastener: preset")
		})
	}
}

func TestHeadStyleString(t *testing.T) {
	require.Equal(t, "socket head cap screw", fastener.HeadSocketCap.String())
	require.Equal(t, "hex head bolt", fastener.HeadHex.String())
	require.Equal(t, "HeadStyle(9)", fastener.HeadStyle(9).String())
}
