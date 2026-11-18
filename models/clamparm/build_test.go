package clamparm_test

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// asmCase is one representative case of instructions.md. joint is the joint (0 for J1 to 3 for J4) whose limit the
// case poses, or -1.
type asmCase struct {
	name   string
	params clamparm.Params
	joint  int
}

func asmPlacement(t testing.TB) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	return &f
}

// asmCases lists every representative case of instructions.md.
func asmCases(t testing.TB) []asmCase {
	t.Helper()
	with := func(edit func(*clamparm.Params)) clamparm.Params {
		p := clamparm.DefaultParams()
		edit(&p)
		return p
	}
	return []asmCase{
		{"default", clamparm.DefaultParams(), -1},
		{"short", with(func(p *clamparm.Params) { p.LinkLength = 80 }), -1},
		{"long", with(func(p *clamparm.Params) { p.LinkLength = 150 }), -1},
		{"thin-desk", with(func(p *clamparm.Params) { p.Opening = 10 }), -1},
		{"thick-desk", with(func(p *clamparm.Params) { p.Opening = 40 }), -1},
		{"raised", with(func(p *clamparm.Params) { p.Shoulder, p.Elbow, p.Wrist = 110, -70, -40 }), -1},
		{"placed", with(func(p *clamparm.Params) { p.Placement = asmPlacement(t) }), -1},
		{"pan-max", with(func(p *clamparm.Params) { p.Pan = clamparm.J1Max }), 0},
		{"pan-min", with(func(p *clamparm.Params) { p.Pan = -clamparm.J1Max }), 0},
		{"shoulder-min", with(func(p *clamparm.Params) { p.Shoulder, p.Opening = clamparm.ShoulderMin, 40 }), 1},
		{"shoulder-max", with(func(p *clamparm.Params) { p.Shoulder = clamparm.JointMax }), 1},
		{"elbow-min", with(func(p *clamparm.Params) { p.Elbow = -clamparm.JointMax }), 2},
		{"elbow-max", with(func(p *clamparm.Params) { p.Elbow = clamparm.JointMax }), 2},
		{"wrist-min", with(func(p *clamparm.Params) { p.Wrist = clamparm.WristMin }), 3},
		{"wrist-max", with(func(p *clamparm.Params) { p.Wrist = clamparm.WristMax }), 3},
	}
}

// asmBuilt is one built case. Each case is built once and shared by every test that needs it.
type asmBuilt struct {
	once  sync.Once
	doc   *decad.Document
	model *clamparm.Model
	err   error
}

var asmCache sync.Map // case name -> *asmBuilt

// asmBuild returns the case's document and model, building it on first use.
func asmBuild(t *testing.T, c asmCase) (*decad.Document, *clamparm.Model) {
	t.Helper()
	v, _ := asmCache.LoadOrStore(c.name, &asmBuilt{})
	b := v.(*asmBuilt)
	b.once.Do(func() {
		b.doc = clamparm.NewDocument()
		b.model, b.err = clamparm.Build(t.Context(), b.doc, c.params)
	})
	require.NoError(t, b.err, "case %s", c.name)
	return b.doc, b.model
}

var asmFlatCache sync.Map // case name -> *asmBuilt

// asmBuildFlat builds the case again and moves every body to the document root. decad's Intersect refuses operands
// in different containers, so the overlap checks use this copy; the containers are checked on the shared build.
func asmBuildFlat(t *testing.T, c asmCase) *clamparm.Model {
	t.Helper()
	v, _ := asmFlatCache.LoadOrStore(c.name, &asmBuilt{})
	b := v.(*asmBuilt)
	b.once.Do(func() {
		b.doc = clamparm.NewDocument()
		b.model, b.err = clamparm.Build(t.Context(), b.doc, c.params)
		if b.err != nil {
			return
		}
		for _, body := range b.doc.Bodies() {
			if b.err = b.doc.Root().AddBody(body); b.err != nil {
				return
			}
		}
	})
	require.NoError(t, b.err, "case %s", c.name)
	return b.model
}

// asmPrinted lists the printed parts in container order with their names.
func asmPrinted(m *clamparm.Model) ([]string, []clamparm.Part) {
	names := []string{"frame", "pad", "swivel", "link A", "link B", "head", "adapter", "knob J1", "knob J2", "knob J3",
		"knob J4", "screw knob", "stud knob"}
	parts := []clamparm.Part{m.Clamp, m.Pad, m.Swivel, m.Links[0], m.Links[1], m.Head, m.Adapter, m.Knobs[0], m.Knobs[1],
		m.Knobs[2], m.Knobs[3], m.ScrewKnobs[0], m.ScrewKnobs[1]}
	return names, parts
}

// asmBolts and asmNuts list the hardware bodies in model order with their names.
func asmBolts(m *clamparm.Model) ([]string, []*decad.Body) {
	names := []string{"bolt J1", "bolt J2", "bolt J3", "bolt J4", "stud", "clamp screw"}
	bodies := make([]*decad.Body, len(m.Bolts))
	for i, b := range m.Bolts {
		bodies[i] = b.Body
	}
	return names, bodies
}

func asmNuts(m *clamparm.Model) ([]string, []*decad.Body) {
	names := []string{"nut J1", "nut J2", "nut J3", "nut J4", "jaw nut", "adapter nut"}
	bodies := make([]*decad.Body, len(m.Nuts))
	for i, n := range m.Nuts {
		bodies[i] = n.Body
	}
	return names, bodies
}

// asmWorld is the transform Placement applies to the case's world frame.
func asmWorld(t testing.TB, p clamparm.Params) r3.Transform {
	t.Helper()
	if p.Placement == nil {
		return r3.Identity()
	}
	xf, err := r3.FromFrame(*p.Placement)
	require.NoError(t, err)
	return xf
}

type asmMeshKey struct {
	body *decad.Body
	tol  float64
}

var asmMeshCache sync.Map // asmMeshKey -> *decad.Mesh

// asmTessellate returns the body's mesh at tol, tessellating once.
func asmTessellate(t testing.TB, body *decad.Body, tol float64) *decad.Mesh {
	t.Helper()
	key := asmMeshKey{body, tol}
	if v, ok := asmMeshCache.Load(key); ok {
		return v.(*decad.Mesh)
	}
	mesh, err := body.Tessellate(context.Background(), units.Millimeters(tol))
	require.NoError(t, err)
	v, _ := asmMeshCache.LoadOrStore(key, mesh)
	return v.(*decad.Mesh)
}

func TestClampArmBuildCases(t *testing.T) {
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			doc, m := asmBuild(t, c)
			names, parts := asmPrinted(m)
			boltNames, bolts := asmBolts(m)
			nutNames, nuts := asmNuts(m)

			require.Len(t, doc.Bodies(), 25)
			for i, part := range parts {
				require.True(t, part.Body.IsSolid(), names[i])
				requireFullyConstrained(t, part.Sketches)
			}
			for i, body := range append(append([]*decad.Body{}, bolts...), nuts...) {
				require.True(t, body.IsSolid(), "hardware %d", i)
			}
			for i, b := range m.Bolts {
				requireFullyConstrained(t, []*sketch.Sketch{b.Head, b.Shank})
				require.NotEmpty(t, boltNames[i])
			}
			for i, n := range m.Nuts {
				sketches := []*sketch.Sketch{n.Hexagon}
				for _, s := range []*sketch.Sketch{n.LowerChamfer, n.UpperChamfer} {
					if s != nil {
						sketches = append(sketches, s)
					}
				}
				requireFullyConstrained(t, sketches)
				require.NotEmpty(t, nutNames[i])
			}
			// The knobs record the outline, the nut pocket's hexagon and one mark; the screw knobs the outline, the
			// pocket's hexagon and two marks.
			for _, k := range m.Knobs {
				require.Len(t, k.Sketches, 3)
			}
			for _, k := range m.ScrewKnobs {
				require.Len(t, k.Sketches, 4)
			}

			want := []struct {
				name   string
				bodies []*decad.Body
			}{
				{clamparm.FrameContainerName, []*decad.Body{m.Clamp.Body}},
				{clamparm.PadContainerName, []*decad.Body{m.Pad.Body}},
				{clamparm.SwivelContainerName, []*decad.Body{m.Swivel.Body}},
				{clamparm.LinksContainerName, []*decad.Body{m.Links[0].Body, m.Links[1].Body}},
				{clamparm.HeadContainerName, []*decad.Body{m.Head.Body}},
				{clamparm.AdapterContainerName, []*decad.Body{m.Adapter.Body}},
				{clamparm.KnobsContainerName, []*decad.Body{m.Knobs[0].Body, m.Knobs[1].Body, m.Knobs[2].Body,
					m.Knobs[3].Body}},
				{clamparm.ScrewKnobsContainerName, []*decad.Body{m.ScrewKnobs[0].Body, m.ScrewKnobs[1].Body}},
				{clamparm.BoltsContainerName, bolts},
				{clamparm.NutsContainerName, nuts},
			}
			children := doc.Root().Children()
			require.Len(t, children, len(want))
			for i, w := range want {
				require.Equal(t, w.name, children[i].Name())
				require.ElementsMatch(t, w.bodies, children[i].Bodies(), w.name)
			}
			require.Empty(t, doc.Root().Bodies())
			require.NoError(t, doc.VerifyContainers().Err())
		})
	}
}

func TestClampArmBuildContainerNames(t *testing.T) {
	require.Equal(t, "Frame", clamparm.FrameContainerName)
	require.Equal(t, "Pad", clamparm.PadContainerName)
	require.Equal(t, "Swivel", clamparm.SwivelContainerName)
	require.Equal(t, "Links", clamparm.LinksContainerName)
	require.Equal(t, "Head", clamparm.HeadContainerName)
	require.Equal(t, "Adapter", clamparm.AdapterContainerName)
	require.Equal(t, "Knobs", clamparm.KnobsContainerName)
	require.Equal(t, "Screw Knobs", clamparm.ScrewKnobsContainerName)
	require.Equal(t, "Bolts", clamparm.BoltsContainerName)
	require.Equal(t, "Nuts", clamparm.NutsContainerName)
}

func TestClampArmBuildParts(t *testing.T) {
	doc := clamparm.NewDocument()
	p := clamparm.DefaultParams()
	parts, err := clamparm.BuildParts(t.Context(), doc, p)
	require.NoError(t, err)
	// BuildParts files nothing: no container exists and the 13 printed bodies sit in the root.
	require.Empty(t, doc.Root().Children())
	require.Len(t, doc.Root().Bodies(), 13)
	d, err := p.Derive()
	require.NoError(t, err)
	require.Equal(t, d, parts.Dimensions)
	require.Equal(t, d.Head, parts.Head.Frame)
	require.Equal(t, d.Interface, parts.Adapter.Frame)
	require.Equal(t, d.PadFrame, parts.Pad.Frame)
	requireVec(t, r3.NewVec(0, 1, 0), parts.Clamp.PrintUp)
	requireVec(t, r3.NewVec(0, 0, -1), parts.Pad.PrintUp)
	requireVec(t, r3.NewVec(0, 0, 1), parts.Swivel.PrintUp)
	for _, k := range parts.Knobs {
		requireVec(t, k.Frame.N(), k.PrintUp)
	}
	for _, k := range parts.ScrewKnobs {
		requireVec(t, k.Frame.N(), k.PrintUp)
	}
}

// asmUnion is the bounding box of the bodies.
func asmUnion(t testing.TB, bodies []*decad.Body) (r3.Vec, r3.Vec) {
	t.Helper()
	lo := r3.NewVec(math.Inf(1), math.Inf(1), math.Inf(1))
	hi := r3.NewVec(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, b := range bodies {
		box, err := b.Bounds()
		require.NoError(t, err)
		lo = r3.NewVec(math.Min(lo.X, box.Min.X), math.Min(lo.Y, box.Min.Y), math.Min(lo.Z, box.Min.Z))
		hi = r3.NewVec(math.Max(hi.X, box.Max.X), math.Max(hi.Y, box.Max.Y), math.Max(hi.Z, box.Max.Z))
	}
	return lo, hi
}

// TestClampArmBuildDefaultBounds pins the assembly's extent in the default pose. Each value follows from the derived
// frames: the spine, the screw knob's outer face, the knobs' outer faces and the adapter disc.
func TestClampArmBuildDefaultBounds(t *testing.T) {
	c := asmCases(t)[0]
	_, m := asmBuild(t, c)
	d := m.Dimensions
	var all []*decad.Body
	_, parts := asmPrinted(m)
	for _, part := range parts {
		all = append(all, part.Body)
	}
	_, bolts := asmBolts(m)
	_, nuts := asmNuts(m)
	all = append(append(all, bolts...), nuts...)
	lo, hi := asmUnion(t, all)

	// The spine's outer face, the adapter disc's side (radius 20), the joint knobs' outer faces (y = 22) and the
	// screw knob's outer face at -Opening - 77.5.
	require.InDelta(t, -14, lo.X, boundTol)
	require.InDelta(t, -20, lo.Y, boundTol)
	require.InDelta(t, 22, hi.Y, boundTol)
	require.InDelta(t, -c.params.Opening-77.5, lo.Z, boundTol)
	// The adapter disc (radius 20) about the stud axis is the farthest part in +X; the head level there.
	require.InDelta(t, d.Interface.Origin().X+20, hi.X, boundTol)
	// The J3 knob is the highest: its lobe, 11 + 4 from the axis, points nearest to straight up.
	u := d.KnobFrames[2].U()
	phi := math.Atan2(u.Z, u.X)
	top := math.Inf(-1)
	for k := range 6 {
		top = math.Max(top, 11*math.Sin(phi+float64(k)*math.Pi/3)+4)
	}
	require.InDelta(t, d.Joints[2].Z+top, hi.Z, 0.05)
}

// TestClampArmBuildVerifyAll tessellates the frame, the swivel, the links and the head with full verification.
func TestClampArmBuildVerifyAll(t *testing.T) {
	_, m := asmBuild(t, asmCases(t)[0])
	for name, body := range map[string]*decad.Body{
		"frame": m.Clamp.Body, "swivel": m.Swivel.Body, "link A": m.Links[0].Body, "link B": m.Links[1].Body,
		"head": m.Head.Body,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := body.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
		})
	}
}

// TestClampArmBuildPlaced requires every body of the placed case to be the default's moved by the placement frame.
func TestClampArmBuildPlaced(t *testing.T) {
	cases := asmCases(t)
	_, def := asmBuild(t, cases[0])
	var placedCase asmCase
	for _, c := range cases {
		if c.name == "placed" {
			placedCase = c
		}
	}
	_, placed := asmBuild(t, placedCase)
	xf := asmWorld(t, placedCase.params)

	require.Equal(t, def.Dimensions, placed.Dimensions, "Dimensions are the model's world frame before Placement")
	_, defParts := asmPrinted(def)
	_, placedParts := asmPrinted(placed)
	var pairs [][2]*decad.Body
	for i, part := range defParts {
		want, err := r3.FromFrame(part.Frame)
		require.NoError(t, err)
		got, err := r3.FromFrame(placedParts[i].Frame)
		require.NoError(t, err)
		moved, err := want.Then(xf)
		require.NoError(t, err)
		for _, v := range []r3.Vec{r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1)} {
			requireVec(t, moved.Apply(v), got.Apply(v))
		}
		requireVec(t, xf.ApplyDir(part.PrintUp), placedParts[i].PrintUp)
		pairs = append(pairs, [2]*decad.Body{part.Body, placedParts[i].Body})
	}
	_, defBolts := asmBolts(def)
	_, placedBolts := asmBolts(placed)
	_, defNuts := asmNuts(def)
	_, placedNuts := asmNuts(placed)
	for i := range defBolts {
		pairs = append(pairs, [2]*decad.Body{defBolts[i], placedBolts[i]}, [2]*decad.Body{defNuts[i], placedNuts[i]})
	}
	for _, pair := range pairs {
		dv, err := pair[0].Volume()
		require.NoError(t, err)
		pv, err := pair[1].Volume()
		require.NoError(t, err)
		require.InDelta(t, dv.Value.Base(), pv.Value.Base(), dv.Bound.Base()+pv.Bound.Base()+1e-6)
		dc, err := pair[0].Centroid()
		require.NoError(t, err)
		pc, err := pair[1].Centroid()
		require.NoError(t, err)
		want := xf.Apply(dc.Value)
		tol := dc.Bound.Base() + pc.Bound.Base() + 1e-6
		require.InDelta(t, want.X, pc.Value.X, tol)
		require.InDelta(t, want.Y, pc.Value.Y, tol)
		require.InDelta(t, want.Z, pc.Value.Z, tol)
	}
}

// TestClampArmBuildInvalid requires every rejected input to return an error and add no body.
func TestClampArmBuildInvalid(t *testing.T) {
	bad := func(edit func(*clamparm.Params)) clamparm.Params {
		p := clamparm.DefaultParams()
		edit(&p)
		return p
	}
	notFrame := r3.Frame{}
	cases := []struct {
		name string
		p    clamparm.Params
		want string
	}{
		{"nan link length", bad(func(p *clamparm.Params) { p.LinkLength = math.NaN() }), "LinkLength must be finite"},
		{"short link", bad(func(p *clamparm.Params) { p.LinkLength = 79 }), "LinkLength must be from 80 to 150"},
		{"opening", bad(func(p *clamparm.Params) { p.Opening = 41 }), "Opening must be from 10 to 40"},
		{"pan", bad(func(p *clamparm.Params) { p.Pan = clamparm.J1Max + 1 }), "Pan must be from"},
		{"shoulder", bad(func(p *clamparm.Params) { p.Shoulder = clamparm.ShoulderMin - 1 }), "Shoulder must be from"},
		{"elbow", bad(func(p *clamparm.Params) { p.Elbow = clamparm.JointMax + 1 }), "Elbow must be from"},
		{"wrist", bad(func(p *clamparm.Params) { p.Wrist = clamparm.WristMax + 1 }), "Wrist must be from"},
		{"placement", bad(func(p *clamparm.Params) { p.Placement = &notFrame }), "Placement is not a valid frame"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := clamparm.NewDocument()
			_, err := clamparm.Build(t.Context(), doc, tc.p)
			require.ErrorContains(t, err, "clamparm: ")
			require.ErrorContains(t, err, tc.want)
			require.Empty(t, doc.Bodies())
			_, err = clamparm.BuildParts(t.Context(), doc, tc.p)
			require.ErrorContains(t, err, tc.want)
			require.Empty(t, doc.Bodies())
		})
	}

	t.Run("nil context", func(t *testing.T) {
		doc := clamparm.NewDocument()
		//nolint:staticcheck // a nil context is the case under test
		_, err := clamparm.Build(nil, doc, clamparm.DefaultParams())
		require.ErrorContains(t, err, "clamparm: context must not be nil")
		require.Empty(t, doc.Bodies())
	})
	t.Run("nil document", func(t *testing.T) {
		_, err := clamparm.Build(t.Context(), nil, clamparm.DefaultParams())
		require.ErrorContains(t, err, "clamparm: document must not be nil")
	})
	t.Run("cancelled context", func(t *testing.T) {
		doc := clamparm.NewDocument()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := clamparm.Build(ctx, doc, clamparm.DefaultParams())
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorContains(t, err, "clamparm: before creating geometry")
		require.Empty(t, doc.Bodies())
	})
}
