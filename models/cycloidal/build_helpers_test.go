package cycloidal_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func chainDocument() *decad.Document {
	return decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
}

// overlap returns the volume the two bodies share, zero when they only touch or are apart. It intersects copies
// moved to the document root, because a boolean needs both operands in one container and the drive's bodies sit
// in one container per kind.
func overlap(t *testing.T, a, b *decad.Body) float64 {
	t.Helper()
	ca, err := decad.Copy(t.Context(), a)
	require.NoError(t, err)
	cb, err := decad.Copy(t.Context(), b)
	require.NoError(t, err)
	require.NoError(t, a.Document().Root().AddBody(ca))
	require.NoError(t, a.Document().Root().AddBody(cb))
	both, err := decad.Intersect(t.Context(), ca, cb)
	if err != nil {
		require.ErrorContains(t, err, "the intersect result is empty")
		return 0
	}
	volume, err := both.Volume()
	require.NoError(t, err)
	return volume.Value.Base()
}
