package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func eachClutchCase(t *testing.T, fn func(t *testing.T, c fanCase, m *turbinefan.Model, doc *decad.Document)) {
	t.Helper()
	for _, c := range clutchCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			fn(t, c, m, doc)
		})
	}
}

// TestTurbineFanClutchBuild checks the clutch mount's 17 bodies and its five containers in order.
func TestTurbineFanClutchBuild(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, doc *decad.Document) {
		require.Len(t, doc.Bodies(), 17)
		require.NoError(t, doc.VerifyContainers().Err())
		require.Nil(t, m.SetScrew)
		require.Nil(t, m.SetScrewNut)
		want := map[string][]*decad.Body{
			turbinefan.HubContainerName:    {m.Hub},
			turbinefan.BladesContainerName: {m.Blades[0], m.Blades[1], m.Blades[2]},
			turbinefan.ClampContainerName:  {m.Clamp},
			turbinefan.BoltsContainerName: {m.BladeBolts[0].Body, m.BladeBolts[1].Body, m.BladeBolts[2].Body,
				m.EarBolts[0].Body, m.EarBolts[1].Body, m.EarBolts[2].Body},
			turbinefan.NutsContainerName: {m.BladeNuts[0].Body, m.BladeNuts[1].Body, m.BladeNuts[2].Body,
				m.EarNuts[0].Body, m.EarNuts[1].Body, m.EarNuts[2].Body},
		}
		children := doc.Root().Children()
		require.Len(t, children, len(want))
		for _, box := range children {
			bodies, ok := want[box.Name()]
			require.True(t, ok, "unexpected container %q", box.Name())
			require.Equal(t, bodies, box.Bodies(), box.Name())
			for _, b := range bodies {
				require.True(t, b.IsSolid(), box.Name())
			}
		}
	})
}

// TestTurbineFanClutchSketches requires 33 fully constrained sketches from BuildParts and the ear fasteners' own.
func TestTurbineFanClutchSketches(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, _ *decad.Document) {
		require.Len(t, m.Sketches, 33)
		for i, s := range m.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
		for i, b := range m.EarBolts {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Profile), "ear bolt %d profile", i+1)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Socket), "ear bolt %d socket", i+1)
		}
		for i, n := range m.EarNuts {
			for _, s := range []*sketch.Sketch{n.Hexagon, n.LowerChamfer, n.UpperChamfer} {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "ear nut %d", i+1)
			}
		}
	})
}

// TestTurbineFanClutchBounds checks the hub's and the clamp ring's bounds and tessellates four bodies with every
// verification on.
func TestTurbineFanClutchBounds(t *testing.T) {
	m, _ := built(t, caseNamed(t, "clutch-605ZZ"))
	hub, err := m.Hub.Bounds()
	require.NoError(t, err)
	tol := hub.Bound.Base() + 1e-3
	require.InDelta(t, -28, hub.Min.X, tol)
	require.InDelta(t, 28, hub.Max.X, tol)
	require.InDelta(t, -28, hub.Min.Y, tol)
	require.InDelta(t, 28, hub.Max.Y, tol)
	require.InDelta(t, -3, hub.Min.Z, tol)
	require.InDelta(t, 20, hub.Max.Z, tol)
	clamp, err := m.Clamp.Bounds()
	require.NoError(t, err)
	tol = clamp.Bound.Base() + 0.01
	require.InDelta(t, -28, clamp.Min.X, tol)
	require.InDelta(t, 28, clamp.Max.X, tol)
	require.InDelta(t, 28, clamp.Max.Y, tol)
	// The 270 degree slot spans only z -27.5 to -25.5, so the box still reaches -28 above and below it; the slot's own
	// section reaches only the wall beside it.
	require.InDelta(t, -28, clamp.Min.Y, tol)
	lowest := math.Inf(1)
	for _, l := range section(t, m.Clamp, atZ(t, -26.5)).Loops {
		for _, v := range l {
			lowest = math.Min(lowest, v.Y)
		}
	}
	require.InDelta(t, -math.Sqrt(28*28-2.17*2.17), lowest, 0.01)
	require.InDelta(t, turbinefan.ClampBack, clamp.Min.Z, tol)
	require.InDelta(t, turbinefan.ClampBack+turbinefan.ClampThickness, clamp.Max.Z, tol)
	for name, item := range map[string]struct {
		body *decad.Body
		tol  float64
	}{
		"hub": {m.Hub, 0.05}, "clamp": {m.Clamp, 0.05}, "ear bolt": {m.EarBolts[0].Body, 0.01},
		"ear nut": {m.EarNuts[0].Body, 0.01},
	} {
		require.True(t, meshOf(t, item.body, item.tol).BoundaryVerified(), name)
	}
}
