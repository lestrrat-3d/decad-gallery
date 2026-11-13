package main

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestEscapementExploded checks that the derived exploded view moves every body, that no two bodies overlap, that no
// body hides another from the exploded camera except within a stack of bodies that Decad lays one behind another
// along an arbor, and that every moved body has a guide line.
func TestEscapementExploded(t *testing.T) {
	b, err := buildEscapement(t.Context())
	require.NoError(t, err)
	layout, err := b.explode(t.Context())
	require.NoError(t, err)
	require.NotNil(t, layout.Spread)
	require.True(t, layout.Spread.Ran, "the thin escapement spreads its layers")
	require.Len(t, layout.Parts, 29)

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

	hidden, err := layout.Hidden(escapementExplodeDir.Scale(-1))
	require.NoError(t, err)
	stack := map[string]parts.ExplodedPart{}
	for _, part := range layout.Parts {
		stack[part.Name] = part
	}
	var pairs []string
	for _, o := range hidden {
		front, back := stack[o.Front], stack[o.Back]
		if front.Stack > 0 && back.Stack > 0 && front.Stage == back.Stage {
			continue
		}
		pairs = append(pairs, o.Front+" over "+o.Back)
	}
	require.Empty(t, pairs, "bodies that may hide another from the exploded camera")

	guided := map[string]bool{}
	for _, guide := range layout.Guides {
		guided[guide.Part] = true
	}
	for _, part := range layout.Parts {
		if part.Name != escapement.BackPlateName {
			require.True(t, guided[part.Name], "%s has a guide line", part.Name)
		}
	}
}
