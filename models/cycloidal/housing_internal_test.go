package cycloidal

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBuildHousing(t *testing.T) {
	doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
	r, err := begin(t.Context(), doc, DefaultParams())
	require.NoError(t, err)
	housing, err := r.buildHousing(t.Context())
	require.NoError(t, err)
	verifyBody(t, "housing", housing)
}
