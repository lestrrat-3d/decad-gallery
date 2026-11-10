package m2fastener_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func site() m2fastener.Site {
	return m2fastener.Site{Dir: r3.Vec{Z: 1}, Corner: r3.Vec{X: 1}, Length: 10, Cavity: 6}
}

func TestValidate(t *testing.T) {
	require.NoError(t, site().Validate())
	s := site()
	s.Length = 11
	require.Error(t, s.Validate())
	s = site()
	s.Dir = r3.Vec{Z: 2}
	require.Error(t, s.Validate())
	s = site()
	s.Corner = r3.Vec{X: 1, Z: 1}.Scale(1 / r3.Vec{X: 1, Z: 1}.Len())
	require.Error(t, s.Validate())
	s = site()
	s.Cavity = 2
	require.Error(t, s.Validate())
}

func zRange(t *testing.T, b *decad.Body) (lo, hi float64) {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err)
	return box.Min.Z, box.Max.Z
}

func TestTools(t *testing.T) {
	doc := decad.New()
	s := site()
	pocket, ps, err := m2fastener.PocketTool(t.Context(), doc, s)
	require.NoError(t, err)
	hole, hs, err := m2fastener.HoleTool(t.Context(), doc, s)
	require.NoError(t, err)
	cavity, cs, err := m2fastener.CavityTool(t.Context(), doc, s)
	require.NoError(t, err)
	for _, sk := range [...]*sketch.Sketch{ps, hs, cs} {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))
	}

	lo, hi := zRange(t, pocket)
	require.InDelta(t, -1, lo, 1e-3)
	require.InDelta(t, 2, hi, 1e-3)
	box, err := pocket.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 2.2, box.Max.X, 1e-3)

	lo, hi = zRange(t, hole)
	require.InDelta(t, -1, lo, 1e-3)
	require.InDelta(t, 13, hi, 1e-3)

	lo, hi = zRange(t, cavity)
	cb, err := cavity.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 2.51, cb.Max.X, 5e-3)
	require.InDelta(t, 6, lo, 1e-3)
	require.InDelta(t, 8.0, hi, 1e-3)
}

func TestPlacement(t *testing.T) {
	doc := decad.New()
	s := site()
	bolt, err := m2fastener.PlaceBolt(t.Context(), doc, s)
	require.NoError(t, err)
	box, err := bolt.Body.Bounds()
	require.NoError(t, err)
	// Dir is +Z, so the head's top face is the bolt's lowest face and the tip is 2 + 10 beyond it.
	require.InDelta(t, 0, box.Min.Z, 1e-3, "the head is flush with the face")
	require.InDelta(t, 12, box.Max.Z, 1e-3)

	nut, err := m2fastener.PlaceNut(t.Context(), doc, s)
	require.NoError(t, err)
	nb, err := nut.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 7.0, (nb.Min.Z+nb.Max.Z)/2, 1e-3)
}
