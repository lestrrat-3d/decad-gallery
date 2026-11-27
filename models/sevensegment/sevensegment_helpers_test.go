package sevensegment_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
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

// placementCase is one placement the placement test moves the display by.
type placementCase struct {
	name         string
	key          string // the pose name in SEVENSEGMENT_POSES
	origin, u, v r3.Vec
}

// placementCases are the placements TestSevenSegmentPlacement checks, in the order of the fixture's placed poses.
var placementCases = []placementCase{
	{"100 along X", "x", r3.Vec{X: 100}, r3.Vec{X: 1}, r3.Vec{Y: 1}},
	{"offset 10, 20, 30", "offset", r3.Vec{X: 10, Y: 20, Z: 30}, r3.Vec{X: 1}, r3.Vec{Y: 1}},
	{"quarter turn about Z", "quarter", r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{X: -1}},
}

// frame returns the placement's frame.
func (c placementCase) frame() (r3.Frame, error) { return r3.NewFrame(c.origin, c.u, c.v) }

// posesEnv names the environment variable that selects the poses of the shared posed build. Unset, the build makes
// every pose, so one test run on its own works. A CI shard sets it to the poses its tests read: a comma-separated list
// of pose names, or "none".
const posesEnv = "SEVENSEGMENT_POSES"

// digit0Pose is the name of the digit 0 pose. The placed poses are named by placementCase.key.
const digit0Pose = "digit0"

// allPoseNames lists every pose of the shared build, in build order.
func allPoseNames() []string {
	names := []string{digit0Pose}
	for _, c := range placementCases {
		names = append(names, c.key)
	}
	return names
}

// selectPoses returns the poses a posesEnv value names, in build order. It reports unset as every pose. It rejects an
// empty value, an unknown name and a repeated name, so a wrong value fails the build instead of changing it.
func selectPoses(value string, set bool) ([]string, error) {
	all := allPoseNames()
	if !set {
		return all, nil
	}
	if value == "none" {
		return nil, nil
	}
	if value == "" {
		return nil, fmt.Errorf("%s is set but empty; name poses (%s) or use none", posesEnv, strings.Join(all, ", "))
	}
	seen := map[string]struct{}{}
	for _, name := range strings.Split(value, ",") {
		if !slices.Contains(all, name) {
			return nil, fmt.Errorf("%s names unknown pose %q; poses are %s", posesEnv, name, strings.Join(all, ", "))
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("%s names pose %q twice", posesEnv, name)
		}
		seen[name] = struct{}{}
	}
	var out []string
	for _, name := range all {
		if _, ok := seen[name]; ok {
			out = append(out, name)
		}
	}
	return out, nil
}

// posedModel is the default display, its document, and the printed parts the other tests pose: the digit 0 parts and
// the parts of the default digit at each placement of placementCases. They all come from one BuildPosed call, so the
// bodies of the poses are copies of the default display's unposed bodies. Only the selected poses are built; built
// lists them.
type posedModel struct {
	doc    *decad.Document
	model  *sevensegment.Model
	digit0 *sevensegment.Parts
	placed map[string]*sevensegment.Parts // by placementCase.key
	built  []string
}

var posedBuild build[posedModel]

func buildPosed(ctx context.Context) (posedModel, error) {
	value, set := os.LookupEnv(posesEnv)
	selected, err := selectPoses(value, set)
	if err != nil {
		return posedModel{}, err
	}
	var poses []sevensegment.Pose
	for _, name := range selected {
		if name == digit0Pose {
			poses = append(poses, sevensegment.Pose{Digit: 0})
			continue
		}
		for _, c := range placementCases {
			if c.key != name {
				continue
			}
			frame, err := c.frame()
			if err != nil {
				return posedModel{}, err
			}
			poses = append(poses, sevensegment.Pose{Digit: sevensegment.DefaultParams().Digit, Placement: &frame})
		}
	}
	doc := sevensegment.NewDocument()
	m, extras, err := sevensegment.BuildPosed(ctx, doc, sevensegment.DefaultParams(), poses...)
	if err != nil {
		return posedModel{}, err
	}
	// The poses' own bodies sit in no container. Take them out of the document so that it holds the model's 78 bodies,
	// as a plain Build leaves it. A removed body still reads its mesh, volume and bounds, which is all the tests that
	// use these parts do.
	// printed lists every printed body but the spring.
	bodies := func(p *sevensegment.Parts) []*decad.Body {
		out := []*decad.Body{p.Spring}
		for _, b := range printed(&sevensegment.Model{Parts: *p}) {
			out = append(out, b)
		}
		return out
	}
	model := map[*decad.Body]struct{}{}
	for _, b := range bodies(&m.Parts) {
		model[b] = struct{}{}
	}
	own := map[*decad.Body]struct{}{}
	for _, parts := range extras {
		for _, b := range bodies(parts) {
			if _, ok := model[b]; !ok {
				own[b] = struct{}{}
			}
		}
	}
	for b := range own {
		if err := decad.Remove(b).Do(ctx); err != nil {
			return posedModel{}, err
		}
	}
	out := posedModel{doc: doc, model: m, placed: map[string]*sevensegment.Parts{}, built: selected}
	for i, name := range selected {
		if name == digit0Pose {
			out.digit0 = extras[i]
			continue
		}
		out.placed[name] = extras[i]
	}
	return out, nil
}

// poseReporter is the part of *testing.T that requirePose uses.
type poseReporter interface {
	require.TestingT
	Helper()
}

// requirePose fails the test, naming the pose, when the shared build did not make it.
func requirePose(t poseReporter, d posedModel, name string) {
	t.Helper()
	if !slices.Contains(d.built, name) {
		require.FailNowf(t, "pose not built", "pose %q was not built: %s selects %v; add %q to the shard's poses in "+
			"SEVENSEGMENT_POSES of scripts/ci/run_tests.py", name, posesEnv, d.built, name)
	}
}

// builtDefault returns the default display, built once and shared: the tests only read it.
func builtDefault(t *testing.T) (*decad.Document, *sevensegment.Model) {
	t.Helper()
	d := posedBuild.built(t, buildPosed)
	return d.doc, d.model
}

// builtPlaced returns the printed parts of the default digit placed by placementCases[i], built once and shared.
func builtPlaced(t *testing.T, i int) *sevensegment.Parts {
	t.Helper()
	d := posedBuild.built(t, buildPosed)
	requirePose(t, d, placementCases[i].key)
	return d.placed[placementCases[i].key]
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
		if m.GroovePins[s] != nil {
			out["groove pin "+s.String()] = m.GroovePins[s]
		}
		out["yellow face "+s.String()] = m.Flags[s].Yellow
		out["black body "+s.String()] = m.Flags[s].Black
		out["pivot block "+s.String()+" -u"] = m.PivotBlocks[s][0]
		out["pivot block "+s.String()+" +u"] = m.PivotBlocks[s][1]
	}
	return out
}

// builtDigit0 returns the printed parts at digit 0, built once and shared: the tests read them and move copies of their
// meshes, never the bodies. The bodies the digit does not move are those of the default display.
func builtDigit0(t *testing.T) *sevensegment.Parts {
	t.Helper()
	d := posedBuild.built(t, buildPosed)
	requirePose(t, d, digit0Pose)
	return d.digit0
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
		return r3.Vec{Y: sevensegment.SegmentSpan, Z: sevensegment.RotorAxisZ}, r3.Vec{X: 1}
	case sevensegment.SegG:
		return r3.Vec{Z: sevensegment.RotorAxisZ}, r3.Vec{X: 1}
	case sevensegment.SegD:
		return r3.Vec{Y: -sevensegment.SegmentSpan, Z: sevensegment.RotorAxisZ}, r3.Vec{X: 1}
	case sevensegment.SegB:
		return r3.Vec{X: sevensegment.SegmentSpan / 2, Y: sevensegment.SegmentSpan / 2, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
	case sevensegment.SegC:
		return r3.Vec{X: sevensegment.SegmentSpan / 2, Y: -sevensegment.SegmentSpan / 2, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
	case sevensegment.SegE:
		return r3.Vec{X: -sevensegment.SegmentSpan / 2, Y: -sevensegment.SegmentSpan / 2, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
	}
	return r3.Vec{X: -sevensegment.SegmentSpan / 2, Y: sevensegment.SegmentSpan / 2, Z: sevensegment.RotorAxisZ}, r3.Vec{Y: 1}
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
