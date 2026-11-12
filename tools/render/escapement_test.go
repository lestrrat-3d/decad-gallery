package main

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestEscapementExploded checks that no two bodies of the exploded view overlap, and that every body has an offset.
func TestEscapementExploded(t *testing.T) {
	b, err := buildEscapement(t.Context())
	require.NoError(t, err)
	var names []string
	var meshes []meshgap.Mesh
	for _, nb := range b.named {
		offset, ok := escapementExplode[nb.Name]
		require.True(t, ok, "%s has an exploded offset", nb.Name)
		mesh := b.meshes[nb.Name]
		vs := mesh.Vertices()
		moved := make([]r3.Vec, len(vs))
		for i, v := range vs {
			moved[i] = v.Add(r3.NewVec(0, 0, offset))
		}
		mg, err := meshgap.New(moved, mesh.Triangles(), mesh.Bound().Base())
		require.NoError(t, err)
		names = append(names, nb.Name)
		meshes = append(meshes, mg)
	}
	require.Len(t, names, 29)
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			gap, separated, err := meshes[i].Exceeds(meshes[j], 0)
			require.NoError(t, err)
			require.True(t, separated, "%s and %s overlap or touch in the exploded view (gap %v)", names[i], names[j], gap)
		}
	}
}
