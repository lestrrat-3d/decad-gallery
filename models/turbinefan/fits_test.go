package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// regularMinimum is the Regular fit class minimum of docs/model-pipeline.md "Part clearances". The test does not read
// the model's nominal value: it proves the placed parts meet this bound.
const regularMinimum = 0.35

// TestTurbineFanFits checks every Regular fit between the hub and a blade: the stud in its socket, in the plane z = 10
// through the stud's axis; the stud's inner end to the socket floor and the blade's root to the hub's rim, in the plane
// through the shaft axis and the blade's radial line; and blades 2 and 3 to the tab's roots and sides, in layers along
// the shaft. The bore's press fit is a design value, which TestTurbineFanBoreFit measures.
func TestTurbineFanFits(t *testing.T) {
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		for k := 1; k <= turbinefan.BladeCount; k++ {
			stud := clearance(t, m.Hub, m.Blades[k-1], atZ(t, turbinefan.HubMid))
			require.GreaterOrEqual(t, stud.Lower().Base(), regularMinimum, "blade %d: stud in its socket", k)
			require.InDelta(t, turbinefan.RegularFit, stud.Distance.Value.Base(), 0.01, "blade %d: nominal stud fit", k)
			through := clearance(t, m.Hub, m.Blades[k-1], radial(t, k))
			require.GreaterOrEqual(t, through.Lower().Base(), regularMinimum, "blade %d: stud end and root", k)
		}
		for _, k := range []int{2, 3} {
			for _, z := range []float64{6, 8, 10, 12, 14} {
				cl := clearance(t, m.Hub, m.Blades[k-1], atZ(t, z))
				require.GreaterOrEqual(t, cl.Lower().Base(), regularMinimum, "blade %d to the tab at z %g", k, z)
				_ = cl
				// The least distance from the blade's section to the tab's edges, those below the roots.
				hub, blade := section(t, m.Hub, atZ(t, z)), section(t, m.Blades[k-1], atZ(t, z))
				bound := hub.OffSurface.Base() + blade.OffSurface.Base()
				least := math.Inf(1)
				for _, hl := range hub.Loops {
					for i := range hl {
						a, b := hl[i], hl[(i+1)%len(hl)]
						if a.Y > m.Dimensions.RootSide || b.Y > m.Dimensions.RootSide {
							continue
						}
						for _, bl := range blade.Loops {
							for _, v := range bl {
								least = math.Min(least, distanceToLoop(v, []r3.Vec{a, b}))
							}
						}
					}
				}
				require.GreaterOrEqual(t, least-bound, regularMinimum, "blade %d to the tab's sides and tail at z %g", k, z)
				if z == 10 {
					t.Logf("%s blade %d to the tab at z 10: %.3f", c.name, k, least)
				}
			}
		}
	})
}

// TestTurbineFanNoInterference asks decad to verify the finished bodies: no pair of the hub and the blades shares
// material or stays undecided.
func TestTurbineFanNoInterference(t *testing.T) {
	m, doc := built(t, caseNamed(t, "605ZZ"))
	names := map[*decad.Body]string{m.Hub: "hub"}
	for k, b := range m.Blades {
		names[b] = fmt.Sprintf("blade %d", k+1)
	}
	report, err := doc.Verify(t.Context())
	require.NoError(t, err)
	for _, pair := range report.Interferences {
		a, aok := names[pair.A]
		b, bok := names[pair.B]
		require.False(t, aok && bok, "%s and %s share material", a, b)
	}
	for _, d := range report.Diagnostics {
		if d.Pair == nil {
			continue
		}
		a, aok := names[d.Pair.A]
		b, bok := names[d.Pair.B]
		require.False(t, aok && bok, "%s and %s are undecided: %s", a, b, d.Code)
	}
}
