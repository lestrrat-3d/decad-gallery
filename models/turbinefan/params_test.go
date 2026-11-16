package turbinefan_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
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
		require.Equal(t, handcrankfan.DefaultBladePitch, q.BladePitch)
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
		"bore radius":   {small.BoreRadius, 2.55, 1e-12},
		"bore flat":     {small.BoreFlat, 2.05, 1e-12},
		"bore loop":     {small.BoreLoop, 15.8, 0.05},
		"large radius":  {large.BoreRadius, 4.05, 1e-12},
		"large flat":    {large.BoreFlat, 3.30, 1e-12},
		"large loop":    {large.BoreLoop, 25.1, 0.05},
		"root centre x": {small.RootCenter.X, 10, 1e-4}, "root centre y": {small.RootCenter.Y, -11.1803, 1e-4},
		"root disc x": {small.RootDisc.X, 9.3333, 1e-4}, "root disc y": {small.RootDisc.Y, -10.4350, 1e-4},
		"root side": {small.RootSide, -11.1803, 1e-4}, "clamp y": {small.ClampY, -14.7803, 1e-4},
		"side end": {small.SideEnd, -18.3803, 1e-4}, "tail corner x": {small.TailCorner.X, 4.5, 1e-4},
		"tail end y": {small.TailEnd.Y, -1.4688, 1e-4}, "reach": {small.Reach, 23.4688, 1e-4},
		"large reach":  {large.Reach, 23.4688, 1e-4},
		"clamp cavity": {small.ClampSite.Cavity, 16, 1e-12}, "clamp length": {small.ClampSite.Length, 16, 0},
		"blade 2 x": {small.BladeSites[1].Head.X, -8.2272, 1e-4}, "blade 2 y": {small.BladeSites[1].Head.Y, -4.75, 1e-4},
		"blade 3 x": {small.BladeSites[2].Head.X, 8.2272, 1e-4}, "blade 1 y": {small.BladeSites[0].Head.Y, 9.5, 1e-12},
		"blade head z": {small.BladeSites[0].Head.Z, 20, 0},
	} {
		require.InDelta(t, want[1], want[0], want[2], name)
	}
	require.Equal(t, small.ClampSite.Tip(), small.ClampSite.CavityFar())
	require.Equal(t, m2fastener.HeadPocketDepth+16, 2*turbinefan.TabHalfWidth)
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
		{"pitch-low", func(p *turbinefan.Params) { p.BladePitch = 9 * math.Pi / 180 }, "BladePitch must lie in [10°, 40°] radians"},
		{"pitch-high", func(p *turbinefan.Params) { p.BladePitch = 41 * math.Pi / 180 }, "BladePitch must lie in [10°, 40°] radians"},
		{"bad-hand", func(p *turbinefan.Params) { p.Hand = 2 }, "Hand must be RightHand or LeftHand"},
		{"invalid-frame", func(p *turbinefan.Params) { p.Placement = &r3.Frame{} }, "Placement must be a valid frame"},
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
