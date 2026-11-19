package clamparm_test

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
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

var asmCache sync.Map

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

var asmFlatCache sync.Map

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

var asmMeshCache sync.Map

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
