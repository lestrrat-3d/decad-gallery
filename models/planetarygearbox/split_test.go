package planetarygearbox

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSplitShafts applies "Split shafts" to every shaft and pin of the default gearbox.
func TestSplitShafts(t *testing.T) {
	cornersDefault.once.Do(func() {
		cornersDefault.box, cornersDefault.err = Build(t.Context(), NewDocument(), DefaultParams())
	})
	require.NoError(t, cornersDefault.err)
	requireSplit(t, cornersDefault.box, DefaultParams())
}

func TestSplitPlanet605Shaft608(t *testing.T) { requireSplitKit(t, "planet605-shaft608", false) }
func TestSplitPlanet608Shaft605(t *testing.T) { requireSplitKit(t, "planet608-shaft605", false) }
func TestSplitPlanet608Shaft608(t *testing.T) { requireSplitKit(t, "planet608-shaft608", false) }
func TestSplitTwoStages608Shaft(t *testing.T) { requireSplitKit(t, "planet605-shaft608", true) }

func TestSplitTwoStages(t *testing.T) {
	p := DefaultParams()
	p.Stages = 2
	requireSplitVariant(t, p)
}

func TestSplitNoFlat(t *testing.T) {
	p := DefaultParams()
	p.FlatDepth = 0
	requireSplitVariant(t, p)
}
