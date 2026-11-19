package swisslever

// CI: isolated-tests

import (
	"testing"
)

// TestSplitShafts reads the arbors as halves on their cut face, at the default pose and at another phase.
func TestSplitShafts(t *testing.T) {
	t.Run("default", func(t *testing.T) { requireSplitShafts(t, posedArbors(t, DefaultParams().Phase)) })
	t.Run("phase 0.6", func(t *testing.T) { requireSplitShafts(t, posedArbors(t, otherPhase)) })
}

// TestSplitShaftsBuilt reads the arbors of the built, posed movement as halves on their cut face, at the default pose
// and at another phase, so the check follows the pose Build gives each arbor (the lever and wheel backed off their
// last contact), not a pose the test computes for itself.
func TestSplitShaftsBuilt(t *testing.T) {
	t.Run("default", func(t *testing.T) { requireSplitShafts(t, builtArbors(t, DefaultParams().Phase)) })
	t.Run("phase 0.6", func(t *testing.T) { requireSplitShafts(t, builtArbors(t, otherPhase)) })
}
