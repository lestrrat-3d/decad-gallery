package spiralspring_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}

// springCase is one parameter branch of the model.
type springCase struct {
	name  string
	edit  func(p *spiralspring.Params)
	turns int
	// noMesh skips the mesh check, which takes minutes for 40 turns.
	noMesh bool
}

var springCases = []springCase{
	{name: "default", edit: func(*spiralspring.Params) {}, turns: 5},
	{name: "one-turn", edit: func(p *spiralspring.Params) { p.OuterDiameter = 28 }, turns: 1},
	{name: "round-bore", edit: func(p *spiralspring.Params) { p.ArborFlatDepth = 0 }, turns: 5},
	{name: "turns-override", edit: func(p *spiralspring.Params) { p.Turns = 3 }, turns: 3},
	{name: "large-arbor", edit: func(p *spiralspring.Params) { p.HubDiameter, p.ArborDiameter = 16, 8 }, turns: 5},
	{name: "thickness-floor", edit: func(p *spiralspring.Params) {
		p.OuterDiameter, p.Thickness, p.MinGap, p.Height = 30, 1.5, 1, 4
	}, turns: 2},
	{name: "gap-floor", edit: func(p *spiralspring.Params) {
		p.OuterDiameter, p.Thickness, p.MinGap, p.Height = 32, 1.5, 1, 4
	}, turns: 3},
	{name: "max-turns", edit: func(p *spiralspring.Params) { p.OuterDiameter = 310 }, turns: 40, noMesh: true},
}

func (c springCase) params() spiralspring.Params {
	p := spiralspring.DefaultParams()
	c.edit(&p)
	return p
}

// holeProfile returns the spring's profile: the one region with two holes, the bore and the eye hole.
func holeProfile(t *testing.T, s *sketch.Sketch) *sketch.Profile {
	t.Helper()
	for _, pr := range s.Profiles() {
		require.True(t, pr.Valid)
		if len(pr.Holes) == 2 {
			return pr
		}
	}
	require.Fail(t, "no profile with two holes")
	return nil
}

func entityLength(t *testing.T, e sketch.Entity) float64 {
	t.Helper()
	switch v := e.(type) {
	case *sketch.Line:
		return v.Length()
	case *sketch.Arc:
		sweep := math.Mod(v.EndAngle()-v.StartAngle(), 2*math.Pi)
		if sweep < 0 {
			sweep += 2 * math.Pi
		}
		return v.R() * sweep
	case *sketch.Circle:
		return 2 * math.Pi * v.R()
	}
	require.Failf(t, "unexpected entity", "%T", e)
	return 0
}

// expectedFaces is the face count the construction predicts.
func expectedFaces(dm spiralspring.Dimensions, boreLoopChamfered, eyeLoopChamfered, chamfered bool) int {
	holes := 2
	if dm.BoreFlatDistance > 0 {
		holes = 3
	}
	n := (4*dm.Turns + 6) + holes + 2 + 5
	if !chamfered {
		return n
	}
	n += 2 * (4*dm.Turns + 11)
	if boreLoopChamfered {
		if dm.BoreFlatDistance > 0 {
			// Per cap: the arc, the flat, and one face at each of the two arc-to-flat corners.
			n += 8
		} else {
			n += 2
		}
	}
	if eyeLoopChamfered {
		n += 2
	}
	return n
}

func requireBuilt(t *testing.T, c springCase) (*spiralspring.Model, spiralspring.Params) {
	t.Helper()
	p := c.params()
	doc := decad.New()
	m, err := spiralspring.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()))
	require.True(t, m.Body.IsSolid())
	require.Equal(t, c.turns, m.Dimensions.Turns)
	requireFullyConstrained(t, m.Profile)
	return m, p
}

func requireBounds(t *testing.T, m *spiralspring.Model, p spiralspring.Params, want [6]float64) {
	t.Helper()
	box, err := m.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, want[0], box.Min.X, tol)
	require.InDelta(t, want[1], box.Max.X, tol)
	require.InDelta(t, want[2], box.Min.Y, tol)
	require.InDelta(t, want[3], box.Max.Y, tol)
	require.InDelta(t, want[4], box.Min.Z, tol)
	require.InDelta(t, want[5], box.Max.Z, tol)
}

func localBounds(dm spiralspring.Dimensions, p spiralspring.Params) [6]float64 {
	return [6]float64{
		dm.EyeCenterX - dm.EyeRadius, dm.CoilRadius,
		math.Min(-(dm.CoilRadius - dm.CenterOffset), dm.EyeCenterY-dm.EyeRadius), dm.CoilRadius,
		0, p.Height,
	}
}

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

// requireVolume checks the body's volume against the profile area times the height, less the chamfers.
func requireVolume(t *testing.T, m *spiralspring.Model, p spiralspring.Params) {
	t.Helper()
	pr := holeProfile(t, m.Profile)
	dm := m.Dimensions
	volume, err := m.Body.Volume()
	require.NoError(t, err)
	v := volume.Value.Base()
	fillets := 5 * p.Height * dm.FilletRadius * dm.FilletRadius
	if pr.Area < edgefinish.LargeFaceArea {
		require.GreaterOrEqual(t, v, pr.Area*p.Height-1e-6)
		require.LessOrEqual(t, v, pr.Area*p.Height+fillets)
		return
	}
	perimeter := 0.0
	for _, e := range pr.Outer {
		perimeter += entityLength(t, e.Entity)
	}
	rb := dm.BoreRadius
	boreRun := 2 * math.Pi * rb
	if dm.BoreFlatDistance > 0 {
		boreRun = rb*(2*math.Pi-2*math.Acos(dm.BoreFlatDistance/rb)) + 2*math.Sqrt(rb*rb-dm.BoreFlatDistance*dm.BoreFlatDistance)
	}
	if boreRun >= edgefinish.LongEdgeRun {
		perimeter += boreRun
	}
	if eyeRun := math.Pi * dm.EyeHoleDiameter; eyeRun >= edgefinish.LongEdgeRun {
		perimeter += eyeRun
	}
	want := pr.Area*p.Height - 0.25*perimeter
	tol := 0.02*0.25*perimeter + fillets
	require.InDelta(t, want, v, tol)
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
