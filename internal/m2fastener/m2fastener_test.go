package m2fastener_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

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

func TestHexPocketTool(t *testing.T) {
	s := site()
	tool, sk, err := m2fastener.HexPocketTool(t.Context(), decad.New(), s, m2fastener.NutCavityHeight+m2fastener.Stock)
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))

	lo, hi := zRange(t, tool)
	require.InDelta(t, s.Cavity, lo, 1e-3)
	require.InDelta(t, s.CavityFar()+m2fastener.Stock, hi, 1e-3)

	// Corners lie on NutCavityRadius, the first along Corner (+X); flats are NutCavityRadius*sqrt(3)/2 across the axis.
	box, err := tool.Bounds()
	require.NoError(t, err)
	require.InDelta(t, m2fastener.NutCavityRadius, box.Max.X, 1e-3)
	require.InDelta(t, -m2fastener.NutCavityRadius, box.Min.X, 1e-3)
	flat := m2fastener.NutCavityRadius * math.Sqrt(3) / 2
	require.InDelta(t, flat, box.Max.Y, 1e-3)
	require.InDelta(t, -flat, box.Min.Y, 1e-3)

	vol, err := tool.Volume()
	require.NoError(t, err)
	area := 3 * math.Sqrt(3) / 2 * m2fastener.NutCavityRadius * m2fastener.NutCavityRadius
	require.InEpsilon(t, area*(m2fastener.NutCavityHeight+m2fastener.Stock), vol.Value.Base(), 1e-3)
}

func TestRoundedHexPocketTool(t *testing.T) {
	s := site()
	s.Dir = r3.Vec{Z: -1}
	depth := m2fastener.NutCavityHeight + m2fastener.Stock
	sharp, _, err := m2fastener.HexPocketTool(t.Context(), decad.New(), s, depth)
	require.NoError(t, err)
	tool, sk, err := m2fastener.RoundedHexPocketTool(t.Context(), decad.New(), s, depth, 0.6)
	require.NoError(t, err)
	require.True(t, tool.IsSolid())
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))

	vs, err := sharp.Volume()
	require.NoError(t, err)
	vr, err := tool.Volume()
	require.NoError(t, err)
	// The round takes (1 - pi/4) r^2 off each of the six floor edges, each 2 NutCavityRadius / 2 long.
	k := (1 - math.Pi/4) * 0.6 * 0.6 * 6 * m2fastener.NutCavityRadius
	removed := vs.Value.Base() - vr.Value.Base()
	require.GreaterOrEqual(t, removed, 0.5*k)
	require.LessOrEqual(t, removed, 1.5*k)
}

func TestDimensionsFromPreset(t *testing.T) {
	require.Equal(t, "M2", m2fastener.PresetName)
	require.Equal(t, fastener.HeadSocketCap, m2fastener.BoltSize.Head)
	const tol = 1e-12
	require.InDelta(t, m2fastener.BoltSize.ShankDiameter/2+m2fastener.HeadClearance, m2fastener.HoleRadius, tol)
	require.InDelta(t, m2fastener.BoltSize.SocketCap.HeadDiameter/2+m2fastener.HeadClearance, m2fastener.HeadPocketRadius, tol)
	require.InDelta(t, m2fastener.BoltSize.HeadHeight, m2fastener.HeadPocketDepth, tol)
	require.InDelta(t, m2fastener.NutSize.Thickness+2*m2fastener.NutClearance, m2fastener.NutCavityHeight, tol)
	require.Equal(t, 4/math.Sqrt(3)+0.2, m2fastener.NutCavityRadius)
}
