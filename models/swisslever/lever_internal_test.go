package swisslever

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLeverSketches requires the lever outline (two slots, the dart, the corner and gap rounds, three frame rounds) and
// the notch tool to verify fully constrained with DOF 0.
func TestLeverSketches(t *testing.T) {
	b := partBuilder(t)
	s, err := b.leverSketch()
	require.NoError(t, err)
	_, err = b.profile(s, 2, "lever")
	require.NoError(t, err)
	_, err = b.notchTool()
	require.NoError(t, err)
	requireSketchesConstrained(t, b, 2)
}

// TestLever is blocked: decad refuses the notch cut of steps.md, so no lever body exists yet.
func TestLever(t *testing.T) {
	b := partBuilder(t)
	_, err := b.buildLever()
	if err != nil {
		t.Skipf("blocked: %v", err)
	}
	t.Fatal("decad accepts the lever notch cut: write the lever body, fork and key tests")
}
