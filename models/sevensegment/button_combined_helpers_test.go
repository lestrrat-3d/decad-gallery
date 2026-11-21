package sevensegment

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
	"testing"
)

func combinedMesh(t *testing.T, b *decad.Body) meshgap.Mesh {
	t.Helper()
	m, e := b.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, e)
	out, e := meshgap.New(m.Vertices(), m.Triangles(), m.Bound().Base())
	require.NoError(t, e)
	return out
}
