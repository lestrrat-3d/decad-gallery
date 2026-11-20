package turbinefan_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
)

// TestTurbineFanOnClutch builds the real one-way clutch with its adapter in one document and the clutch-mount fan placed
// on it in another, and checks the face interface, the stack and every pair of bodies through the pair. The two documents
// are separate because both models file their hub in a container named Hub.
func TestTurbineFanOnClutch(t *testing.T) {
	for _, cp := range []onewayclutch.Params{
		onewayclutch.DefaultParams(),
		{Shaft: "608ZZ", Freewheel: onewayclutch.CounterClockwise, BoreClearance: 0.05},
	} {
		t.Run(cp.Shaft, func(t *testing.T) { requireFanOnClutch(t, cp) })
	}
}
