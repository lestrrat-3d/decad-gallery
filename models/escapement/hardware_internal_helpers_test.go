package escapement

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// requireNoOverlap checks that a and b share no volume by their section outlines at each z.
func requireNoOverlap(t *testing.T, a, b *decad.Body, zs ...float64) {
	t.Helper()
	for _, z := range zs {
		oa, ob := sectionAt(t, a, z), sectionAt(t, b, z)
		c, err := oa.Clearance(ob)
		require.NoError(t, err)
		require.False(t, c.Overlaps, "overlap at z %v", z)
	}
}
