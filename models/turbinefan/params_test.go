package turbinefan_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestTurbineFanParams(t *testing.T) {
	p := turbinefan.DefaultParams()
	require.Equal(t, turbinefan.Params605ZZ(), p)
	require.Equal(t, ballbearing.Params605ZZ().BoreDiameter, p.ShaftDiameter)
	require.Equal(t, ballbearing.Params608ZZ().BoreDiameter, turbinefan.Params608ZZ().ShaftDiameter)
	for _, q := range []turbinefan.Params{turbinefan.Params605ZZ(), turbinefan.Params608ZZ()} {
		require.Equal(t, ballbearing.PressFitClearance, q.BoreClearance)
		require.Equal(t, turbinefan.DefaultBladePitch, q.BladePitch)
		require.Nil(t, q.Placement)
		require.Equal(t, turbinefan.RightHand, q.Hand)
	}
	require.InDelta(t, 0.5, p.FlatDepth, 0)
	require.InDelta(t, 0.75, turbinefan.Params608ZZ().FlatDepth, 0)

	small, err := p.Derive()
	require.NoError(t, err)
	large, err := turbinefan.Params608ZZ().Derive()
	require.NoError(t, err)
	for name, want := range map[string][3]float64{
		"bore radius": {small.BoreRadius, 2.55, 1e-12}, "bore flat": {small.BoreFlat, 2.05, 1e-12},
		"bore flat width": {small.BoreFlatWidth, 3.0332, 1e-4}, "bore loop": {small.BoreLoop, 15.81, 0.05},
		"shaft flat width": {small.ShaftFlatWidth, 3.0, 1e-4}, "tip": {small.SetScrewTip, 1.95, 1e-12},
		"head": {small.SetScrewHead, 9.95, 1e-12}, "floor": {small.CounterboreFloor, 7.55, 1e-12},
		"large radius": {large.BoreRadius, 4.05, 1e-12}, "large flat": {large.BoreFlat, 3.30, 1e-12},
		"large flat width": {large.BoreFlatWidth, 4.6957, 1e-4}, "large loop": {large.BoreLoop, 25.12, 0.05},
		"large shaft flat": {large.ShaftFlatWidth, 4.6637, 1e-4}, "large tip": {large.SetScrewTip, 3.20, 1e-12},
		"large head": {large.SetScrewHead, 11.2, 1e-12}, "large floor": {large.CounterboreFloor, 8.8, 1e-12},
		"screw head y": {small.SetScrewSite.Head.Y, -9.95, 1e-12}, "screw head z": {small.SetScrewSite.Head.Z, 4, 0},
		"large screw head y": {large.SetScrewSite.Head.Y, -11.2, 1e-12}, "screw dir": {small.SetScrewSite.Dir.Y, 1, 0},
		"screw corner": {small.SetScrewSite.Corner.Z, 1, 0}, "screw length": {small.SetScrewSite.Length, 6, 0},
		"screw cavity": {small.SetScrewSite.Cavity, 4.3, 1e-12}, "large screw cavity": {large.SetScrewSite.Cavity, 4.3, 1e-12},
		"blade 2 x": {small.BladeSites[1].Head.X, -8.2272, 1e-4}, "blade 2 y": {small.BladeSites[1].Head.Y, -4.75, 1e-4},
		"blade 3 x": {small.BladeSites[2].Head.X, 8.2272, 1e-4}, "blade 1 y": {small.BladeSites[0].Head.Y, 9.5, 1e-12},
		"blade head z": {small.BladeSites[0].Head.Z, 20, 0},
	} {
		require.InDelta(t, want[1], want[0], want[2], name)
	}
	require.Equal(t, small.SetScrewSite.Tip(), small.SetScrewSite.Length+m2fastener.HeadPocketDepth)
}

// TestTurbineFanValidate rejects each bad field in the order Derive checks.
func TestTurbineFanValidate(t *testing.T) {
	def := turbinefan.DefaultParams()
	for _, item := range []struct {
		name string
		edit func(p *turbinefan.Params)
		want string
	}{
		{"nan-shaft", func(p *turbinefan.Params) { p.ShaftDiameter = math.NaN() }, "ShaftDiameter must be finite and positive"},
		{"inf-shaft", func(p *turbinefan.Params) { p.ShaftDiameter = math.Inf(1) }, "ShaftDiameter must be finite and positive"},
		{"zero-shaft", func(p *turbinefan.Params) { p.ShaftDiameter = 0 }, "ShaftDiameter must be finite and positive"},
		{"zero-flat", func(p *turbinefan.Params) { p.FlatDepth = 0 }, "FlatDepth must be finite and positive"},
		{"zero-clearance", func(p *turbinefan.Params) { p.BoreClearance = 0 }, "BoreClearance must be finite and positive"},
		{"zero-pitch", func(p *turbinefan.Params) { p.BladePitch = 0 }, "BladePitch must be finite and positive"},
		{"flat-too-deep", func(p *turbinefan.Params) { p.FlatDepth = 1.3 }, "FlatDepth must be at most a quarter"},
		{"press-clearance", func(p *turbinefan.Params) { p.BoreClearance = 0.04 }, "BoreClearance must be from 0.05 to 0.5 mm"},
		{"loose-clearance", func(p *turbinefan.Params) { p.BoreClearance = 0.51 }, "BoreClearance must be from 0.05 to 0.5 mm"},
		{"shaft-too-wide", func(p *turbinefan.Params) { p.ShaftDiameter = 8.2; p.FlatDepth = 0.5 }, "less than a 1.6 mm wall"},
		{"608-loose", func(p *turbinefan.Params) { *p = turbinefan.Params608ZZ(); p.BoreClearance = 0.11 }, "less than a 1.6 mm wall"},
		{"shaft-flat-narrow", func(p *turbinefan.Params) { p.FlatDepth = 0.2 }, "shaft's flat at least 2 mm wide"},
		{"bore-flat-narrow", func(p *turbinefan.Params) { p.FlatDepth = 0.25 }, "bore's flat at least 2.8 mm wide"},
		{"pitch-low", func(p *turbinefan.Params) { p.BladePitch = 9 * math.Pi / 180 }, "BladePitch must lie in [10°, 40°] radians"},
		{"pitch-high", func(p *turbinefan.Params) { p.BladePitch = 41 * math.Pi / 180 }, "BladePitch must lie in [10°, 40°] radians"},
		{"bad-hand", func(p *turbinefan.Params) { p.Hand = 2 }, "Hand must be RightHand or LeftHand"},
		{"invalid-frame", func(p *turbinefan.Params) { p.Placement = &r3.Frame{} }, "Placement must be a valid frame"},
		{"bad-mount", func(p *turbinefan.Params) { p.Mount = turbinefan.Mount(2) }, "Mount must be ShaftMount or ClutchMount"},
	} {
		t.Run(item.name, func(t *testing.T) {
			p := def
			item.edit(&p)
			require.ErrorContains(t, p.Validate(), "turbinefan: ")
			require.ErrorContains(t, p.Validate(), item.want)
			doc := turbinefan.NewDocument()
			_, err := turbinefan.Build(t.Context(), doc, p)
			require.ErrorContains(t, err, item.want)
			require.Empty(t, doc.Bodies())
			_, err = turbinefan.BuildParts(t.Context(), doc, p)
			require.ErrorContains(t, err, item.want)
			require.Empty(t, doc.Bodies())
		})
	}
	// The 608ZZ's widest clearance that still leaves the 1.6 mm wall to the sockets.
	edge := turbinefan.Params608ZZ()
	edge.BoreClearance = 0.1
	require.NoError(t, edge.Validate())
	// Each field is checked before the next: a bad first field wins.
	both := def
	both.ShaftDiameter, both.BladePitch = 0, 0
	require.ErrorContains(t, both.Validate(), "ShaftDiameter")

	// One hub serves both mounts, so a shallow flat fails in ClutchMount too.
	shallow := turbinefan.DefaultParams()
	shallow.FlatDepth = 0.25
	require.Error(t, shallow.Validate())
	shallow.Mount = turbinefan.ClutchMount
	require.ErrorContains(t, shallow.Validate(), "bore's flat at least 2.8 mm wide")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err := turbinefan.Build(ctx, doc, def)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, doc.Bodies())
	_, err = turbinefan.Build(t.Context(), nil, def)
	require.ErrorContains(t, err, "document")
	_, err = turbinefan.Build(nil, doc, def) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}

func TestTurbineFanClutchParams(t *testing.T) {
	def, err := turbinefan.ParamsForClutch(onewayclutch.DefaultParams())
	require.NoError(t, err)
	require.InDelta(t, 5, def.ShaftDiameter, 0)
	require.InDelta(t, 0.5, def.FlatDepth, 0)
	require.Equal(t, turbinefan.RightHand, def.Hand)
	require.Equal(t, turbinefan.ClutchMount, def.Mount)
	require.Equal(t, ballbearing.PressFitClearance, def.BoreClearance)
	require.Equal(t, turbinefan.DefaultBladePitch, def.BladePitch)
	mf, err := turbinefan.MountFrame(nil)
	require.NoError(t, err)
	require.NotNil(t, def.Placement)
	require.Equal(t, mf, *def.Placement)
	require.NoError(t, def.Validate())

	left, err := turbinefan.ParamsForClutch(onewayclutch.Params{
		Shaft: "608ZZ", Freewheel: onewayclutch.CounterClockwise, BoreClearance: 0.05})
	require.NoError(t, err)
	require.InDelta(t, 8, left.ShaftDiameter, 0)
	require.InDelta(t, 0.75, left.FlatDepth, 0)
	require.Equal(t, turbinefan.LeftHand, left.Hand)
	require.NoError(t, left.Validate())
	gearbox, err := turbinefan.ParamsForClutch(onewayclutch.Params{
		Shaft: "608ZZ-gearbox", Freewheel: onewayclutch.Clockwise, BoreClearance: 0.05})
	require.NoError(t, err)
	require.InDelta(t, 0.5, gearbox.FlatDepth, 0)

	_, err = turbinefan.ParamsForClutch(onewayclutch.Params{Shaft: "605ZZ", BoreClearance: 0.05})
	require.ErrorContains(t, err, "turbinefan: clutch: onewayclutch: Freewheel")

	require.Equal(t, onewayclutch.Clockwise, turbinefan.RightHand.Freewheel())
	require.Equal(t, onewayclutch.CounterClockwise, turbinefan.LeftHand.Freewheel())
	require.Equal(t, onewayclutch.Direction(0), turbinefan.Hand(5).Freewheel())
	for k, want := range []float64{270, 30, 150} {
		require.InDelta(t, want*math.Pi/180, turbinefan.EarBoltAngle(k+1), 1e-12)
	}

	// The hub has the same cuts in both modes; the adapter sites are the adapter's first two face sites, in the clutch's
	// frame carried through MountFrame.
	cdm, err := onewayclutch.DefaultParams().Derive()
	require.NoError(t, err)
	dm, err := def.Derive()
	require.NoError(t, err)
	shaft, err := turbinefan.DefaultParams().Derive()
	require.NoError(t, err)
	require.Equal(t, shaft.AdapterSites, dm.AdapterSites)
	require.Equal(t, shaft.SetScrewSite, dm.SetScrewSite)
	// The fan's placement point is the clutch cover's outer face, so the fan frame sits ClutchAnchorZ behind it.
	xf := anchoredTransform(t, mf, turbinefan.ClutchAnchorZ)
	for k, site := range dm.AdapterSites {
		face := cdm.FaceSites[k+1] // clutch angles 210 and 330; face hole 1 (90) stays empty
		a := turbinefan.EarBoltAngle(k + 2)
		require.InDelta(t, onewayclutch.FaceCircle*math.Cos(a), site.Head.X, 1e-12)
		require.InDelta(t, onewayclutch.FaceCircle*math.Sin(a), site.Head.Y, 1e-12)
		require.InDelta(t, -(onewayclutch.AdapterThickness + turbinefan.ClutchSeam), site.Head.Z, 1e-12)
		require.Equal(t, r3.Vec{Z: 1}, site.Dir)
		require.InDelta(t, onewayclutch.FaceLength, site.Length, 0)
		require.InDelta(t, 7.75, site.Cavity, 1e-12)
		require.InDelta(t, 0, xf.Apply(site.Head).Sub(face.Head).Len(), 1e-9, "adapter site %d head", k+1)
		require.InDelta(t, 0, xf.ApplyDir(site.Dir).Sub(face.Dir).Len(), 1e-9, "adapter site %d direction", k+1)
		require.InDelta(t, site.Length, face.Length, 0)
		require.InDelta(t, site.Cavity, face.Cavity, 1e-9)
		require.InDelta(t, 0, xf.ApplyDir(site.Corner).Sub(face.Corner).Len(), 1e-9, "adapter site %d corner", k+1)
	}
	// The bore's flat faces -Y in the fan frame and the shaft's flat faces +Y in the clutch frame: ClutchTurn lines them up.
	require.InDelta(t, 0, xf.ApplyDir(r3.Vec{Y: -1}).Sub(r3.Vec{Y: 1}).Len(), 1e-9, "bore flat meets the shaft flat")
	require.NotEqual(t, m2fastener.Site{}, dm.SetScrewSite)
	big, err := left.Derive()
	require.NoError(t, err)
	require.InDelta(t, 3.2, big.SetScrewTip, 1e-12)

	require.InDelta(t, math.Pi, turbinefan.ClutchTurn, 0)
	require.InDelta(t, onewayclutch.CoverTo, mf.Origin().Z, 0)
	require.InDelta(t, -7.75, turbinefan.ClutchAnchorZ, 1e-12)
	require.InDelta(t, math.Cos(turbinefan.ClutchTurn), mf.U().X, 1e-12)
	require.InDelta(t, math.Sin(turbinefan.ClutchTurn), mf.U().Y, 1e-12)
	f, err := r3.NewFrame(r3.Vec{X: 10, Y: -5, Z: 7}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	placed, err := turbinefan.MountFrame(&f)
	require.NoError(t, err)
	outer, err := r3.FromFrame(f)
	require.NoError(t, err)
	inner, err := r3.FromFrame(mf)
	require.NoError(t, err)
	both, err := r3.FromFrame(placed)
	require.NoError(t, err)
	for _, v := range []r3.Vec{{}, {X: 1, Y: 2, Z: 3}, {X: -4, Z: 5}} {
		require.InDelta(t, 0, both.Apply(v).Sub(outer.Apply(inner.Apply(v))).Len(), 1e-9)
	}
	_, err = turbinefan.MountFrame(&r3.Frame{})
	require.ErrorContains(t, err, "turbinefan: clutch Placement must be a valid frame")
}

func TestTurbineFanClutchBladePitch(t *testing.T) {
	steep, err := turbinefan.ParamsForClutch(onewayclutch.DefaultParams(), turbinefan.WithBladePitch(turbinefan.MaxBladePitch))
	require.NoError(t, err)
	require.Equal(t, turbinefan.MaxBladePitch, steep.BladePitch)
	require.Equal(t, turbinefan.ClutchMount, steep.Mount)
	def, err := turbinefan.ParamsForClutch(onewayclutch.DefaultParams())
	require.NoError(t, err)
	require.Equal(t, turbinefan.DefaultBladePitch, def.BladePitch)
	_, err = turbinefan.ParamsForClutch(onewayclutch.DefaultParams(), turbinefan.WithBladePitch(turbinefan.MaxBladePitch+0.1))
	require.ErrorContains(t, err, "BladePitch")
}
