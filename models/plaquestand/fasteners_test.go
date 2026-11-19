package plaquestand_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/stretchr/testify/require"
)

// TestPlaqueStandFasteners checks each bolt and nut against the stack in instructions.md "Fasteners": the head ends
// flush with the rail top and the tip, clamped, flush with the bracket's bottom face; the nut sits inside its pocket;
// no fastener overlaps the part it passes through; and every wall round a pocket keeps 1 mm.
func TestPlaqueStandFasteners(t *testing.T) {
	const tb, j = plaquestand.BaseThickness, plaquestand.JointFit
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			for k, site := range m.Dimensions.Sites {
				bolt, nut := m.Bolts[k].Body, m.Nuts[k].Body
				bb, err := bolt.Bounds()
				require.NoError(t, err)
				tol := bb.Bound.Base() + 2e-3
				require.InDelta(t, tb+j, bb.Max.Z, tol, "bolt %d head is flush with the rail top", k+1)
				require.InDelta(t, j, bb.Min.Z, tol, "bolt %d tip stands inside the pocket", k+1)
				require.LessOrEqual(t, math.Max(bb.Max.X-site.Head.X, bb.Max.Y-site.Head.Y), m2fastener.HeadPocketRadius+tol)
				require.LessOrEqual(t, math.Max(site.Head.X-bb.Min.X, site.Head.Y-bb.Min.Y), m2fastener.HeadPocketRadius+tol)
				// Clamped, the joint drops by j: the tip then ends flush with the bracket's bottom face, never below it.
				require.InDelta(t, 0, site.Head.Z-j-site.Tip(), 1e-9)

				nb, err := nut.Bounds()
				require.NoError(t, err)
				ntol := nb.Bound.Base() + 2e-3
				require.InDelta(t, 0.2, nb.Min.Z, ntol)
				require.InDelta(t, 1.8, nb.Max.Z, ntol)
				require.LessOrEqual(t, nb.Max.X-site.Head.X, m2fastener.NutCavityRadius+ntol)
				require.LessOrEqual(t, site.Head.X-nb.Min.X, m2fastener.NutCavityRadius+ntol)
				require.LessOrEqual(t, nb.Max.Y-site.Head.Y, m2fastener.NutCavityRadius+ntol)
				require.LessOrEqual(t, site.Head.Y-nb.Min.Y, m2fastener.NutCavityRadius+ntol)
				require.InDelta(t, site.Head.X, (nb.Min.X+nb.Max.X)/2, ntol)
				require.InDelta(t, site.Head.Y, (nb.Min.Y+nb.Max.Y)/2, ntol)

				// No fastener overlaps the part it passes through.
				for _, z := range []float64{1, 3, 5.4, 7.4} {
					for hostName, host := range map[string]*decad.Body{"bracket": m.Brackets[k], "rail": m.Rail} {
						hs := section(t, host, z)
						for partName, part := range map[string]*decad.Body{"bolt": bolt, "nut": nut} {
							ps := section(t, part, z)
							if len(hs.Loops) == 0 || len(ps.Loops) == 0 {
								continue
							}
							cl, err := hs.Clearance(ps)
							require.NoError(t, err)
							require.False(t, cl.Overlaps, "%s %d overlaps the %s at z %g", partName, k+1, hostName, z)
						}
					}
				}
			}
		})
	}

	t.Run("walls", func(t *testing.T) {
		const w, h = plaquestand.RailWidth, plaquestand.BracketWidth
		for name, v := range map[string]float64{
			"rail under the head pocket":       tb + j - m2fastener.HeadPocketDepth - (tb - plaquestand.RailHeight + j),
			"rail round the head pocket":       w/2 - m2fastener.HeadPocketRadius,
			"rail end to the head pocket":      h/2 - m2fastener.HeadPocketRadius,
			"bracket face to the nut pocket":   h/2 - m2fastener.NutCavityRadius,
			"bracket between pocket and floor": (tb - plaquestand.RailHeight) - m2fastener.NutCavityHeight,
		} {
			require.GreaterOrEqual(t, v, 1.0, name)
		}
	})
}
