package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/stretchr/testify/require"
)

// TestPoseNoOverlap poses every body at several phases and requires that no two overlap: the wheel and the stones, and
// the pin and the lever, by section; every other pair by mesh.
func TestPoseNoOverlap(t *testing.T) {
	l := localParts(t)
	g := &l.g
	slots := l.m.slots()
	for _, phase := range []float64{0, 0.1, 0.226, 0.24, 0.25, 0.27, 0.6, 0.75} {
		s, err := g.StateAt(phase)
		require.NoError(t, err)
		table, err := poses(g, s)
		require.NoError(t, err)
		meshes := make([]meshgap.Mesh, len(slots))
		for i, sl := range slots {
			meshes[i] = l.posedMesh(t, sl.name, table[sl.name])
		}
		for i := range slots {
			for j := i + 1; j < len(slots); j++ {
				a, b := slots[i].name, slots[j].name
				if byDesign(a, b) || byOutline(a, b) {
					continue
				}
				// The lever's arm rests on a banking post at the banking, a stated contact 0.05° short of touching:
				// a gap under what a mesh proves, read by section below.
				if pairIs(a, b, LeverName, BackPlateName) && math.Abs(math.Abs(s.LeverAngle)-g.Banking) < nearBanking {
					continue
				}
				gap, separated, err := meshes[i].Exceeds(meshes[j], 0)
				require.NoError(t, err)
				require.True(t, separated, "phase %v: %s and %s are not proven apart (gap lower bound %v)", phase, a, b, gap)
			}
		}

		// The hairspring is drawn relaxed on the staff. Its hub bore clears the staff's flat corners only while the
		// balance stays within poseHairspringReach of rest; its faces clear the balance wheel and the retainer by height.
		spring, _ := l.posedLoops(t, HairspringName, 21, table[HairspringName])
		if math.Abs(s.BalanceAngle) <= poseHairspringReach {
			staff, off2 := l.posedLoops(t, BalanceStaffName, 21, table[BalanceStaffName])
			gap := springGap(spring, g.B, 0, 4, staff)
			require.GreaterOrEqual(t, gap, tightFit-0.02-off2, "phase %v: hairspring hub bore to the staff", phase)
		}
		require.GreaterOrEqual(t, springLow-balanceHigh, 0.25-3e-3, "phase %v: hairspring to the balance wheel", phase)
		require.GreaterOrEqual(t, retainerLow-(springLow+SpringParams().Height), 0.25-3e-3,
			"phase %v: hairspring to the retainer", phase)

		// The wheel's teeth and the stones, in the wheel's plane: the stones may touch a tooth, never enter it.
		wheel, off1 := l.posedLoops(t, EscapeWheelName, midHeight, table[EscapeWheelName])
		for _, name := range []string{EntryStoneName, ExitStoneName} {
			stone, off2 := l.posedLoops(t, name, midHeight, table[name])
			bound := 0.02 + off1 + off2
			require.LessOrEqual(t, loopsDepth(stone, wheel), bound, "phase %v: %s in the escape wheel", phase, name)
			require.LessOrEqual(t, loopsDepth(wheel, stone), bound, "phase %v: escape wheel in the %s", phase, name)
		}
		if math.Abs(math.Abs(s.LeverAngle)-g.Banking) < nearBanking {
			lever, off1 := l.posedLoops(t, LeverName, forkBandF, table[LeverName])
			posts, off2 := l.posedLoops(t, BackPlateName, forkBandF, table[BackPlateName])
			require.LessOrEqual(t, loopsDepth(lever, posts), 0.02+off1+off2, "phase %v: lever in a banking post", phase)
			require.LessOrEqual(t, loopsDepth(posts, lever), 0.02+off1+off2, "phase %v: banking post in the lever", phase)
		}
		// The impulse pin and the lever's fork, at the lever's band F: the pin may touch a corner round, never enter it.
		pin, off1 := l.posedLoops(t, RollerTableName, forkBandF, table[RollerTableName])
		lever, off2 := l.posedLoops(t, LeverName, forkBandF, table[LeverName])
		bound := 0.02 + off1 + off2
		require.LessOrEqual(t, loopsDepth(pin, lever), bound, "phase %v: pin in the lever", phase)
		require.LessOrEqual(t, loopsDepth(lever, pin), bound, "phase %v: lever in the pin", phase)
	}
}
