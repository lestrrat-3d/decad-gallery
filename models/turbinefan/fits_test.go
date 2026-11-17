package turbinefan_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/stretchr/testify/require"
)

// regularMinimum is the Regular fit class minimum of docs/model-pipeline.md "Part clearances". The test does not read
// the model's nominal value: it proves the placed parts meet this bound.
const regularMinimum = 0.35

// TestTurbineFanFits checks every Regular fit between the hub and a blade: the stud in its socket, in the hub's mid plane
// through the stud's axis; the stud's inner end to the socket floor and the blade's root to the hub's rim, in the plane
// through the shaft axis and the blade's radial line; and every blade to the hub in layers along the shaft. The bore's press fit is a design value, which TestTurbineFanBoreFit measures.
func TestTurbineFanFits(t *testing.T) {
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		for k := 1; k <= turbinefan.BladeCount; k++ {
			stud := clearance(t, m.Hub, m.Blades[k-1], atZ(t, sz.HubMid))
			require.GreaterOrEqual(t, stud.Lower().Base(), regularMinimum, "blade %d: stud in its socket", k)
			require.InDelta(t, turbinefan.RegularFit, stud.Distance.Value.Base(), 0.01, "blade %d: nominal stud fit", k)
			through := clearance(t, m.Hub, m.Blades[k-1], radial(t, k))
			require.GreaterOrEqual(t, through.Lower().Base(), regularMinimum, "blade %d: stud end and root", k)
		}
		for k := 1; k <= turbinefan.BladeCount; k++ {
			for _, z := range []float64{0.3 * sz.HubLength, 0.5 * sz.HubLength, 0.7 * sz.HubLength} {
				cl := clearance(t, m.Hub, m.Blades[k-1], atZ(t, z))
				require.GreaterOrEqual(t, cl.Lower().Base(), regularMinimum, "blade %d to the hub at z %g", k, z)
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
