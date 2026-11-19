package sevensegment_test

import (
	"context"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// build is one assembly build shared by the tests that read it. A test starts it before t.Parallel, so the builds of
// every test that will run begin at once and overlap (each in a document of its own: decad documents are independent,
// and the bodies they hand out are immutable and safe to read from anywhere); built waits for it.
type build[T any] struct {
	once sync.Once
	done chan struct{}
	val  T
	err  error
}

// start begins the build in the background, the first time it is called.
func (b *build[T]) start(f func(context.Context) (T, error)) {
	b.once.Do(func() {
		b.done = make(chan struct{})
		go func() {
			defer close(b.done)
			// The build outlives the test that started it, so it runs under no test's context.
			b.val, b.err = f(context.Background())
		}()
	})
}

// built starts the build if no test has, waits for it and returns it.
func (b *build[T]) built(t *testing.T, f func(context.Context) (T, error)) T {
	t.Helper()
	b.start(f)
	<-b.done
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
	doc := decad.New()
	m, err := sevensegment.Build(ctx, doc, sevensegment.DefaultParams())
	return defaultModel{doc, m}, err
}

// startDefault begins the default build for a test about to call t.Parallel.
func startDefault() { defaultBuild.start(buildDefault) }

// builtDefault returns the default display, built once and shared: the tests only read it.
func builtDefault(t *testing.T) (*decad.Document, *sevensegment.Model) {
	t.Helper()
	d := defaultBuild.built(t, buildDefault)
	return d.doc, d.model
}

// printed returns the printed bodies of m by name.
func printed(m *sevensegment.Model) map[string]*decad.Body {
	out := map[string]*decad.Body{
		"front wall": m.FrontWall, "rear wall": m.RearWall, "detent plate": m.DetentPlate, "guide plate": m.GuidePlate,
		"cradle plate": m.CradlePlate, "top plate": m.TopPlate, "shaft": m.Shaft, "ratchet wheel": m.RatchetWheel, "collar ring": m.CollarRing,
		"knob": m.Knob,
	}
	for _, s := range sevensegment.Segments() {
		out["cam "+s.String()] = m.Cams[s]
		out["slider "+s.String()] = m.Sliders[s]
		out["yellow face "+s.String()] = m.Flags[s].Yellow
		out["black body "+s.String()] = m.Flags[s].Black
	}
	return out
}

var digit0Build build[*sevensegment.Parts]

func buildDigit0(ctx context.Context) (*sevensegment.Parts, error) {
	return sevensegment.BuildParts(ctx, decad.New(), sevensegment.Params{Digit: 0})
}

// startDigit0 begins the digit 0 build for a test about to call t.Parallel.
func startDigit0() { digit0Build.start(buildDigit0) }

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

// meshEntry is one cached tessellation and its gap mesh.
type meshEntry struct {
	once sync.Once
	mesh *decad.Mesh
	mg   meshgap.Mesh
	err  error
}

var meshCache sync.Map // meshKey -> *meshEntry

// meshOf returns the mesh of b at tol and its gap mesh, each computed once per body and tolerance and shared. Callers
// read them only: a test that moves a mesh copies its vertices first.
func meshOf(t *testing.T, b *decad.Body, tol float64) (*decad.Mesh, meshgap.Mesh) {
	t.Helper()
	v, _ := meshCache.LoadOrStore(meshKey{b, tol}, &meshEntry{})
	e := v.(*meshEntry)
	e.once.Do(func() {
		e.mesh, e.err = b.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if e.err != nil {
			return
		}
		e.mg, e.err = meshgap.New(e.mesh.Vertices(), e.mesh.Triangles(), e.mesh.Bound().Base())
	})
	require.NoError(t, e.err)
	return e.mesh, e.mg
}

func TestSevenSegmentDefault(t *testing.T) {
	startDefault()
	t.Parallel()
	doc, m := builtDefault(t)
	require.Len(t, doc.Bodies(), 60)
	for _, b := range doc.Bodies() {
		require.True(t, b.IsSolid())
	}
	require.NotEmpty(t, m.Sketches)
	for i, s := range m.Sketches {
		report := s.Verify(t.Context())
		require.Equal(t, 0, report.DOF, "sketch %d", i)
		require.Empty(t, report.Redundant, "sketch %d", i)
		require.Empty(t, report.Conflicts, "sketch %d", i)
	}
	// No two printed bodies overlap: every pair is separated, the stated fits keeping at least their 0.05 mm.
	meshes := map[string]meshgap.Mesh{}
	boxes := map[string]decad.Box{}
	for name, b := range printed(m) {
		_, meshes[name] = meshOf(t, b, 0.05)
		box, err := b.Bounds()
		require.NoError(t, err)
		boxes[name] = box
	}
	names := make([]string, 0, len(meshes))
	for n := range meshes {
		names = append(names, n)
	}
	pressed := map[string]struct{}{"ratchet wheel": {}, "collar ring": {}, "knob": {}}
	for _, seg := range sevensegment.Segments() {
		pressed["cam "+seg.String()] = struct{}{}
	}
	shaftPairs := 0
	for i, a := range names {
		for _, b := range names[i+1:] {
			if !boxesTouch(boxes[a], boxes[b], 0.05) {
				continue
			}
			if strings.HasPrefix(a, "yellow face ") && b == "black body "+strings.TrimPrefix(a, "yellow face ") ||
				strings.HasPrefix(b, "yellow face ") && a == "black body "+strings.TrimPrefix(b, "yellow face ") {
				continue // the two bodies of one flag share their boundary faces
			}
			// Every pair is checked, the shaft's too. The parts pressed on the shaft keep its 0.05 mm press fit; their
			// meshes are read at a finer chord (0.005 mm) so that the proven bound is tighter than the fit. No other pair
			// overlaps by design, so there is no allow-list beyond the two bodies of one flag above.
			ma, mb, clearance := meshes[a], meshes[b], 0.0
			if other, ok := shaftMate(a, b); ok {
				if _, ok := pressed[other]; ok {
					_, ma = meshOf(t, printed(m)[a], 0.005)
					_, mb = meshOf(t, printed(m)[b], 0.005)
					clearance = 0.03 // the 0.05 mm press fit less the meshes' 0.02 mm allowance
					shaftPairs++
				}
			}
			sep, err := ma.ExceedsClearance(mb, clearance)
			require.NoError(t, err)
			require.True(t, sep, "%s and %s overlap or touch, or keep less than %.2f mm", a, b, clearance)
		}
	}
	require.Equal(t, len(pressed), shaftPairs, "every part pressed on the shaft is checked against it")
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

func TestSevenSegmentContainers(t *testing.T) {
	startDefault()
	t.Parallel()
	doc, m := builtDefault(t)
	require.NoError(t, doc.VerifyContainers().Err())
	want := map[string]struct{ parent, kind string }{
		"front wall":    {sevensegment.FrameContainerName, sevensegment.WallsContainerName},
		"rear wall":     {sevensegment.FrameContainerName, sevensegment.WallsContainerName},
		"guide plate":   {sevensegment.FrameContainerName, sevensegment.PlatesContainerName},
		"cradle plate":  {sevensegment.FrameContainerName, sevensegment.PlatesContainerName},
		"top plate":     {sevensegment.FrameContainerName, sevensegment.PlatesContainerName},
		"detent plate":  {sevensegment.FrameContainerName, sevensegment.DetentPlateContainerName},
		"shaft":         {sevensegment.CamshaftContainerName, sevensegment.ShaftContainerName},
		"ratchet wheel": {sevensegment.CamshaftContainerName, sevensegment.RatchetWheelContainerName},
		"collar ring":   {sevensegment.CamshaftContainerName, sevensegment.CollarRingContainerName},
		"knob":          {sevensegment.CamshaftContainerName, sevensegment.KnobContainerName},
	}
	for _, s := range sevensegment.Segments() {
		want["cam "+s.String()] = struct{ parent, kind string }{sevensegment.CamshaftContainerName,
			sevensegment.CamsContainerName}
		want["slider "+s.String()] = struct{ parent, kind string }{"", sevensegment.SlidersContainerName}
	}
	for name, b := range printed(m) {
		if strings.HasPrefix(name, "yellow face") || strings.HasPrefix(name, "black body") {
			continue // checked below
		}
		c, err := doc.ContainerOf(b)
		require.NoError(t, err, name)
		require.Equal(t, want[name].kind, c.Name(), name)
		if want[name].parent == "" {
			require.Equal(t, doc.Root(), c.Parent(), name)
		} else {
			require.Equal(t, want[name].parent, c.Parent().Name(), name)
		}
	}
	// Each flag is a unit of the Flags container holding its Yellow Face and its Black Body.
	for _, s := range sevensegment.Segments() {
		for kind, b := range map[string]*decad.Body{sevensegment.YellowFaceContainerName: m.Flags[s].Yellow,
			sevensegment.BlackBodyContainerName: m.Flags[s].Black} {
			c, err := doc.ContainerOf(b)
			require.NoError(t, err)
			require.Equal(t, kind, c.Name())
			require.True(t, strings.HasPrefix(c.Parent().Name(), sevensegment.FlagUnitName), "segment %s", s)
			require.Equal(t, sevensegment.FlagsContainerName, c.Parent().Parent().Name())
		}
	}
	for i, b := range m.Bearings {
		c, err := doc.ContainerOf(b.Body)
		require.NoError(t, err, "bearing %d", i)
		require.Equal(t, sevensegment.BearingsContainerName, c.Name())
	}
	for i := range m.Bolts {
		c, err := doc.ContainerOf(m.Bolts[i].Body)
		require.NoError(t, err)
		require.Equal(t, sevensegment.BoltsContainerName, c.Name())
		c, err = doc.ContainerOf(m.Nuts[i].Body)
		require.NoError(t, err)
		require.Equal(t, sevensegment.NutsContainerName, c.Name())
	}
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

func TestSevenSegmentFasteners(t *testing.T) {
	startDefault()
	t.Parallel()
	_, m := builtDefault(t)
	const tol = 0.02
	parts := printed(m)
	pmeshes := map[string]*decad.Mesh{}
	grids := map[string]*crossingGrid{}
	for name, b := range parts {
		pmeshes[name], _ = meshOf(t, b, 0.05)
		grids[name] = newCrossingGrid(pmeshes[name])
	}
	for i, site := range m.Sites {
		base := site.Head.Dot(site.Dir)
		bolt, _ := meshOf(t, m.Bolts[i].Body, 0.01)
		lo, hi := along(bolt, site.Dir)
		require.InDelta(t, base, lo, tol, "bolt %d head top is flush", i+1)
		require.InDelta(t, base+site.Tip(), hi, tol, "bolt %d tip", i+1)
		// The tip stays inside a face: it does not pass the farthest face of the parts around the bolt's axis. The faces are
		// read from the part meshes, not from the site the bolt was built from.
		far := math.Inf(-1)
		for _, mesh := range pmeshes {
			for _, v := range mesh.Vertices() {
				d := v.Sub(site.Head)
				along := d.Dot(site.Dir)
				if along < 0 || d.Sub(site.Dir.Scale(along)).Len() > 2.5 {
					continue
				}
				far = math.Max(far, v.Dot(site.Dir))
			}
		}
		require.LessOrEqual(t, hi, far+tol, "bolt %d tip stands outside the parts' far face (%.2f)", i+1, far)
		nut, _ := meshOf(t, m.Nuts[i].Body, 0.01)
		nlo, nhi := along(nut, site.Dir)
		require.GreaterOrEqual(t, nlo, base+site.Cavity-tol, "nut %d lies inside its cavity", i+1)
		require.LessOrEqual(t, nhi, base+site.CavityFar()+tol, "nut %d lies inside its cavity", i+1)
		centre := r3.Vec{}
		for _, v := range nut.Vertices() {
			centre = centre.Add(v)
		}
		centre = centre.Scale(1 / float64(len(nut.Vertices())))
		// No bolt point lies inside a part it passes. A head vertex is probed 0.1 mm toward the head's middle, so that the
		// vertices on the flush top and on the underside (which rests on the pocket floor) are read from inside the head: a
		// head that reaches into the part beside its pocket puts those probes in the material. The shank is read at its
		// vertices.
		for name, grid := range grids {
			for _, v := range bolt.Vertices() {
				p, depth := v, v.Sub(site.Head).Dot(site.Dir)
				switch {
				case depth <= m2fastener.HeadPocketDepth+0.05 && depth < m2fastener.HeadPocketDepth/2:
					p = v.Add(site.Dir.Scale(0.1))
				case depth <= m2fastener.HeadPocketDepth+0.05:
					p = v.Sub(site.Dir.Scale(0.1))
				}
				if grid.pointIn(p) {
					require.Fail(t, "bolt overlaps a part", "bolt %d vertex %v (probe %v) inside %s", i+1, v, p, name)
				}
			}
			// The nut lies in its cavity: no vertex, probed 0.05 mm toward the nut's middle, is in a part, and the nut keeps
			// at least 0.05 mm (the coarse part mesh's own bound) from every part near it.
			for _, v := range nut.Vertices() {
				toward := centre.Sub(v)
				if grid.pointIn(v.Add(toward.Scale(0.05 / toward.Len()))) {
					require.Fail(t, "nut overlaps a part", "nut %d vertex %v inside %s", i+1, v, name)
				}
			}
		}
		_, nutGap := meshOf(t, m.Nuts[i].Body, 0.01)
		nutBox, err := m.Nuts[i].Body.Bounds()
		require.NoError(t, err)
		for name, b := range parts {
			box, err := b.Bounds()
			require.NoError(t, err)
			if !boxesTouch(box, nutBox, 0.5) {
				continue
			}
			_, partGap := meshOf(t, b, 0.05)
			ok, err := nutGap.ExceedsClearance(partGap, 0.05)
			require.NoError(t, err)
			require.True(t, ok, "nut %d keeps 0.05 mm from %s", i+1, name)
		}
	}
}

// TestSevenSegmentBearings checks the two 605ZZ bearings against the parts around them: the wall seat grips the outer
// race with 0.05 mm of radial clearance (P2), the shaft's journal equals the bore (N1, a nominal exception: no clearance
// and no overlap), and no other printed part touches a bearing.
func TestSevenSegmentBearings(t *testing.T) {
	startDefault()
	t.Parallel()
	_, m := builtDefault(t)
	parts := printed(m)
	for i, walls := range []struct {
		name string
		wall *decad.Body
	}{{"front wall", m.FrontWall}, {"rear wall", m.RearWall}} {
		bearing := m.Bearings[i].Body
		box, err := bearing.Bounds()
		require.NoError(t, err)
		mid := (box.Min.Y + box.Max.Y) / 2
		race := sevensegment.SectionAt(t, bearing, mid)
		require.Len(t, race, 2, "%s bearing section: outer ring and bore", walls.name)
		outer, bore := race[:1], race[1:]
		if radius(race[0]) < radius(race[1]) {
			outer, bore = race[1:], race[:1]
		}
		seat := sevensegment.LoopsGap(outer, sevensegment.SectionAt(t, walls.wall, mid))
		require.InDelta(t, sevensegment.PressFit, seat, 0.02, "%s seat to the outer race (P2)", walls.name)
		journal := sevensegment.SectionAt(t, m.Shaft, mid)
		require.InDelta(t, radius(bore[0]), radius(journal[0]), 0.01, "%s: journal and bore (N1)", walls.name)
		// No overlap: the journal's radius never exceeds the bore's.
		for _, p := range journal[0] {
			require.LessOrEqual(t, p.Len(), radius(bore[0])+0.005, "%s: the journal stays inside the bore", walls.name)
		}
		_, bearingGap := meshOf(t, bearing, 0.05)
		for name, b := range parts {
			if b == walls.wall || b == m.Shaft {
				continue
			}
			pbox, err := b.Bounds()
			require.NoError(t, err)
			if !boxesTouch(pbox, box, 0.05) {
				continue
			}
			_, partGap := meshOf(t, b, 0.05)
			ok, err := bearingGap.ExceedsClearance(partGap, 0)
			require.NoError(t, err)
			require.True(t, ok, "%s bearing and %s touch", walls.name, name)
		}
	}
}

// radius returns the mean distance of the points of a loop from the origin.
func radius(loop []r3.Vec) float64 {
	sum := 0.0
	for _, p := range loop {
		sum += p.Len()
	}
	return sum / float64(len(loop))
}

// TestSevenSegmentPlacement moves the whole display three ways: by 100 mm along X, by (10, 20, 30), and by a quarter turn
// about Z. Each part's bounds and each bolt site must follow the placement. Decad builds all three (an earlier note here
// said it refused slider b for most translations; these and several others build).
func TestSevenSegmentPlacement(t *testing.T) {
	type placement struct {
		name         string
		origin, u, v r3.Vec
	}
	cases := []placement{
		{"100 along X", r3.Vec{X: 100}, r3.Vec{X: 1}, r3.Vec{Y: 1}},
		{"offset 10, 20, 30", r3.Vec{X: 10, Y: 20, Z: 30}, r3.Vec{X: 1}, r3.Vec{Y: 1}},
		{"quarter turn about Z", r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{X: -1}},
	}
	builds := make([]build[*sevensegment.Parts], len(cases))
	frames := make([]r3.Frame, len(cases))
	for i, c := range cases {
		var err error
		frames[i], err = r3.NewFrame(c.origin, c.u, c.v)
		require.NoError(t, err)
		p := sevensegment.DefaultParams()
		p.Placement = &frames[i]
		builds[i].start(func(ctx context.Context) (*sevensegment.Parts, error) {
			return sevensegment.BuildParts(ctx, decad.New(), p)
		})
	}
	startDefault()
	t.Parallel()
	_, base := builtDefault(t)
	plain := printed(base)
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			frame := frames[i]
			p := sevensegment.DefaultParams()
			p.Placement = &frame
			parts := builds[i].built(t, func(ctx context.Context) (*sevensegment.Parts, error) {
				return sevensegment.BuildParts(ctx, decad.New(), p)
			})
			place, err := r3.FromFrame(frame)
			require.NoError(t, err)
			for name, b := range printed(&sevensegment.Model{Parts: *parts}) {
				got, err := b.Bounds()
				require.NoError(t, err)
				want, err := plain[name].Bounds()
				require.NoError(t, err)
				// The placed box is the box of the plain one's corners, which a quarter turn keeps axis-aligned.
				lo, hi := r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
				for _, x := range []float64{want.Min.X, want.Max.X} {
					for _, y := range []float64{want.Min.Y, want.Max.Y} {
						for _, z := range []float64{want.Min.Z, want.Max.Z} {
							q := place.Apply(r3.Vec{X: x, Y: y, Z: z})
							lo = r3.Vec{X: math.Min(lo.X, q.X), Y: math.Min(lo.Y, q.Y), Z: math.Min(lo.Z, q.Z)}
							hi = r3.Vec{X: math.Max(hi.X, q.X), Y: math.Max(hi.Y, q.Y), Z: math.Max(hi.Z, q.Z)}
						}
					}
				}
				for axis, pair := range map[string][2]float64{"x": {lo.X, got.Min.X}, "y": {lo.Y, got.Min.Y}, "z": {lo.Z, got.Min.Z},
					"max x": {hi.X, got.Max.X}, "max y": {hi.Y, got.Max.Y}, "max z": {hi.Z, got.Max.Z}} {
					require.InDelta(t, pair[0], pair[1], 0.02, "%s: %s", name, axis)
				}
			}
			for k, s := range parts.Sites {
				head, dir := place.Apply(base.Sites[k].Head), place.ApplyDir(base.Sites[k].Dir)
				require.InDelta(t, head.X, s.Head.X, 1e-9, "site %d head x", k)
				require.InDelta(t, head.Y, s.Head.Y, 1e-9, "site %d head y", k)
				require.InDelta(t, head.Z, s.Head.Z, 1e-9, "site %d head z", k)
				require.InDelta(t, dir.X, s.Dir.X, 1e-9, "site %d dir x", k)
				require.InDelta(t, dir.Y, s.Dir.Y, 1e-9, "site %d dir y", k)
				require.InDelta(t, dir.Z, s.Dir.Z, 1e-9, "site %d dir z", k)
			}
		})
	}
	p := sevensegment.DefaultParams()
	p.Placement = &r3.Frame{}
	_, err := sevensegment.BuildParts(t.Context(), decad.New(), p)
	require.EqualError(t, err, "sevensegment: invalid placement frame")
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

// TestSevenSegmentAllDigits poses the digit-0 build for each digit: the cams turned, the sliders lifted and the flags
// turned. It checks every cam against its slider (the pin keeps the Regular minimum from the groove), every slider
// against the plates, the other sliders and its own flag, and every flag against the plates and the other flags.
func TestSevenSegmentAllDigits(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	var cams, sliders, yellows, blacks [7]*decad.Mesh
	var fixed [3]meshgap.Mesh
	for _, s := range sevensegment.Segments() {
		cams[s], _ = meshOf(t, parts.Cams[s], 0.05)
		sliders[s], _ = meshOf(t, parts.Sliders[s], 0.05)
		yellows[s], _ = meshOf(t, parts.Flags[s].Yellow, 0.05)
		blacks[s], _ = meshOf(t, parts.Flags[s].Black, 0.05)
	}
	_, fixed[0] = meshOf(t, parts.TopPlate, 0.05)
	_, fixed[1] = meshOf(t, parts.GuidePlate, 0.05)
	_, fixed[2] = meshOf(t, parts.CradlePlate, 0.05)
	shaft := r3.Vec{Z: sevensegment.ShaftZ}
	for k := range 10 {
		var lifted, black, yellow [7]meshgap.Mesh
		for _, s := range sevensegment.Segments() {
			was, _ := sevensegment.Lit(0, s)
			now, _ := sevensegment.Lit(k, s)
			dz := sevensegment.Travel * (b2f(now) - b2f(was))
			lifted[s] = posed(t, sliders[s], shaft, r3.Vec{Y: 1}, 0, dz)
			point, axis := segmentAxis(s)
			dphi := dm.FlagAngle(s, 36*float64(k)) - dm.FlagAngle(s, 0)
			black[s] = posed(t, blacks[s], point, axis, dphi, 0)
			yellow[s] = posed(t, yellows[s], point, axis, dphi, 0)
			cam := posed(t, cams[s], shaft, r3.Vec{Y: 1}, 36*float64(k), 0)
			ok, err := cam.ExceedsClearance(lifted[s], 0.35)
			require.NoError(t, err)
			require.True(t, ok, "digit %d: pin of segment %s closer than 0.35 to its cam", k, s)
			for _, plate := range fixed {
				ok, err := plate.ExceedsClearance(lifted[s], 0)
				require.NoError(t, err)
				require.True(t, ok, "digit %d: slider %s touches a plate", k, s)
				for _, flag := range []meshgap.Mesh{black[s], yellow[s]} {
					ok, err := plate.ExceedsClearance(flag, 0)
					require.NoError(t, err)
					require.True(t, ok, "digit %d: flag %s touches a plate", k, s)
				}
			}
			// The tongue and pin keep their gaps to the notch and pocket walls (T7 is 0.20 on sections, T8 more; the mesh check is
			// coarser, 0.15).
			ok, err = black[s].ExceedsClearance(lifted[s], 0.15)
			require.NoError(t, err)
			require.True(t, ok, "digit %d: slider %s closer than 0.15 to its flag", k, s)
		}
		for i := range lifted {
			for j := i + 1; j < len(lifted); j++ {
				for _, pair := range [][2]meshgap.Mesh{{lifted[i], lifted[j]}, {black[i], black[j]}, {black[i], lifted[j]},
					{lifted[i], black[j]}} {
					ok, err := pair[0].ExceedsClearance(pair[1], 0)
					require.NoError(t, err)
					require.True(t, ok, "digit %d: segments %d and %d touch", k, i, j)
				}
			}
		}
	}
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
