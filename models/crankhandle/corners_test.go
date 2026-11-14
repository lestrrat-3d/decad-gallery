package crankhandle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestCrankHandleCorners applies "Corners and roots" to the crank in its print orientation, axis +Z. The section
// through the set screw turns 45 degrees at each chamfered rim.
func TestCrankHandleCorners(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			heights, err := cornercheck.MidHeights(m.Crank, r3.Vec{Z: 1})
			require.NoError(t, err)
			require.Contains(t, heights, m.Dimensions.ScrewHeight, "a section must cut through the set screw's axis")
			// The D bore's two corners are mating corners: they stay sharp, like the edges of the shaft's flat. A flat
			// of the default depth turns them by less than 45 degrees; a deeper one is listed.
			dm := m.Dimensions
			var exempt []cornercheck.Exempt
			if turn := math.Acos(dm.BoreFlat/dm.BoreRadius) * 180 / math.Pi; turn > 45 {
				w := math.Sqrt(dm.BoreRadius*dm.BoreRadius - dm.BoreFlat*dm.BoreFlat)
				for _, x := range []float64{-w, w} {
					exempt = append(exempt, cornercheck.Exempt{
						Reason: "D bore corner",
						Near:   cornercheck.Disc(r3.Vec{X: x, Y: -dm.BoreFlat}, r3.Vec{Z: 1}, 0.3),
					})
				}
			}
			fs, err := cornercheck.Check(t.Context(), m.Crank, r3.Vec{Z: 1}, exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 8))
		})
	}
}
