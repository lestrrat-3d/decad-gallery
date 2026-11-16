package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// cornerMinWall is the press-fit wall of "Corners and roots": four 0.4 mm lines.
const cornerMinWall = 1.6

// TestOneWayClutchCorners applies "Corners and roots" to the four printed parts, each in its print orientation (axis
// vertical, so +Z in the clutch's frame), in every preset and both directions. The one exemption is the back plate's
// hex pocket corners: 18 vertical 60 degree corners that match the M2 nut's sharp corners, read in the section run
// z 0 to 2.0 and exempt about each site within NutCavityRadius plus or minus 0.3. The ring's three mount nut pockets
// are the same corners, 18 more, read in the section run z -1.25 to 0.75, and are exempt the same way about each mount
// site. The adapter ring is checked in TestOneWayClutchAdapter, with no exemption. Every other corner listed under
// "Retained sharp corners" in instructions.md passes the rule's limits (the D bore's corners, the head tip and lock
// face, the ring's tooth tips and roots) or lies between layers (the head pocket and hex pocket rims), where no
// section reads it. The hub, back plate and cover are press-fit rings, so
// each section's bore keeps a wall of 1.6 mm or more to every other outline.
func TestOneWayClutchCorners(t *testing.T) {
	axis := r3.Vec{Z: 1}
	for _, tc := range dirCases() {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := buildCase(t, tc.params)
			for _, part := range []struct {
				name  string
				bored bool
			}{{"hub", true}, {"ring", false}, {"back plate", true}, {"cover", true}} {
				t.Run(part.name, func(t *testing.T) {
					body := partsOf(m)[part.name]
					var exempt []cornercheck.Exempt
					// The back plate's pockets are at the three plate bolts, the ring's at the three mount bolts.
					var sites []m2fastener.Site
					switch part.name {
					case "back plate":
						sites = m.Sites[:]
					case "ring":
						sites = m.Dimensions.MountSites[:]
					}
					for _, site := range sites {
						exempt = append(exempt, cornercheck.Exempt{
							Reason: "hex nut pocket corners match the nut's sharp corners",
							Near: cornercheck.Annulus(r3.Vec{X: site.Head.X, Y: site.Head.Y}, axis,
								m2fastener.NutCavityRadius-0.3, m2fastener.NutCavityRadius+0.3),
						})
					}
					fs, err := cornercheck.Check(t.Context(), body, axis, exempt)
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
