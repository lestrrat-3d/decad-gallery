package main

// CI: isolated-tests

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestSwissPresets checks the presets that need no build: the sizes of the images, the exploded layers and the parts
// table cover the movement's bodies exactly.
func TestSwissPresets(t *testing.T) {
	for name, size := range map[string][2]int{
		"hero": {1600, 900}, "index": {480, 270},
		"assembled-front": {1200, 675}, "assembled-angle": {1200, 675},
		"assembled-side": {1200, 675}, "assembled-back": {1000, 1000},
		"closeup": {1200, 675}, "exploded": {1600, 900},
	} {
		p, ok := swissleverModel().Presets[name]
		require.True(t, ok, "preset %s", name)
		require.Equal(t, size, [2]int{p.Width, p.Height}, "size of %s", name)
	}
	require.GreaterOrEqual(t, swissExplodedWidth, 1600)
	require.GreaterOrEqual(t, swissExplodedHeight, 900)

	names := map[string]struct{}{}
	for _, n := range swissBodyNames() {
		names[n] = struct{}{}
	}
	layered := map[string]struct{}{}
	for _, layer := range swissExplodedLayers {
		for _, n := range layer {
			_, known := names[n]
			require.True(t, known, "exploded layer names the unknown body %q", n)
			_, dup := layered[n]
			require.False(t, dup, "%q is in two exploded layers", n)
			layered[n] = struct{}{}
		}
	}
	require.Len(t, layered, len(names), "every body is in an exploded layer")

	shown := map[string]struct{}{}
	for _, part := range swissParts {
		p, ok := swissleverModel().Presets["part-"+part.shot]
		require.True(t, ok, "preset for part %s", part.shot)
		require.Equal(t, [2]int{480, 270}, [2]int{p.Width, p.Height})
		for _, n := range part.bodies {
			_, known := names[n]
			require.True(t, known, "part %s names the unknown body %q", part.shot, n)
			shown[n] = struct{}{}
		}
	}
	// Identical bodies share one image: six bearings, four bolts, four nuts, and the Balance Staff with the Pallet Arbor.
	for n := range names {
		if _, ok := shown[n]; ok {
			continue
		}
		twin := strings.HasSuffix(n, "Bearing") || strings.HasPrefix(n, "Bolt ") || strings.HasPrefix(n, "Nut ") ||
			n == swisslever.BalanceStaffName
		require.True(t, twin, "%q has no part image", n)
	}
}

// TestSwissExploded checks the exploded view of the real movement: every body has an offset, the layers lie in slabs
// that keep the gap, no two bodies overlap or touch, and the exploded parts fill at least 80% of the image.
func TestSwissExploded(t *testing.T) {
	b := swissBuildOK(t)
	offsets, err := b.swissExplodedOffsets()
	require.NoError(t, err)

	var names []string
	var meshes []meshgap.Mesh
	for _, nb := range b.named {
		offset, ok := offsets[nb.Name]
		require.True(t, ok, "%s has an exploded offset", nb.Name)
		require.Zero(t, offset.X)
		require.Zero(t, offset.Y)
		mesh := b.meshes[nb.Name]
		vs := mesh.Vertices()
		moved := make([]r3.Vec, len(vs))
		for i, v := range vs {
			moved[i] = v.Add(offset)
		}
		mg, err := meshgap.New(moved, mesh.Triangles(), mesh.Bound().Base())
		require.NoError(t, err)
		names = append(names, nb.Name)
		meshes = append(meshes, mg)
	}
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			gap, separated, err := meshes[i].Exceeds(meshes[j], 0)
			require.NoError(t, err)
			require.True(t, separated, "%s and %s overlap or touch in the exploded view (gap %v)", names[i], names[j], gap)
		}
	}

	// Each layer starts swissExplodedGap in front of the one behind it.
	var front float64
	for i, layer := range swissExplodedLayers {
		low, high := math.Inf(1), math.Inf(-1)
		for _, n := range layer {
			l, h := meshBounds(b.meshes[n])
			low, high = math.Min(low, l.Z+offsets[n].Z), math.Max(high, h.Z+offsets[n].Z)
		}
		if i > 0 {
			require.InDelta(t, swissExplodedGap, low-front, 1e-6, "gap before layer %d", i)
		}
		front = high
	}

	scene, err := b.explodedScene()
	require.NoError(t, err)
	spanX, spanY := projectedSpan(scene.Camera, float64(swissExplodedWidth)/swissExplodedHeight, scene.Models)
	require.GreaterOrEqual(t, math.Max(spanX, spanY), 0.8, "exploded parts fill %.2f of the width and %.2f of the height",
		spanX, spanY)
}

// TestSwissMovers checks that the animation's movers leave every body within the build backoff of where the movement
// was built, at the default state, and turn the balance group by exactly the balance angle at the swing's ends.
func TestSwissMovers(t *testing.T) {
	p := swisslever.DefaultParams()
	g, err := swisslever.Layout(p)
	require.NoError(t, err)
	built, err := g.StateAt(p.Phase)
	require.NoError(t, err)
	b := &swissBuild{movement: &swisslever.Movement{State: built}, geometry: g}
	movers, err := b.swissMovers()
	require.NoError(t, err)
	require.Len(t, movers, 13)
	end, err := g.StateAt(0.5)
	require.NoError(t, err)
	for _, m := range movers {
		require.InDelta(t, 0, m.angle(built), 3*swisslever.BuildBackoff, "%s at the built state", m.name)
		if m.name == swisslever.BalanceWheelName {
			require.InDelta(t, end.BalanceAngle-built.BalanceAngle, m.angle(end), 1e-12)
		}
	}
}
