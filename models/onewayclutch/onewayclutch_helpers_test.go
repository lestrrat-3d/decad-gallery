package onewayclutch_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
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

func nutName(i int) string { return "nut " + string(rune('1'+i)) }

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
