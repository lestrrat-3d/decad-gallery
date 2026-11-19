package onewayclutch_test

// CI: isolated-tests

import (
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestOneWayClutchEnds checks that the ring's two ends stand EndStandoff past every body that turns with the shaft, so a
// flat container face at either end of the clutch touches only the ring, and that the ring's axial play is less than
// the standoff by at least a Regular gap (0.35 mm): the ring, moved a little past its play toward either plate, meets
// that plate, and moved a little short of it, does not.
func TestOneWayClutchEnds(t *testing.T) {
	const regularMin = onewayclutch.RegularFit - onewayclutch.FitMargin
	for _, c := range fitCases() {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			ring, err := m.Ring.Bounds()
			require.NoError(t, err)
			tol := ring.Bound.Base() + 1e-3
			require.InDelta(t, onewayclutch.RingFrom, ring.Min.Z, tol, "ring's back end")
			require.InDelta(t, onewayclutch.RingTo, ring.Max.Z, tol, "ring's front end")

			shaftSide := map[string]*decad.Body{
				"hub": m.Hub, "back plate": m.BackPlate, "cover": m.Cover,
			}
			for i := range m.Bolts {
				shaftSide[fmt.Sprintf("bolt %d", i+1)] = m.Bolts[i].Body
				shaftSide[fmt.Sprintf("nut %d", i+1)] = m.Nuts[i].Body
			}
			for name, body := range shaftSide {
				b, err := body.Bounds()
				require.NoError(t, err, name)
				btol := b.Bound.Base() + 1e-3
				require.GreaterOrEqual(t, b.Min.Z, ring.Min.Z+onewayclutch.EndStandoff-btol, "%s clear of the ring's back end", name)
				require.LessOrEqual(t, b.Max.Z, ring.Max.Z-onewayclutch.EndStandoff+btol, "%s clear of the ring's front end", name)
			}

			// The standoff exceeds the ring's play by at least a Regular gap.
			require.GreaterOrEqual(t, onewayclutch.EndStandoff-onewayclutch.ToothEndGap, regularMin)

			plateMesh := map[string]meshgap.Mesh{
				"back plate": fitMeshOf(t, m.BackPlate, fitMeshTol),
				"cover":      fitMeshOf(t, m.Cover, fitMeshTol),
			}
			ringMesh, err := m.Ring.Tessellate(t.Context(), units.Millimeters(fitMeshTol), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			for _, tc := range []struct {
				plate string
				dir   float64
			}{{"back plate", -1}, {"cover", 1}} {
				// Short of the play: the ring still clears the plate. Past it: the ring meets the plate.
				for _, step := range []struct {
					move  float64
					clear bool
				}{{onewayclutch.ToothEndGap - 0.1, true}, {onewayclutch.ToothEndGap + 0.1, false}} {
					// The ring's mesh, shifted along the axis; a decad transform would retire the ring from its document.
					verts := ringMesh.Vertices()
					for k := range verts {
						verts[k].Z += tc.dir * step.move
					}
					moved, err := meshgap.New(verts, ringMesh.Triangles(), ringMesh.Bound().Base())
					require.NoError(t, err)
					gap, clear, err := moved.Exceeds(plateMesh[tc.plate], 0)
					require.NoError(t, err)
					require.Equal(t, step.clear, clear, "ring moved %.2f toward the %s (gap bound %g)", step.move, tc.plate, gap)
				}
			}
		})
	}
}
