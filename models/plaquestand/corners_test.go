package plaquestand_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestPlaqueStandCorners applies "Corners and roots" to the stand in every case. It prints lying on its x = 0 face, so
// the print axis is +X and one section at the middle of the width reads the whole profile. The plaque is a nominal
// reference part and is not printed, so it is not checked.
func TestPlaqueStandCorners(t *testing.T) {
	axis := r3.Vec{X: 1}
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			heights, err := cornercheck.MidHeights(m.Stand, axis)
			require.NoError(t, err)
			require.Len(t, heights, 1)
			require.InDelta(t, m.Dimensions.StandWidth/2, heights[0], 1e-6)

			fs, err := cornercheck.Check(t.Context(), m.Stand, axis, nil)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}
