package sevensegment_test

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// build caches one immutable assembly for tests in this process.
type build[T any] struct {
	once sync.Once
	val  T
	err  error
}

func (b *build[T]) built(t *testing.T, f func(context.Context) (T, error)) T {
	t.Helper()
	b.once.Do(func() { b.val, b.err = f(context.Background()) })
	require.NoError(t, b.err)
	return b.val
}

// defaultModel is the default display and its document.
type defaultModel struct {
	doc   *decad.Document
	model *sevensegment.Model
}

var defaultBuild build[defaultModel]

func buildDefault(ctx context.Context) (defaultModel, error) {
	doc := sevensegment.NewDocument()
	m, err := sevensegment.Build(ctx, doc, sevensegment.DefaultParams())
	return defaultModel{doc, m}, err
}

// builtDefault returns the default display, built once and shared: the tests only read it.
func builtDefault(t *testing.T) (*decad.Document, *sevensegment.Model) {
	t.Helper()
	d := defaultBuild.built(t, buildDefault)
	return d.doc, d.model
}

// printed returns the printed bodies of m by name.
func printed(m *sevensegment.Model) map[string]*decad.Body {
	out := map[string]*decad.Body{
		"front wall": m.FrontWall, "rear wall": m.RearWall, "guide plate": m.GuidePlate,
		"cradle plate": m.CradlePlate, "top plate": m.TopPlate, "shaft": m.Shaft, "hub": m.Hub, "rotor": m.Rotor, "housing": m.Housing,
		"button": m.Button,
	}
	for _, s := range sevensegment.Segments() {
		out["cam "+s.String()] = m.Cams[s]
		out["slider "+s.String()] = m.Sliders[s]
		out["yellow face "+s.String()] = m.Flags[s].Yellow
		out["black body "+s.String()] = m.Flags[s].Black
		out["pivot block "+s.String()+" -u"] = m.PivotBlocks[s][0]
		out["pivot block "+s.String()+" +u"] = m.PivotBlocks[s][1]
	}
	return out
}

var digit0Build build[*sevensegment.Parts]

func buildDigit0(ctx context.Context) (*sevensegment.Parts, error) {
	return sevensegment.BuildParts(ctx, sevensegment.NewDocument(), sevensegment.Params{Digit: 0})
}

// builtDigit0 returns the printed parts at digit 0, built once and shared: the tests read them and move copies of their
// meshes, never the bodies.
func builtDigit0(t *testing.T) *sevensegment.Parts {
	t.Helper()
	return digit0Build.built(t, buildDigit0)
}

// meshKey names one tessellation: a body at a chord tolerance.
type meshKey struct {
	body *decad.Body
	tol  float64
}

// meshEntry holds a cached tessellation and its optional gap mesh.
type meshEntry struct {
	meshOnce sync.Once
	mesh     *decad.Mesh
	meshErr  error
	gapOnce  sync.Once
	mg       meshgap.Mesh
	gapErr   error
}

var meshCache sync.Map // meshKey -> *meshEntry

// meshEntryOf returns the cached entry for b and tol after verifying its tessellation.
func meshEntryOf(t *testing.T, b *decad.Body, tol float64) *meshEntry {
	t.Helper()
	v, _ := meshCache.LoadOrStore(meshKey{b, tol}, &meshEntry{})
	e := v.(*meshEntry)
	e.meshOnce.Do(func() {
		e.mesh, e.meshErr = b.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
	})
	require.NoError(t, e.meshErr)
	return e
}

// rawMeshOf returns the verified tessellation without constructing a gap mesh.
func rawMeshOf(t *testing.T, b *decad.Body, tol float64) *decad.Mesh {
	t.Helper()
	return meshEntryOf(t, b, tol).mesh
}

// meshOf returns the mesh of b at tol and its gap mesh, each computed once per body and tolerance and shared. Callers
// read them only: a test that moves a mesh copies its vertices first.
func meshOf(t *testing.T, b *decad.Body, tol float64) (*decad.Mesh, meshgap.Mesh) {
	t.Helper()
	e := meshEntryOf(t, b, tol)
	e.gapOnce.Do(func() {
		e.mg, e.gapErr = meshgap.New(e.mesh.Vertices(), e.mesh.Triangles(), e.mesh.Bound().Base())
	})
	require.NoError(t, e.gapErr)
	return e.mesh, e.mg
}

// shaftMate returns the other body of the pair a, b when one of them is the shaft.
func shaftMate(a, b string) (string, bool) {
	switch {
	case a == "shaft":
		return b, true
	case b == "shaft":
		return a, true
	}
	return "", false
}

// boxesTouch reports whether two boxes lie within margin of each other.
func boxesTouch(a, b decad.Box, margin float64) bool {
	return a.Min.X-margin < b.Max.X && b.Min.X-margin < a.Max.X && a.Min.Y-margin < b.Max.Y &&
		b.Min.Y-margin < a.Max.Y && a.Min.Z-margin < b.Max.Z && b.Min.Z-margin < a.Max.Z
}

// along returns the least and greatest projection onto dir of the vertices of mesh.
func along(mesh *decad.Mesh, dir r3.Vec) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range mesh.Vertices() {
		d := v.Dot(dir)
		lo, hi = math.Min(lo, d), math.Max(hi, d)
	}
	return lo, hi
}

// radius returns the mean distance of the points of a loop from the origin.
func radius(loop []r3.Vec) float64 {
	sum := 0.0
	for _, p := range loop {
		sum += p.Len()
	}
	return sum / float64(len(loop))
}

// posed returns a meshgap mesh of mesh turned by deg about the line through point along dir and moved by dz.
func posed(t *testing.T, mesh *decad.Mesh, point, dir r3.Vec, deg, dz float64) meshgap.Mesh {
	t.Helper()
	turn, err := r3.RotationAround(point, dir, units.Degrees(deg))
	require.NoError(t, err)
	src := mesh.Vertices()
	out := make([]r3.Vec, len(src))
	for i, v := range src {
		out[i] = turn.Apply(v)
		out[i].Z += dz
	}
	mg, err := meshgap.New(out, mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// segmentAxis returns a flag's axis point and direction: through the segment's centre at the rotor height, along X for a, g, d
// and along Y for b, c, e, f.
func segmentAxis(s sevensegment.Segment) (r3.Vec, r3.Vec) {
	switch s {
	case sevensegment.SegA:
		return r3.Vec{Y: 33, Z: sevensegment.RotorAxisZ}, r3.Vec{X: 1}
	case sevensegment.SegG:
		return r3.Vec{Z: sevensegment.RotorAxisZ}, r3.Vec{X: 1}
	case sevensegment.SegD:
		return r3.Vec{Y: -33, Z: sevensegment.RotorAxisZ}, r3.Vec{X: 1}
	case sevensegment.SegB:
		return r3.Vec{X: 16.5, Y: 16.5, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
	case sevensegment.SegC:
		return r3.Vec{X: 16.5, Y: -16.5, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
	case sevensegment.SegE:
		return r3.Vec{X: -16.5, Y: -16.5, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
	}
	return r3.Vec{X: -16.5, Y: 16.5, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// pointIn reports whether p lies inside the closed mesh: a ray along +X crosses it an odd number of times.
func pointIn(p r3.Vec, mesh *decad.Mesh) bool {
	vs, ts := mesh.Vertices(), mesh.Triangles()
	crossings := 0
	for _, tri := range ts {
		if crossesRay(p, vs[tri[0]], vs[tri[1]], vs[tri[2]]) {
			crossings++
		}
	}
	return crossings%2 == 1
}

// crossesRay reports whether pointIn's ray from p crosses the triangle a, b, c. It skips a triangle lying wholly
// more than pointInMargin behind p along X, or beside p in Y or Z.
func crossesRay(p, a, b, c r3.Vec) bool {
	const m = pointInMargin
	if (a.X < p.X-m && b.X < p.X-m && c.X < p.X-m) ||
		(a.Y > p.Y+m && b.Y > p.Y+m && c.Y > p.Y+m) || (a.Y < p.Y-m && b.Y < p.Y-m && c.Y < p.Y-m) ||
		(a.Z > p.Z+m && b.Z > p.Z+m && c.Z > p.Z+m) || (a.Z < p.Z-m && b.Z < p.Z-m && c.Z < p.Z-m) {
		return false
	}
	// Moller-Trumbore with direction +X.
	e1, e2 := b.Sub(a), c.Sub(a)
	dir := r3.Vec{X: 1, Y: 1.37e-4, Z: 2.71e-4} // off the mesh's edges and axes
	h := dir.Cross(e2)
	det := e1.Dot(h)
	if math.Abs(det) < 1e-12 {
		return false
	}
	s := p.Sub(a)
	u := s.Dot(h) / det
	if u < 0 || u > 1 {
		return false
	}
	q := s.Cross(e1)
	v := dir.Dot(q) / det
	if v < 0 || u+v > 1 {
		return false
	}
	return e2.Dot(q)/det > 0
}
