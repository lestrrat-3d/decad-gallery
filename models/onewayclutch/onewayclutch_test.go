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

var cache = map[string]built{}

// buildCase builds a case on first use; the tests only read its bodies.
func buildCase(t *testing.T, name string, p onewayclutch.Params) (*onewayclutch.Model, *decad.Document) {
	t.Helper()
	if b, ok := cache[name]; ok {
		return b.m, b.doc
	}
	doc := decad.New()
	m, err := onewayclutch.Build(t.Context(), doc, p)
	require.NoError(t, err)
	cache[name] = built{m: m, doc: doc}
	return m, doc
}

func buildDefault(t *testing.T) (*onewayclutch.Model, *decad.Document) {
	t.Helper()
	return buildCase(t, "default", onewayclutch.DefaultParams())
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
	earX := onewayclutch.EarCircle*math.Cos(30*math.Pi/180) + onewayclutch.EarRadius
	want := map[string][2]r3.Vec{
		"back plate": {{X: -18.1, Y: -18.1, Z: 0}, {X: 18.1, Y: 18.1, Z: onewayclutch.BackPlateThickness}},
		"hub": {{X: -16, Y: -16, Z: onewayclutch.HubFrom},
			{X: 16, Y: 16, Z: onewayclutch.HubFrom + onewayclutch.HubHeight}},
		"cover": {{X: -18.1, Y: -18.1, Z: onewayclutch.CoverFrom}, {X: 18.1, Y: 18.1, Z: onewayclutch.CoverTo}},
		"ring": {{X: -earX, Y: -onewayclutch.RingOuter, Z: onewayclutch.RingFrom},
			{X: earX, Y: onewayclutch.EarReach, Z: onewayclutch.RingTo}},
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
		require.GreaterOrEqual(t, box.Min.Z, -tol, name)
		require.LessOrEqual(t, box.Max.Z, onewayclutch.CoverTo+tol, name)
		require.LessOrEqual(t, box.Max.X, earX+tol, name)
		require.LessOrEqual(t, box.Max.Y, onewayclutch.EarReach+tol, name)
		require.GreaterOrEqual(t, box.Min.X, -earX-tol, name)
		require.GreaterOrEqual(t, box.Min.Y, -onewayclutch.RingOuter-tol, name)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
		require.NoError(t, err, name)
		meshes[name] = mg
	}
	// The hub's pawl heads reach 15.76 from the axis; the clutch is 18.0 mm long.
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
			gap, clear, err := meshes[a].Exceeds(meshes[b], 0)
			require.NoError(t, err)
			require.True(t, clear, "%s and %s overlap or touch (gap bound %g)", a, b, gap)
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
