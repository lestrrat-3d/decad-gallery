package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func deg(d float64) float64 { return d * math.Pi / 180 }

func TestParamsDefaultsAreTheSpecTable(t *testing.T) {
	p := screwgear.DefaultParams()
	require.Equal(t, 15.0, p.Width)
	require.Equal(t, 3.75, p.Thickness)
	require.Equal(t, 2.625, p.ToothPitch)
	require.Equal(t, 2.625, p.ToothHeight)
	require.Equal(t, 68, p.ToothCount)
	require.Equal(t, 49.5, p.TwistLead)
	require.InDelta(t, deg(25.8), p.ToothSlant, 1e-15)
	require.Equal(t, 0.048, p.ToothBow)
	require.InDelta(t, deg(80), p.CrossAngle, 1e-15)
	require.Equal(t, 1.15, p.Engagement)
	require.Equal(t, 0.0, p.MountAngleA)
	require.Equal(t, 0.0, p.MountAngleB)
	require.Equal(t, -1.31, p.AssemblyPhase)
	require.Equal(t, 15.5, p.CageRadius)
	require.Equal(t, 18.75, p.CageRise)
	require.Equal(t, 3.0, p.CollarHalf)
	require.Equal(t, 3.0, p.CollarWall)
	require.Equal(t, 0.20, p.Clearance)
	require.Equal(t, 0.60, p.RoofAllowance)
	require.Nil(t, p.Placement)
	require.NoError(t, p.Validate())
}

// TestParamsDerivedValues pins the numbers the spec quotes under "Defaults".
func TestParamsDerivedValues(t *testing.T) {
	p := screwgear.DefaultParams()
	require.InDelta(t, 43.6, p.Beta()*180/math.Pi, 0.05)                // helix angle Beta
	require.InDelta(t, 13.85, p.AxisOffset(), 1e-12)                    // A = W - Engagement
	require.InDelta(t, 178.5, p.Length(), 1e-12)                        // N * P
	require.InDelta(t, 141.5, p.Travel(), 1e-12)                        // N*P - 2*(CageRadius + CollarHalf)
	require.InDelta(t, 12.5, p.SleeveInner(), 1e-12)                    // Ri
	require.InDelta(t, 18.5, p.SleeveOuter(), 1e-12)                    // Ro
	require.InDelta(t, 7.7, p.BoreHalfWidth(), 1e-12)                   // hw
	require.InDelta(t, 2.075, p.BoreHalfThickness(), 1e-12)             // ht
	require.InDelta(t, 8.151, p.BoreCorner(), 5e-4)                     // c
	require.InDelta(t, 49.5/(2*math.Pi), p.Lambda(), 1e-12)             // screw parameter
	require.InDelta(t, 8.77, p.AxialWindow(), 5e-3)                     // the mesh stays visible check
	require.InDelta(t, 0.4834, math.Tan(p.ToothSlant), 5e-5)            // tan(Slant)
	require.InDelta(t, 0.169, p.ToothBow*3.75*3.75/4, 5e-4)             // the edge's drop at the faces
	require.InDelta(t, 19.1, p.ToothPitch/p.Lambda()*180/math.Pi, 0.05) // twist per tooth

	sIn, sOut := p.SleeveCut()
	// sIn = sqrt(Ri^2 - c^2) - 1 mm with c the roof-allowance corner radius, as the proof
	// computes it: 8.4765 mm, which the spec rounds to 8.477.
	require.InDelta(t, math.Sqrt(12.5*12.5-p.BoreCorner()*p.BoreCorner())-1, sIn, 1e-12)
	require.InDelta(t, 8.477, sIn, 1e-3)
	require.InDelta(t, 19.5, sOut, 1e-12)
}

func TestParamsValidateRangeChecks(t *testing.T) {
	// Each case changes one input of the defaults, so it passes every check before the
	// one it names, and the error has to name the field.
	leastRise := func(p screwgear.Params) float64 {
		return math.Ceil((p.AxisOffset()/2+p.BoreCorner()+p.CollarWall)*4) / 4
	}
	cases := []struct {
		name string
		edit func(*screwgear.Params)
		want string
	}{
		{"zero ribbonWidth", func(p *screwgear.Params) { p.Width = 0 }, "ribbonWidth"},
		{"negative ribbonThickness", func(p *screwgear.Params) { p.Thickness = -1 }, "ribbonThickness"},
		{"zero toothPitch", func(p *screwgear.Params) { p.ToothPitch = 0 }, "toothPitch"},
		{"zero twistLead", func(p *screwgear.Params) { p.TwistLead = 0 }, "twistLead"},
		{"zero collarHalf", func(p *screwgear.Params) { p.CollarHalf = 0 }, "collarHalf"},
		{"zero collarWall", func(p *screwgear.Params) { p.CollarWall = 0 }, "collarWall"},
		{"zero clearance", func(p *screwgear.Params) { p.Clearance = 0 }, "clearance"},
		{"negative roofAllowance", func(p *screwgear.Params) { p.RoofAllowance = -0.1 }, "roofAllowance"},
		{"3 teeth", func(p *screwgear.Params) { p.ToothCount = 3 }, "toothCount"},
		{"zero toothHeight", func(p *screwgear.Params) { p.ToothHeight = 0 }, "toothHeight"},
		{"toothHeight at half the width", func(p *screwgear.Params) { p.ToothHeight = 7.5 }, "toothHeight"},
		{"slant at 90 degrees", func(p *screwgear.Params) { p.ToothSlant = deg(90) }, "toothSlant"},
		{"slant at -90 degrees", func(p *screwgear.Params) { p.ToothSlant = deg(-90) }, "toothSlant"},
		{"negative toothBow", func(p *screwgear.Params) { p.ToothBow = -0.01 }, "toothBow"},
		{"toothBow past the half width", func(p *screwgear.Params) { p.ToothBow = 2 }, "toothBow"},
		{"zero engagement", func(p *screwgear.Params) { p.Engagement = 0 }, "engagement"},
		{"engagement past toothHeight", func(p *screwgear.Params) { p.Engagement = 2.7 }, "engagement"},
		{"crossAngle 0", func(p *screwgear.Params) { p.CrossAngle = 0 }, "crossAngle"},
		{"crossAngle 180 degrees", func(p *screwgear.Params) { p.CrossAngle = math.Pi }, "crossAngle"},
		{"assemblyPhase +pitch", func(p *screwgear.Params) { p.AssemblyPhase = p.ToothPitch }, "assemblyPhase"},
		{"assemblyPhase -pitch", func(p *screwgear.Params) { p.AssemblyPhase = -p.ToothPitch }, "assemblyPhase"},
		{"bore past the ribbon's end", func(p *screwgear.Params) { p.ToothCount = 12 }, "less than half the ribbon length"},
		{"a 4 mm clearance", func(p *screwgear.Params) { p.Clearance = 4 }, "cut starts in the hollow"},
		{"cage radius 0.02 mm past the bore's corner", func(p *screwgear.Params) {
			p.CageRadius = p.CollarHalf + p.BoreCorner() + 0.02
		}, "cut starts in the hollow"},
		{"a 1.5 mm engagement", func(p *screwgear.Params) { p.Engagement = 1.5 }, "mesh zone and its clearance"},
		{"a 16.5 mm rise", func(p *screwgear.Params) { p.CageRise = 16.5 }, "cageRise"},
		{"a 4.5 mm collarWall with a 0.55 mm clearance", func(p *screwgear.Params) {
			p.CollarWall, p.Clearance = 4.5, 0.55
			p.CageRise = leastRise(*p)
		}, "bores come"},
		{"non-finite width", func(p *screwgear.Params) { p.Width = math.NaN() }, "ribbonWidth"},
		{"infinite lead", func(p *screwgear.Params) { p.TwistLead = math.Inf(1) }, "twistLead"},
		{"an unset placement", func(p *screwgear.Params) { p.Placement = &r3.Frame{} }, "placement"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := screwgear.DefaultParams()
			c.edit(&p)
			require.ErrorContains(t, p.Validate(), c.want)
		})
	}
}

// TestParamsValidateOrder holds that an input failing two checks reports the earlier one.
func TestParamsValidateOrder(t *testing.T) {
	p := screwgear.DefaultParams()
	p.Width, p.CageRise, p.ToothSlant = 0, 1, deg(90)
	require.ErrorContains(t, p.Validate(), "ribbonWidth")

	p = screwgear.DefaultParams()
	p.CageRise, p.Engagement = 1, 1.5
	require.ErrorContains(t, p.Validate(), "mesh zone") // the mesh check comes before the rise check
}

// TestParamsValidateSleeveBounds pins the spec's quoted bound for the end wall and the
// accepted edges of the closed forms.
func TestParamsValidateSleeveBounds(t *testing.T) {
	p := screwgear.DefaultParams()
	need := p.AxisOffset()/2 + p.BoreCorner() + p.CollarWall
	require.InDelta(t, 18.08, need, 5e-3) // cageRise >= A/2 + c + collarWall

	p.CageRise = need + 1e-6
	require.NoError(t, p.Validate())
	p.CageRise = need - 1e-6
	require.ErrorContains(t, p.Validate(), "cageRise")

	// A roof allowance of zero cuts every bore to the clearance alone, and is allowed.
	q := screwgear.DefaultParams()
	q.RoofAllowance = 0
	require.NoError(t, q.Validate())
}
