package turbinefan_test

import (
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// fanCase is one parameter case. Every case changes the hub's bore, the rotor's size or the blades' pitch. A case that
// does not set RotorDiameter builds at DefaultRotorDiameter (192).
type fanCase struct {
	name   string
	params func() turbinefan.Params
}

var fanCases = []fanCase{
	{"605ZZ", turbinefan.Params605ZZ},
	{"608ZZ", turbinefan.Params608ZZ},
	{"regular-bore", func() turbinefan.Params {
		p := turbinefan.Params605ZZ()
		p.BoreClearance = 0.35
		return p
	}},
	{"steep", func() turbinefan.Params {
		p := turbinefan.Params608ZZ()
		p.BladePitch = turbinefan.MaxBladePitch
		return p
	}},
	{"flat", func() turbinefan.Params {
		p := turbinefan.Params605ZZ()
		p.BladePitch = turbinefan.MinBladePitch
		return p
	}},
	// compact is the smallest rotor, which rebuilds the fan as it was before RotorDiameter existed.
	{"compact", func() turbinefan.Params {
		p := turbinefan.Params608ZZ()
		p.RotorDiameter = turbinefan.MinRotorDiameter
		return p
	}},
	// large is the largest rotor at the steepest pitch.
	{"large", func() turbinefan.Params {
		p := turbinefan.Params605ZZ()
		p.RotorDiameter = turbinefan.MaxRotorDiameter
		p.BladePitch = turbinefan.MaxBladePitch
		return p
	}},
	{"placed", func() turbinefan.Params {
		p := turbinefan.Params605ZZ()
		f, _ := r3.NewFrame(r3.Vec{X: 10, Y: -5, Z: 7}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		p.Placement = &f
		return p
	}},
}

// clutchCases are the clutch-mount cases, each from ParamsForClutch with the clutch's own frame.
var clutchCases = []fanCase{
	{"clutch-605ZZ", func() turbinefan.Params { return clutchParams(onewayclutch.DefaultParams()) }},
	{"clutch-608ZZ-left", func() turbinefan.Params {
		return clutchParams(onewayclutch.Params{Shaft: "608ZZ", Freewheel: onewayclutch.CounterClockwise, BoreClearance: 0.05})
	}},
}

// clutchParams returns ParamsForClutch(c) in the fan's local frame, and panics on an error.
func clutchParams(c onewayclutch.Params) turbinefan.Params {
	p, err := turbinefan.ParamsForClutch(c)
	if err != nil {
		panic(err)
	}
	p.Placement = nil
	return p
}

func caseNamed(t *testing.T, name string) fanCase {
	t.Helper()
	for _, c := range append(slices.Clone(fanCases), clutchCases...) {
		if c.name == name {
			return c
		}
	}
	require.FailNow(t, "no case "+name)
	return fanCase{}
}

type builtCase struct {
	m   *turbinefan.Model
	doc *decad.Document
}

var (
	builtMu    sync.Mutex
	builtCases = map[string]builtCase{}
)

// built builds a case on first use and shares it between the tests: Build takes a while and the tests only read its
// bodies.
func built(t *testing.T, c fanCase) (*turbinefan.Model, *decad.Document) {
	t.Helper()
	builtMu.Lock()
	defer builtMu.Unlock()
	if b, ok := builtCases[c.name]; ok {
		return b.m, b.doc
	}
	doc := turbinefan.NewDocument()
	m, err := turbinefan.Build(t.Context(), doc, c.params())
	require.NoError(t, err)
	builtCases[c.name] = builtCase{m: m, doc: doc}
	return m, doc
}

func TestTurbineFanBuild(t *testing.T) {
	for _, c := range fanCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			require.Len(t, doc.Bodies(), 12)
			require.NoError(t, doc.VerifyContainers().Err())
			want := map[string][]*decad.Body{
				turbinefan.HubContainerName:    {m.Hub},
				turbinefan.BladesContainerName: {m.Blades[0], m.Blades[1], m.Blades[2]},
				turbinefan.BoltsContainerName: {
					m.BladeBolts[0].Body, m.BladeBolts[1].Body, m.BladeBolts[2].Body, m.SetScrew.Body},
				turbinefan.NutsContainerName: {
					m.BladeNuts[0].Body, m.BladeNuts[1].Body, m.BladeNuts[2].Body, m.SetScrewNut.Body},
			}
			children := doc.Root().Children()
			require.Len(t, children, len(want))
			for _, box := range children {
				bodies, ok := want[box.Name()]
				require.True(t, ok, "unexpected container %q", box.Name())
				require.Equal(t, bodies, box.Bodies(), box.Name())
				for _, b := range bodies {
					require.True(t, b.IsSolid(), box.Name())
				}
			}
		})
	}
}

// TestTurbineFanSketches requires every sketch of every case (the compact, default and large rotors), the hub's and the
// blades' and each bolt's and nut's, fully constrained: 28 from BuildParts (the hub's two adapter hole sketches and two
// adapter slot sketches among them) and 20 from Build. The clutch mount's sketches are in TestTurbineFanClutchSketches.
func TestTurbineFanSketches(t *testing.T) {
	for _, c := range fanCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			require.Len(t, m.Sketches, 28)
			for i, s := range m.Sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
			}
			for i, b := range m.BladeBolts {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Profile), "blade bolt %d profile", i+1)
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Socket), "blade bolt %d socket", i+1)
			}
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.SetScrew.Profile))
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.SetScrew.Socket))
			for i, n := range append(m.BladeNuts[:], m.SetScrewNut) {
				for _, s := range []*sketch.Sketch{n.Hexagon, n.LowerChamfer, n.UpperChamfer} {
					require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "nut %d", i)
				}
			}
		})
	}
}

// meshOf tessellates body at tol with every verification on.
func meshOf(t *testing.T, body *decad.Body, tol float64) *decad.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	return mesh
}

// section cuts body by the plane f at the finest chord the body allows.
func section(t *testing.T, body *decad.Body, f r3.Frame) decad.SectionOutline {
	t.Helper()
	var o decad.SectionOutline
	var err error
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		if o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			return o
		}
	}
	require.NoError(t, err)
	return o
}

// atZ is the plane z = const of the local frame.
func atZ(t *testing.T, z float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}

// atX is the plane x = const of the local frame, normal +X, across the clamp bolt's line.
func atX(t *testing.T, x float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}

// atY is the plane y = const of the local frame, normal +Y, with U along +Z and V along +X.
func atY(t *testing.T, y float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Y: y}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	require.NoError(t, err)
	return f
}

// crossingsAtX returns, sorted, the y values within [lo, hi] where the section's loop edges cross the line x = x0.
func crossingsAtX(o decad.SectionOutline, x0, lo, hi float64) []float64 {
	var out []float64
	for _, l := range o.Loops {
		for i := range l {
			a, b := l[i], l[(i+1)%len(l)]
			if (a.X-x0)*(b.X-x0) >= 0 || a.X == b.X {
				continue
			}
			y := a.Y + (x0-a.X)/(b.X-a.X)*(b.Y-a.Y)
			if y >= lo && y <= hi {
				out = append(out, y)
			}
		}
	}
	slices.Sort(out)
	return out
}

// radial is the plane through the shaft axis and blade k's radial line, U along the line and V along +Z.
func radial(t *testing.T, k int) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{}, bladeDir(k), r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}

// bladeDir is the unit vector along blade k's radial line, k = 1, 2, 3.
func bladeDir(k int) r3.Vec {
	a := turbinefan.BladeAngle(k)
	return r3.Vec{X: math.Cos(a), Y: math.Sin(a)}
}

// clearance returns the polyline clearance between a and b in the plane f, failing on an overlap.
func clearance(t *testing.T, a, b *decad.Body, f r3.Frame) decad.SectionClearance {
	t.Helper()
	oa, ob := section(t, a, f), section(t, b, f)
	require.NotEmpty(t, oa.Loops)
	require.NotEmpty(t, ob.Loops)
	c, err := oa.Clearance(ob)
	require.NoError(t, err)
	require.False(t, c.Overlaps, "overlap at %v", f.Origin())
	return c
}

// requireNoOverlap requires that no part shares volume with host in the section of both at frame f; name tells the
// case in a failure.
func requireNoOverlap(t *testing.T, host *decad.Body, f r3.Frame, name string, parts ...*decad.Body) {
	t.Helper()
	a := section(t, host, f)
	for _, part := range parts {
		b := section(t, part, f)
		if len(a.Loops) == 0 || len(b.Loops) == 0 {
			continue
		}
		cl, err := a.Clearance(b)
		require.NoError(t, err)
		require.False(t, cl.Overlaps, "%s: overlap", name)
	}
}

// distanceToLoop returns the least distance from v to the closed polyline l in the XY plane.
func distanceToLoop(v r3.Vec, l []r3.Vec) float64 {
	least := math.Inf(1)
	for i := range l {
		a, b := l[i], l[(i+1)%len(l)]
		ab := r3.Vec{X: b.X - a.X, Y: b.Y - a.Y}
		av := r3.Vec{X: v.X - a.X, Y: v.Y - a.Y}
		u := 0.0
		if d := ab.Dot(ab); d > 0 {
			u = math.Max(0, math.Min(1, av.Dot(ab)/d))
		}
		least = math.Min(least, math.Hypot(av.X-u*ab.X, av.Y-u*ab.Y))
	}
	return least
}

// inSection reports whether the point (u, v) of the plane, given in the section's own coordinates by uv, lies in the
// section's material, by the even-odd rule over its loops.
func inSection(o decad.SectionOutline, uv func(r3.Vec) (float64, float64), u, v float64) bool {
	crossings := 0
	for _, l := range o.Loops {
		for i := range l {
			au, av := uv(l[i])
			bu, bv := uv(l[(i+1)%len(l)])
			if (av > v) != (bv > v) && u < au+(v-av)/(bv-av)*(bu-au) {
				crossings++
			}
		}
	}
	return crossings%2 == 1
}

// boreCases are the cases that differ in the bore; pitchCases the ones that differ in the blades' pitch. Neither holds
// the placed case, whose bodies the placement test compares with the 605ZZ case.
var (
	boreCases  = []string{"605ZZ", "608ZZ", "regular-bore", "compact", "large"}
	pitchCases = []string{"605ZZ", "steep", "flat", "compact", "large"}
)

// eachCase runs fn as a subtest for each named case.
func eachCase(t *testing.T, names []string, fn func(t *testing.T, c fanCase, m *turbinefan.Model)) {
	t.Helper()
	for _, name := range names {
		c := caseNamed(t, name)
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			fn(t, c, m)
		})
	}
}

// TestTurbineFanBounds checks the hub's bounds at the compact (14 mm radius, 20 long), default (18.9, 30) and large
// (18.9, 35) rotors and tessellates the hub, a blade, the set screw and its nut with every verification on.
func TestTurbineFanBounds(t *testing.T) {
	eachCase(t, []string{"compact", "605ZZ", "large"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		box, err := m.Hub.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-3
		require.InDelta(t, -sz.HubRadius, box.Min.X, tol)
		require.InDelta(t, sz.HubRadius, box.Max.X, tol)
		require.InDelta(t, -sz.HubRadius, box.Min.Y, tol)
		require.InDelta(t, sz.HubRadius, box.Max.Y, tol)
		require.InDelta(t, 0, box.Min.Z, tol)
		require.InDelta(t, sz.HubLength, box.Max.Z, tol)
		if c.name == "compact" {
			// The compact hub and blade volumes pin this 128 mm build.
			hv, err := m.Hub.Volume()
			require.NoError(t, err)
			bv, err := m.Blades[0].Volume()
			require.NoError(t, err)
			require.InEpsilon(t, 9703.740630, hv.Value.Base(), 1e-6, "hub volume")
			require.InEpsilon(t, 1968.166101784, bv.Value.Base(), 1e-6, "blade volume")
		}
		for name, item := range map[string]struct {
			body *decad.Body
			tol  float64
		}{
			"hub": {m.Hub, 0.05}, "blade": {m.Blades[0], 0.1}, "set screw": {m.SetScrew.Body, 0.01},
			"set screw nut": {m.SetScrewNut.Body, 0.01},
		} {
			mesh := meshOf(t, item.body, item.tol)
			require.True(t, mesh.BoundaryVerified(), name)
		}
	})
}

// anchoredTransform returns the move from the local frame that puts the point (0, 0, anchorZ) on frame f.
func anchoredTransform(t *testing.T, f r3.Frame, anchorZ float64) r3.Transform {
	t.Helper()
	outer, err := r3.FromFrame(f)
	require.NoError(t, err)
	moved, err := r3.NewFrame(outer.Apply(r3.Vec{Z: -anchorZ}), outer.ApplyDir(r3.Vec{X: 1}), outer.ApplyDir(r3.Vec{Y: 1}))
	require.NoError(t, err)
	xf, err := r3.FromFrame(moved)
	require.NoError(t, err)
	return xf
}

// TestTurbineFanPlacement checks that the placed case's bodies and sites are the default's moved so that the hub's
// front face (z = Size.HubLength) lies on the frame's origin.
func TestTurbineFanPlacement(t *testing.T) {
	local, _ := built(t, caseNamed(t, "605ZZ"))
	placed, _ := built(t, caseNamed(t, "placed"))
	frame := *caseNamed(t, "placed").params().Placement
	xf := anchoredTransform(t, frame, local.Dimensions.Size.HubLength)
	pairs := map[string][2]*decad.Body{
		"hub": {local.Hub, placed.Hub}, "blade 1": {local.Blades[0], placed.Blades[0]},
		"blade 3": {local.Blades[2], placed.Blades[2]}, "blade bolt": {local.BladeBolts[1].Body, placed.BladeBolts[1].Body},
		"blade nut": {local.BladeNuts[2].Body, placed.BladeNuts[2].Body},
		"set screw": {local.SetScrew.Body, placed.SetScrew.Body}, "set screw nut": {local.SetScrewNut.Body, placed.SetScrewNut.Body},
	}
	for name, pair := range pairs {
		lv, err := pair[0].Volume()
		require.NoError(t, err)
		pv, err := pair[1].Volume()
		require.NoError(t, err)
		require.InDelta(t, lv.Value.Base(), pv.Value.Base(), 1e-6*lv.Value.Base(), name)
		lb, err := pair[0].Bounds()
		require.NoError(t, err)
		pb, err := pair[1].Bounds()
		require.NoError(t, err)
		// The frame maps local X to world Y, Y to Z and Z to X.
		tol := lb.Bound.Base() + pb.Bound.Base() + 1e-6
		lo, hi := xf.Apply(lb.Min), xf.Apply(lb.Max)
		require.InDelta(t, math.Min(lo.X, hi.X), pb.Min.X, tol, name)
		require.InDelta(t, math.Max(lo.X, hi.X), pb.Max.X, tol, name)
		require.InDelta(t, math.Min(lo.Y, hi.Y), pb.Min.Y, tol, name)
		require.InDelta(t, math.Max(lo.Z, hi.Z), pb.Max.Z, tol, name)
	}
	require.InDelta(t, 0, placed.SetScrewSite.Head.Sub(xf.Apply(local.SetScrewSite.Head)).Len(), 1e-9)
	require.InDelta(t, 0, placed.SetScrewSite.Dir.Sub(xf.ApplyDir(local.SetScrewSite.Dir)).Len(), 1e-9)
	for i := range local.BladeSites {
		require.InDelta(t, 0, placed.BladeSites[i].Head.Sub(xf.Apply(local.BladeSites[i].Head)).Len(), 1e-9)
		require.InDelta(t, 0, placed.BladeSites[i].Dir.Sub(xf.ApplyDir(local.BladeSites[i].Dir)).Len(), 1e-9)
	}
}

// TestTurbineFanPlacementPoint checks that a ShaftMount hub's front face lies on the frame's origin plane and the hub
// extends Size.HubLength back along the frame's axis.
func TestTurbineFanPlacementPoint(t *testing.T) {
	p := turbinefan.Params605ZZ()
	f, err := r3.NewFrame(r3.Vec{X: 3, Y: 4, Z: 5}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	p.Placement = &f
	m, err := turbinefan.Build(t.Context(), turbinefan.NewDocument(), p)
	require.NoError(t, err)
	b, err := m.Hub.Bounds()
	require.NoError(t, err)
	tol := b.Bound.Base() + 1e-6
	require.InDelta(t, 5, b.Max.Z, tol, "front face on the frame origin")
	require.InDelta(t, 5-m.Dimensions.Size.HubLength, b.Min.Z, tol, "back face HubLength behind it")
	require.InDelta(t, 3, (b.Min.X+b.Max.X)/2, tol, "hub on the axis through the origin")
}

// TestTurbineFanEnvelope requires every blade vertex to lie between the hub's faces (0 <= z <= HubLength) and inside the
// tip radius, at both ends of the pitch range, for the compact, default and large rotors.
func TestTurbineFanEnvelope(t *testing.T) {
	eachCase(t, []string{"steep", "flat", "605ZZ", "compact", "large"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		for k, blade := range m.Blades {
			lo, hi, far := math.Inf(1), math.Inf(-1), 0.0
			for _, v := range meshOf(t, blade, 0.1).Vertices() {
				lo, hi, far = math.Min(lo, v.Z), math.Max(hi, v.Z), math.Max(far, math.Hypot(v.X, v.Y))
			}
			require.GreaterOrEqual(t, lo, 0.0, "blade %d: lowest z", k+1)
			require.LessOrEqual(t, hi, sz.HubLength, "blade %d: highest z", k+1)
			require.LessOrEqual(t, far, sz.TipRadius+0.5, "blade %d: radius", k+1)
			t.Logf("%s blade %d: z from %.2f to %.2f, radius %.2f", c.name, k+1, lo, hi, far)
		}
	})
}
