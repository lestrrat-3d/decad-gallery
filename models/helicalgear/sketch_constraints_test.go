package helicalgear

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchassert"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestRootDiscSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	s, err := rootDiscSketch(t.Context(), sketch.NewWorld(), p)
	require.NoError(t, err)
	sketchassert.FullyConstrained(t, "root disc", s)
}
