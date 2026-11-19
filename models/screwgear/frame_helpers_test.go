package screwgear_test

import (
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func requireVec(t *testing.T, want, got r3.Vec, tol float64) {
	t.Helper()
	require.InDelta(t, want.X, got.X, tol)
	require.InDelta(t, want.Y, got.Y, tol)
	require.InDelta(t, want.Z, got.Z, tol)
}
