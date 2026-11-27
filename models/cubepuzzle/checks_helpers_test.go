package cubepuzzle_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

type checkedPart struct {
	name      string
	mesh      meshgap.Mesh
	vertices  []r3.Vec
	triangles [][3]int
	bound     float64
}

func checkMesh(t *testing.T, ctx context.Context, name string, body *decad.Body, chord float64) checkedPart {
	t.Helper()
	require.True(t, body.IsSolid(), "%s is not solid", name)
	m, err := body.Tessellate(ctx, units.Millimeters(chord), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err, "%s mesh", name)
	v, f, bound := m.Vertices(), m.Triangles(), m.Bound().Base()
	g, err := meshgap.New(v, f, bound)
	require.NoError(t, err, "%s gap mesh", name)
	return checkedPart{name: name, mesh: g, vertices: v, triangles: f, bound: bound}
}

func checkParts(t *testing.T, prefix string, bodies []*decad.Body, chord float64) []checkedPart {
	t.Helper()
	parts := make([]checkedPart, 0, len(bodies))
	for i, body := range bodies {
		parts = append(parts, checkMesh(t, t.Context(), fmt.Sprintf("%s %d", prefix, i), body, chord))
	}
	return parts
}

func requireGap(t *testing.T, a, b checkedPart, clearance float64) {
	t.Helper()
	lower, ok, err := a.mesh.Exceeds(b.mesh, clearance)
	require.NoError(t, err, "%s and %s", a.name, b.name)
	require.True(t, ok, "%s and %s: clearance %.3f mm, proven lower bound %.6f mm", a.name, b.name, clearance, lower)
}
