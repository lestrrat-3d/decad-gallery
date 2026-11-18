package main

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestClampArmPresets checks the presets that need no build: the image sizes, and that the exploded order and the
// parts table name every body of the clamp arm.
func TestClampArmPresets(t *testing.T) {
	for _, view := range clampViews {
		p, ok := presets["clamparm/"+view]
		require.True(t, ok, "preset %s", view)
		require.Equal(t, [2]int{1600, 900}, [2]int{p.width, p.height}, "size of %s", view)
	}

	names := map[string]struct{}{}
	for _, n := range clampBodyNames {
		_, dup := names[n]
		require.False(t, dup, "body %q is listed twice", n)
		names[n] = struct{}{}
	}
	require.Len(t, names, 25)

	ordered := map[string]struct{}{}
	for _, n := range clampExplodedOrder {
		_, known := names[n]
		require.True(t, known, "exploded order names the unknown body %q", n)
		_, dup := ordered[n]
		require.False(t, dup, "%q is in the exploded order twice", n)
		ordered[n] = struct{}{}
	}
	require.Len(t, ordered, len(names), "every body is in the exploded order")

	shown := map[string]struct{}{}
	for _, part := range clampParts {
		p, ok := presets["clamparm/part-"+part.shot]
		require.True(t, ok, "preset for part %s", part.shot)
		require.Equal(t, [2]int{480, 270}, [2]int{p.width, p.height}, "size of part %s", part.shot)
		for _, n := range part.bodies {
			_, known := names[n]
			require.True(t, known, "part %s names the unknown body %q", part.shot, n)
			_, dup := shown[n]
			require.False(t, dup, "%q is in two parts", n)
			shown[n] = struct{}{}
		}
	}
	require.Len(t, shown, len(names), "every body is in the parts table")
}

// TestClampArmExploded checks the exploded view of the real default clamp arm: every body has an offset, no two
// exploded bounding boxes intersect or touch, and the exploded parts fill at least 80% of the image width or height.
func TestClampArmExploded(t *testing.T) {
	a, err := newClampArm(t.Context(), clamparm.DefaultParams())
	require.NoError(t, err)
	offsets := a.clampExplodedOffsets()
	require.Len(t, offsets, len(clampBodyNames))

	type box struct{ lo, hi r3.Vec }
	boxes := map[string]box{}
	for _, name := range clampBodyNames {
		offset, ok := offsets[name]
		require.True(t, ok, "%s has an exploded offset", name)
		lo, hi := meshBounds(a.meshes[name])
		boxes[name] = box{lo.Add(offset), hi.Add(offset)}
	}
	for i, n := range clampBodyNames {
		for _, m := range clampBodyNames[i+1:] {
			p, q := boxes[n], boxes[m]
			apart := p.hi.X < q.lo.X || q.hi.X < p.lo.X || p.hi.Y < q.lo.Y || q.hi.Y < p.lo.Y ||
				p.hi.Z < q.lo.Z || q.hi.Z < p.lo.Z
			require.True(t, apart, "%s and %s overlap or touch in the exploded view", n, m)
		}
	}

	scene, err := a.clampExplodedScene()
	require.NoError(t, err)
	spanX, spanY := projectedSpan(scene.Camera, 16.0/9, scene.Models)
	require.GreaterOrEqual(t, math.Max(spanX, spanY), 0.8, "exploded parts fill %.2f of the width and %.2f of the height",
		spanX, spanY)
}
