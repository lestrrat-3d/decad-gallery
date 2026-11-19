package turbinefan_test

import (
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
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
