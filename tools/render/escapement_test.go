package main

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestEscapementExploded checks the derived exploded view against the acceptance criteria of Decad's layer spread
// (agent-docs/parts-design.md "Escapement acceptance"): every body moves, no two bodies overlap, the layout stays
// compact, and a part covered completely by a part it is joined to lies on that part's axis.
func TestEscapementExploded(t *testing.T) {
	b, err := buildEscapement(t.Context())
	require.NoError(t, err)
	layout, err := b.explode(t.Context())
	require.NoError(t, err)
	require.NotNil(t, layout.Spread)
	require.True(t, layout.Spread.Ran, "the thin escapement spreads its layers")
	require.Len(t, layout.Parts, 29)

	for name, extent := range map[string]struct {
		got units.Value
		max float64
	}{"width": {layout.Spread.Width, 150}, "depth": {layout.Spread.Depth, 250}, "height": {layout.Spread.Height, 350}} {
		mm, err := extent.got.In(units.Millimeter)
		require.NoError(t, err)
		require.Less(t, mm, extent.max, "the exploded %s in mm", name)
	}

	offsets := map[string]r3.Vec{}
	for _, part := range layout.Parts {
		offsets[part.Name] = part.Offset
	}
	var names []string
	var meshes []meshgap.Mesh
	for _, nb := range b.named {
		offset, ok := offsets[nb.Name]
		require.True(t, ok, "%s has an exploded offset", nb.Name)
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

	set, err := b.movement.Set(b.doc, b.params)
	require.NoError(t, err)
	joined := map[[2]string]bool{}
	for _, j := range set.Joints() {
		joined[[2]string{j.A, j.B}], joined[[2]string{j.B, j.A}] = true, true
	}
	hidden, err := layout.Hidden(escapementExplodeDir.Scale(-1))
	require.NoError(t, err)
	for _, o := range hidden {
		if !joined[[2]string{o.Front, o.Back}] {
			continue
		}
		covered, err := o.Covered.In(units.One)
		require.NoError(t, err)
		if covered < 1 {
			continue
		}
		across := offsets[o.Front].Sub(offsets[o.Back]).Cross(layout.Spread.Axis)
		require.InDelta(t, 0, across.Len(), 1e-6, "%s covers %s but they are not on one axis", o.Front, o.Back)
	}

	guided := map[string]bool{}
	for _, guide := range layout.Guides {
		guided[guide.Part] = true
	}
	for _, part := range layout.Parts {
		if part.Name != escapement.BackPlateName && part.Offset != (r3.Vec{}) {
			require.True(t, guided[part.Name], "%s has a guide", part.Name)
		}
	}
}
