package planetarygearbox

// CI: isolated-tests

import (
	"testing"
)

// TestSplitShafts applies "Split shafts" to every shaft and pin of the default gearbox.
func TestSplitShafts(t *testing.T) {
	_, box := SharedDefaultBuild(t)
	requireSplit(t, box, DefaultParams())
}

func TestSplitPlanet605Shaft608(t *testing.T) { requireSplitKit(t, "planet605-shaft608", false) }
func TestSplitPlanet608Shaft605(t *testing.T) { requireSplitKit(t, "planet608-shaft605", false) }
func TestSplitPlanet608Shaft608(t *testing.T) { requireSplitKit(t, "planet608-shaft608", false) }
func TestSplitTwoStages608Shaft(t *testing.T) { requireSplitKitAlone(t, "planet605-shaft608", true) }

func TestSplitTwoStages(t *testing.T) {
	p := DefaultParams()
	p.Stages = 2
	requireSplitVariant(t, p)
}

func TestSplitNoFlat(t *testing.T) {
	p := DefaultParams()
	p.FlatDepth = 0
	// No corner test reads this variant. Leave the paired corner and shaft fixture in its slot.
	requireSplit(t, buildCornerSplitVariant(t, p), p)
}
