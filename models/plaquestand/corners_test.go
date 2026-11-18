package plaquestand_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestPlaqueStandCorners applies "Corners and roots" to every printed part in every case. A bracket prints lying on a
// side face, so its print axis is +X and one section at the middle of its width reads the profile, the channel, the
// bolt hole and the pocket's roof. The rail prints flat on its bottom face (axis +Z). The plaque is a nominal
// reference part and is not printed, so it is not checked.
func TestPlaqueStandCorners(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			for k, b := range m.Brackets {
				x0 := float64(k) * (m.Dimensions.Span - plaquestand.BracketWidth)
				heights, err := cornercheck.MidHeights(b, r3.Vec{X: 1})
				require.NoError(t, err)
				require.Len(t, heights, 1)
				require.InDelta(t, x0+plaquestand.BracketWidth/2, heights[0], 1e-6)
				fs, err := cornercheck.Check(t.Context(), b, r3.Vec{X: 1}, nil)
				require.NoError(t, err)
				require.Empty(t, fs, "bracket %d: %s", k+1, cornercheck.Report(fs, 5))
			}
			heights, err := cornercheck.MidHeights(m.Rail, r3.Vec{Z: 1})
			require.NoError(t, err)
			require.Len(t, heights, 2)
			require.InDelta(t, 5.4, heights[0], 1e-6)
			require.InDelta(t, 7.4, heights[1], 1e-6)
			fs, err := cornercheck.Check(t.Context(), m.Rail, r3.Vec{Z: 1}, nil)
			require.NoError(t, err)
			require.Empty(t, fs, "rail: %s", cornercheck.Report(fs, 5))
		})
	}
}
