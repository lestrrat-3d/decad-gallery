package fastenertool_test

import (
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func frame(t *testing.T) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: 3}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}
