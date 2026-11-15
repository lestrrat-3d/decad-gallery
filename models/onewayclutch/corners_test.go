package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// cornerMinWall is the press-fit wall of "Corners and roots": four 0.4 mm lines.
const cornerMinWall = 1.6

// TestOneWayClutchCorners applies "Corners and roots" to the four printed parts, each in its print orientation (axis
// vertical, so +Z in the clutch's frame), in every preset and both directions. No part has a finding, so the test
// exempts nothing: every corner listed under "Retained sharp corners" in instructions.md passes the rule's limits
// (the D bore's corners, the head tip and lock face, the ring's tooth tips and roots) or lies between layers (the
// head pocket and nut cavity rims), where no section reads it. The hub, back plate and cover are press-fit rings, so
// each section's bore keeps a wall of 1.6 mm or more to every other outline.
func TestOneWayClutchCorners(t *testing.T) {
	axis := r3.Vec{Z: 1}
	for _, tc := range dirCases() {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := buildCase(t, tc.name, tc.params)
			for _, part := range []struct {
				name  string
				bored bool
			}{{"hub", true}, {"ring", false}, {"back plate", true}, {"cover", true}} {
				t.Run(part.name, func(t *testing.T) {
					body := partsOf(m)[part.name]
					fs, err := cornercheck.Check(t.Context(), body, axis, nil)
					require.NoError(t, err)
					require.Empty(t, fs, cornercheck.Report(fs, 5))
					if !part.bored {
						return
					}
					heights, err := cornercheck.MidHeights(body, axis)
					require.NoError(t, err)
					require.NotEmpty(t, heights)
					for _, z := range heights {
						s := fitSection(t, body, fitPlane(t, z))
						bore := fitBoreLoop(t, s)
						wall := math.Inf(1)
						for _, loop := range s.Loops {
							if &loop[0] == &bore[0] {
								continue
							}
							for _, p := range loop {
								wall = math.Min(wall, fitNearest(p, [][]r3.Vec{bore}))
							}
						}
						for _, p := range bore {
							for _, loop := range s.Loops {
								if &loop[0] != &bore[0] {
									wall = math.Min(wall, fitNearest(p, [][]r3.Vec{loop}))
								}
							}
						}
						require.GreaterOrEqual(t, wall, cornerMinWall, "bore wall at z %.3f", z)
					}
				})
			}
		})
	}
}
