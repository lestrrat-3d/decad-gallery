package escapement

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSplitShafts reads the arbors and the suspension pin as halves on their cut face, at the default pose and at
// another phase.
func TestSplitShafts(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	t.Run("default", func(t *testing.T) { requireSplitShafts(t, m) })

	other, err := sharedOtherPhase(t)
	require.NoError(t, err)
	t.Run("phase 0.6", func(t *testing.T) { requireSplitShafts(t, other) })
}
