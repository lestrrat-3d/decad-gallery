package m2fastener_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func site() m2fastener.Site {
	return m2fastener.Site{Dir: r3.Vec{Z: 1}, Corner: r3.Vec{X: 1}, Length: 10, Cavity: 6}
}

func zRange(t *testing.T, b *decad.Body) (lo, hi float64) {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err)
	return box.Min.Z, box.Max.Z
}
