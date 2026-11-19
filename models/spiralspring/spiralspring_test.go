package spiralspring_test

// CI: isolated-tests

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSpiralSpringDefault(t *testing.T) {
	m, p := requireBuilt(t, springCases[0])
	requireBounds(t, m, p, localBounds(m.Dimensions, p))
	requireBounds(t, m, p, [6]float64{-32.65, 30, -28, 30, 0, 8})
	require.Equal(t, 98, len(m.Body.Faces()))
	requireVolume(t, m, p)

	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
}

func TestSpiralSpringBranches(t *testing.T) {
	for _, c := range springCases {
		t.Run(c.name, func(t *testing.T) {
			m, p := requireBuilt(t, c)
			dm := m.Dimensions
			requireBounds(t, m, p, localBounds(dm, p))
			requireVolume(t, m, p)
			pr := holeProfile(t, m.Profile)
			chamfered := pr.Area >= edgefinish.LargeFaceArea
			rb := dm.BoreRadius
			boreRun := 2 * math.Pi * rb
			if dm.BoreFlatDistance > 0 {
				boreRun = rb*(2*math.Pi-2*math.Acos(dm.BoreFlatDistance/rb)) +
					2*math.Sqrt(rb*rb-dm.BoreFlatDistance*dm.BoreFlatDistance)
			}
			want := expectedFaces(dm, boreRun >= edgefinish.LongEdgeRun, math.Pi*dm.EyeHoleDiameter >= edgefinish.LongEdgeRun, chamfered)
			require.Equal(t, want, len(m.Body.Faces()))
			switch c.name {
			case "default":
				require.Equal(t, 98, want)
			case "round-bore":
				require.Equal(t, 97, want)
			case "one-turn":
				require.Equal(t, 20, want)
				require.False(t, chamfered)
			case "large-arbor":
				require.True(t, boreRun >= edgefinish.LongEdgeRun)
			case "thickness-floor":
				require.False(t, chamfered)
			}
			if c.noMesh {
				return
			}
			mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			if !chamfered {
				require.True(t, mesh.VolumeVerified())
			}
		})
	}
}

func TestSpiralSpringFits(t *testing.T) {
	for _, name := range []string{"default", "round-bore", "large-arbor"} {
		var c springCase
		for _, sc := range springCases {
			if sc.name == name {
				c = sc
			}
		}
		t.Run(name, func(t *testing.T) {
			m, p := requireBuilt(t, c)
			dm := m.Dimensions
			var bore, eye, flat bool
			for _, f := range m.Body.Faces() {
				switch s := f.Surface().(type) {
				case decad.Cylinder:
					if math.Abs(math.Abs(s.Axis.Z)-1) > 1e-9 {
						continue
					}
					r := s.Radius.Base()
					if math.Hypot(s.Origin.X, s.Origin.Y) < 1e-9 && r < dm.HubRadius {
						bore = true
						require.GreaterOrEqual(t, r-p.ArborDiameter/2, p.BoreClearance-1e-9)
						require.GreaterOrEqual(t, r-p.ArborDiameter/2, spiralspring.ClearanceFloor-1e-9)
						continue
					}
					if math.Hypot(s.Origin.X-dm.EyeCenterX, s.Origin.Y-dm.EyeCenterY) < 1e-9 && r < dm.EyeRadius {
						eye = true
						require.GreaterOrEqual(t, r-p.AnchorPinDiameter/2, p.AnchorClearance-1e-9)
					}
				case decad.Plane:
					n, o := s.Frame.N(), s.Frame.Origin()
					if math.Abs(math.Abs(n.Y)-1) > 1e-9 || math.Hypot(o.X, o.Y) > dm.BoreRadius || math.Abs(o.X) > dm.BoreRadius {
						continue
					}
					flat = true
					require.GreaterOrEqual(t, math.Abs(o.Y)-(p.ArborDiameter/2-p.ArborFlatDepth), p.BoreClearance-1e-9)
				}
			}
			require.True(t, bore, "bore face")
			require.True(t, eye, "eye hole face")
			require.Equal(t, p.ArborFlatDepth > 0, flat, "bore flat face")
		})
	}
}

func TestSpiralSpringPlacement(t *testing.T) {
	frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := spiralspring.DefaultParams()
	p.Placement = &frame
	doc := decad.New()
	m, err := spiralspring.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()))
	requireBounds(t, m, p, [6]float64{10, 18, -27.65, 35, -28, 30})
	requireFullyConstrained(t, m.Profile)
}

func TestSpiralSpringRejectsInvalidInputs(t *testing.T) {
	def := spiralspring.DefaultParams()
	for _, item := range []struct {
		name string
		edit func(p *spiralspring.Params)
	}{
		{"thickness", func(p *spiralspring.Params) { p.Thickness = 1.49 }},
		{"gap", func(p *spiralspring.Params) { p.MinGap = 0.99 }},
		{"height", func(p *spiralspring.Params) { p.Height = 2*p.Thickness - 0.01 }},
		{"turns-negative", func(p *spiralspring.Params) { p.Turns = -1 }},
		{"turns-too-many", func(p *spiralspring.Params) { p.Turns = 6 }},
		{"too-many-turns", func(p *spiralspring.Params) { p.OuterDiameter = 320 }},
		{"no-room", func(p *spiralspring.Params) { p.OuterDiameter = 16 }},
		{"nan", func(p *spiralspring.Params) { p.OuterDiameter = math.NaN() }},
		{"inf", func(p *spiralspring.Params) { p.Height = math.Inf(1) }},
		{"invalid-frame", func(p *spiralspring.Params) { p.Placement = &r3.Frame{} }},
		{"thin-hub-wall", func(p *spiralspring.Params) { p.ArborDiameter = 9 }},
		{"flat-too-deep", func(p *spiralspring.Params) { p.ArborFlatDepth = 2 }},
		{"bore-clearance", func(p *spiralspring.Params) { p.BoreClearance = 0.19 }},
		{"anchor-clearance", func(p *spiralspring.Params) { p.AnchorClearance = 0.19 }},
		{"arbor-zero", func(p *spiralspring.Params) { p.ArborDiameter = 0 }},
		{"outer-below-hub", func(p *spiralspring.Params) { p.OuterDiameter = 10 }},
	} {
		t.Run(item.name, func(t *testing.T) {
			p := def
			item.edit(&p)
			doc := decad.New()
			_, err := spiralspring.Build(t.Context(), doc, p)
			require.Error(t, err)
			require.ErrorContains(t, err, "spiralspring: ")
			require.Zero(t, len(doc.Bodies()))
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err := spiralspring.Build(ctx, doc, def)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, len(doc.Bodies()))
	_, err = spiralspring.Build(t.Context(), nil, def)
	require.ErrorContains(t, err, "document")
	_, err = spiralspring.Build(nil, doc, def) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
