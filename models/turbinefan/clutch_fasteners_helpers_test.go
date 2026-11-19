package turbinefan_test

import (
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// sitePlane is the plane through the shaft axis and a site's line, U along the line outward and V along +Z.
func sitePlane(t *testing.T, u r3.Vec) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{}, u, r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}
