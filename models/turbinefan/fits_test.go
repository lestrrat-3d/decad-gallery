package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanFits checks every Regular hub-to-blade fit: the stud in the socket's centre plane, the stud end and
// blade root in the radial plane, and each blade at three heights along the shaft. TestTurbineFanBoreFit measures the
// bore's press fit.
func TestTurbineFanFits(t *testing.T) {
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		hubMesh := clutchGapMesh(t, m.Hub, 0.05)
		for k := 1; k <= turbinefan.BladeCount; k++ {
			bladeMesh := clutchGapMesh(t, m.Blades[k-1], 0.05)
			clear, err := hubMesh.ExceedsClearance(bladeMesh, regularMinimum)
			require.NoError(t, err)
			require.True(t, clear, "blade %d: whole-surface hub clearance", k)
			stud := clearance(t, m.Hub, m.Blades[k-1], atZ(t, sz.BladeSocketZ()))
			require.GreaterOrEqual(t, stud.Lower().Base(), regularMinimum, "blade %d: stud in its socket", k)
			require.InDelta(t, turbinefan.RegularFit, stud.Distance.Value.Base(), 0.01, "blade %d: nominal stud fit", k)
			through := clearance(t, m.Hub, m.Blades[k-1], radial(t, k))
			require.GreaterOrEqual(t, through.Lower().Base(), regularMinimum, "blade %d: stud end and root", k)
		}
		for k := 1; k <= turbinefan.BladeCount; k++ {
			for _, z := range []float64{0.3 * sz.HubLength, 0.5 * sz.HubLength, 0.7 * sz.HubLength} {
				plane := atZ(t, z)
				hub, blade := section(t, m.Hub, plane), section(t, m.Blades[k-1], plane)
				require.NotEmpty(t, hub.Loops, "hub at z %g", z)
				if len(blade.Loops) == 0 {
					box, err := m.Blades[k-1].Bounds()
					require.NoError(t, err)
					bound := box.Bound.Base()
					require.True(t, z < box.Min.Z-bound || z > box.Max.Z+bound,
						"blade %d has no section at z %g but its bounds cross that plane", k, z)
					continue
				}
				cl, err := hub.Clearance(blade)
				require.NoError(t, err)
				require.False(t, cl.Overlaps, "blade %d overlaps the hub at z %g", k, z)
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
