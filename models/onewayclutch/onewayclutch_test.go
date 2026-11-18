package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

type built struct {
	m   *onewayclutch.Model
	doc *decad.Document
}

type fixtureKey struct {
	shaft         string
	freewheel     onewayclutch.Direction
	boreClearance float64
	hasPlacement  bool
	placement     r3.Frame
}

func keyForParams(p onewayclutch.Params) fixtureKey {
	key := fixtureKey{shaft: p.Shaft, freewheel: p.Freewheel, boreClearance: p.BoreClearance}
	if p.Placement != nil {
		key.hasPlacement = true
		key.placement = *p.Placement
	}
	return key
}

var cache = map[fixtureKey]built{}

// buildCase builds a case on first use; the tests only read its bodies.
func buildCase(t *testing.T, p onewayclutch.Params) (*onewayclutch.Model, *decad.Document) {
	t.Helper()
	key := keyForParams(p)
	if b, ok := cache[key]; ok {
		return b.m, b.doc
	}
	doc := onewayclutch.NewDocument()
	m, err := onewayclutch.Build(t.Context(), doc, p)
	require.NoError(t, err)
	cache[key] = built{m: m, doc: doc}
	return m, doc
}

func buildDefault(t *testing.T) (*onewayclutch.Model, *decad.Document) {
	t.Helper()
	return buildCase(t, onewayclutch.DefaultParams())
}

func TestFixtureKey(t *testing.T) {
	p := onewayclutch.DefaultParams()
	base := keyForParams(p)
	shaft := p
	shaft.Shaft = "608ZZ"
	direction := p
	direction.Freewheel = onewayclutch.CounterClockwise
	clearance := p
	clearance.BoreClearance = 0.35
	zero := r3.Frame{}
	placed := p
	placed.Placement = &zero
	frame := fitFrame(t, r3.Vec{X: 1}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	otherFrame := frame
	placedFrame := p
	placedFrame.Placement = &frame
	placedCopy := p
	placedCopy.Placement = &otherFrame
	for name, variant := range map[string]onewayclutch.Params{
		"shaft": shaft, "direction": direction, "clearance": clearance, "placement": placed,
	} {
		t.Run(name, func(t *testing.T) {
			require.True(t, base != keyForParams(variant))
		})
	}
	require.True(t, keyForParams(placed) != keyForParams(placedFrame))
	require.True(t, keyForParams(placedFrame) == keyForParams(placedCopy))
}

func TestFixtureReuse(t *testing.T) {
	p := onewayclutch.DefaultParams()
	m, doc := buildCase(t, p)
	again, againDoc := buildCase(t, p)
	require.Same(t, m, again)
	require.Same(t, doc, againDoc)
}

// partsOf lists the ten bodies by name.
func partsOf(m *onewayclutch.Model) map[string]*decad.Body {
	out := map[string]*decad.Body{
		"hub": m.Hub, "ring": m.Ring, "back plate": m.BackPlate, "cover": m.Cover,
	}
	for i := range m.Bolts {
		out[boltName(i)] = m.Bolts[i].Body
		out[nutName(i)] = m.Nuts[i].Body
	}
	return out
}

func boltName(i int) string { return "bolt " + string(rune('1'+i)) }
func nutName(i int) string  { return "nut " + string(rune('1'+i)) }

func TestOneWayClutchDefault(t *testing.T) {
	m, doc := buildDefault(t)
	require.Len(t, doc.Bodies(), 10)
	require.NotEmpty(t, m.Sketches)
	for i, s := range m.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
	require.Equal(t, onewayclutch.Clockwise, onewayclutch.DefaultParams().Freewheel)
	require.InDelta(t, 2.55, m.Dimensions.BoreRadius, 1e-12)
	require.InDelta(t, 2.05, m.Dimensions.BoreFlat, 1e-12)

	// Each body's bounds, from the amended stack (ears design section 3.2). The ring's three ears, at 90, 210 and 330
	// degrees, reach EarReach on +Y and, with their end rounds, EarCircle*cos(30) + EarRadius on both sides of x = 0.
	dm := m.Dimensions
	earX := dm.EarCircle*math.Cos(30*math.Pi/180) + onewayclutch.EarRadius
	want := map[string][2]r3.Vec{
		"back plate": {{X: -dm.PlateRadius, Y: -dm.PlateRadius, Z: 0}, {X: dm.PlateRadius, Y: dm.PlateRadius, Z: onewayclutch.BackPlateThickness}},
		"hub": {{X: -16, Y: -16, Z: onewayclutch.HubFrom},
			{X: 16, Y: 16, Z: onewayclutch.HubFrom + onewayclutch.HubHeight}},
		"cover": {{X: -dm.PlateRadius, Y: -dm.PlateRadius, Z: onewayclutch.CoverFrom}, {X: dm.PlateRadius, Y: dm.PlateRadius, Z: onewayclutch.CoverTo}},
		"ring": {{X: -earX, Y: -dm.RingOuter, Z: onewayclutch.RingFrom},
			{X: earX, Y: dm.EarReach, Z: onewayclutch.RingTo}},
	}
	meshes := map[string]meshgap.Mesh{}
	for name, body := range partsOf(m) {
		box, err := body.Bounds()
		require.NoError(t, err, name)
		tol := box.Bound.Base() + 1e-3
		if w, ok := want[name]; ok {
			require.InDelta(t, w[0].Z, box.Min.Z, tol, name)
			require.InDelta(t, w[1].Z, box.Max.Z, tol, name)
			if name != "hub" {
				require.InDelta(t, w[0].X, box.Min.X, tol, name)
				require.InDelta(t, w[1].X, box.Max.X, tol, name)
				require.InDelta(t, w[0].Y, box.Min.Y, tol, name)
				require.InDelta(t, w[1].Y, box.Max.Y, tol, name)
			}
		}
		lo, hi := 0.0, onewayclutch.CoverTo
		if name == "ring" {
			lo, hi = onewayclutch.RingFrom, onewayclutch.RingTo
		}
		require.GreaterOrEqual(t, box.Min.Z, lo-tol, name)
		require.LessOrEqual(t, box.Max.Z, hi+tol, name)
		require.LessOrEqual(t, box.Max.X, earX+tol, name)
		require.LessOrEqual(t, box.Max.Y, dm.EarReach+tol, name)
		require.GreaterOrEqual(t, box.Min.X, -earX-tol, name)
		require.GreaterOrEqual(t, box.Min.Y, -dm.RingOuter-tol, name)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
		require.NoError(t, err, name)
		meshes[name] = mg
	}
	// The hub's pawl heads reach 15.015 from the axis; the plates span 16.7 mm and the ring 19.1 mm.
	hub, err := m.Hub.Bounds()
	require.NoError(t, err)
	require.InDelta(t, m.Dimensions.HeadReach, max(hub.Max.X, hub.Max.Y, -hub.Min.X, -hub.Min.Y), 0.5)

	requireBoltsClearCover(t, m)

	// No two different bodies overlap, apart from a bolt in its nut, a nominal pair.
	names := make([]string, 0, len(meshes))
	for name := range meshes {
		names = append(names, name)
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			if isBoltNutPair(a, b) || isBoltCoverPair(a, b) {
				continue
			}
			clear, err := meshes[a].ExceedsClearance(meshes[b], 0)
			require.NoError(t, err)
			if !clear {
				gap, _, err := meshes[a].Exceeds(meshes[b], 0)
				require.NoError(t, err)
				require.True(t, clear, "%s and %s overlap or touch (gap bound %g)", a, b, gap)
			}
		}
	}
}

// isBoltCoverPair reports whether a and b are a bolt and the cover: the head rests on its pocket's floor by design, so
// the meshes touch there. requireBoltsClearCover measures that pair in sections instead.
func isBoltCoverPair(a, b string) bool {
	return a == "cover" && len(b) == 6 && b[:4] == "bolt" || b == "cover" && len(a) == 6 && a[:4] == "bolt"
}

// requireBoltsClearCover requires, in the sections through the head and through the shank, each bolt to stand clear of
// the cover by the fastener clearances: 0.3 mm round the head and round the shank.
func requireBoltsClearCover(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	for i := range m.Bolts {
		for _, z := range []float64{onewayclutch.CoverTo - 1.0, onewayclutch.CoverFrom + 1.0} {
			f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
			require.NoError(t, err)
			ob, err := m.Bolts[i].Body.SectionOutline(t.Context(), f, units.Millimeters(0.01))
			require.NoError(t, err)
			oc, err := m.Cover.SectionOutline(t.Context(), f, units.Millimeters(0.01))
			require.NoError(t, err)
			c, err := ob.Clearance(oc)
			require.NoError(t, err)
			require.False(t, c.Overlaps, "bolt %d overlaps the cover at z %g", i+1, z)
			require.GreaterOrEqual(t, c.Distance.Value.Base(), 0.28, "bolt %d to the cover at z %g", i+1, z)
		}
	}
}

// isBoltNutPair reports whether a and b are a bolt and a nut (any site): the bolt runs through the nut's bore.
func isBoltNutPair(a, b string) bool {
	return len(a) == 6 && len(b) == 5 && a[:4] == "bolt" && b[:3] == "nut" ||
		len(b) == 6 && len(a) == 5 && b[:4] == "bolt" && a[:3] == "nut"
}

func TestOneWayClutchContainers(t *testing.T) {
	m, doc := buildDefault(t)
	require.NoError(t, doc.VerifyContainers().Err())
	want := map[string][]*decad.Body{
		onewayclutch.HubContainerName:    {m.Hub},
		onewayclutch.RingContainerName:   {m.Ring},
		onewayclutch.PlatesContainerName: {m.BackPlate, m.Cover},
		onewayclutch.BoltsContainerName:  {m.Bolts[0].Body, m.Bolts[1].Body, m.Bolts[2].Body},
		onewayclutch.NutsContainerName:   {m.Nuts[0].Body, m.Nuts[1].Body, m.Nuts[2].Body},
	}
	children := doc.Root().Children()
	require.Len(t, children, len(want))
	for _, box := range children {
		bodies, ok := want[box.Name()]
		require.True(t, ok, "unexpected container %q", box.Name())
		require.Equal(t, bodies, box.Bodies(), box.Name())
	}
}

// TestOneWayClutchPlainDocumentRefusesBuild pins the reason for NewDocument: the ring's mount nut pockets and the back
// plate's cut meet held facets just past a plain document's chain gate (the back plate's held bound is 0.0010075 mm
// against the gate's 0.00095), so decad refuses the build there. A document from NewDocument builds it.
func TestOneWayClutchPlainDocumentRefusesBuild(t *testing.T) {
	_, err := onewayclutch.Build(t.Context(), decad.New(), onewayclutch.DefaultParams())
	require.ErrorIs(t, err, decad.ErrUnsupported)
	require.ErrorContains(t, err, "back plate")
}

// TestOneWayClutchAdapterPlaced builds the adapter alone, in a plain document and in a placed frame: it needs no chain
// tolerance without the clutch, and its bodies and sites follow the placement.
func TestOneWayClutchAdapterPlaced(t *testing.T) {
	local, err := onewayclutch.BuildAdapter(t.Context(), decad.New(), onewayclutch.DefaultParams())
	require.NoError(t, err)
	frame, err := r3.NewFrame(r3.Vec{X: 12, Y: -7, Z: 3}, r3.Vec{Y: 1}, r3.Vec{X: -1})
	require.NoError(t, err)
	xf, err := r3.FromFrame(frame)
	require.NoError(t, err)
	p := onewayclutch.DefaultParams()
	p.Placement = &frame
	placed, err := onewayclutch.BuildAdapter(t.Context(), decad.New(), p)
	require.NoError(t, err)
	a, b := boxOf(t, local.Body, "adapter"), boxOf(t, placed.Body, "adapter")
	lo, hi := xf.Apply(a.Min), xf.Apply(a.Max)
	tol := a.Bound.Base() + b.Bound.Base() + 1e-3
	require.InDelta(t, math.Min(lo.X, hi.X), b.Min.X, tol)
	require.InDelta(t, math.Min(lo.Y, hi.Y), b.Min.Y, tol)
	require.InDelta(t, math.Min(lo.Z, hi.Z), b.Min.Z, tol)
	require.InDelta(t, math.Max(lo.X, hi.X), b.Max.X, tol)
	require.InDelta(t, math.Max(lo.Y, hi.Y), b.Max.Y, tol)
	require.InDelta(t, math.Max(lo.Z, hi.Z), b.Max.Z, tol)
	for i, site := range local.Sites {
		w := placed.Sites[i]
		require.InDelta(t, 0, xf.Apply(site.Head).Sub(w.Head).Len(), 1e-9, "site %d head", i)
		require.InDelta(t, 0, xf.ApplyDir(site.Dir).Sub(w.Dir).Len(), 1e-9, "site %d dir", i)
	}
}

// TestOneWayClutchInnerTools checks the plates' inner-face tools against the vertices in instructions.md "Plates", and
// that the cover's tool mirrors the back plate's about the plane midway between the plates' outer faces.
func TestOneWayClutchInnerTools(t *testing.T) {
	back := [][2]float64{{9.5, 5.0}, {9.5, 3.4}, {10.3, 3.6}, {16.1, 3.6}, {17.7, 2.0}, {17.7, 5.0}}
	cover := [][2]float64{{9.5, 13.5}, {9.5, 15.1}, {10.3, 14.9}, {16.1, 14.9}, {17.7, 16.5}, {17.7, 13.5}}
	dm, err := onewayclutch.DefaultParams().Derive()
	require.NoError(t, err)
	backInner, coverInner := dm.BackPlateInner(), dm.CoverInner()
	require.Len(t, backInner, len(back))
	require.Len(t, coverInner, len(cover))
	for i := range back {
		require.InDelta(t, back[i][0], backInner[i][0], 1e-9, "back plate vertex %d r", i)
		require.InDelta(t, back[i][1], backInner[i][1], 1e-9, "back plate vertex %d z", i)
		require.InDelta(t, cover[i][0], coverInner[i][0], 1e-9, "cover vertex %d r", i)
		require.InDelta(t, cover[i][1], coverInner[i][1], 1e-9, "cover vertex %d z", i)
		require.InDelta(t, onewayclutch.CoverTo-backInner[i][1], coverInner[i][1], 1e-9,
			"cover vertex %d mirrors the back plate", i)
	}
}
