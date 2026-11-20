package spiralspring_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/sketch"
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
