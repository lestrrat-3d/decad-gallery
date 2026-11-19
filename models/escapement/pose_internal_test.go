package escapement

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/stretchr/testify/require"
)

func TestPoseNoOverlap(t *testing.T) {
	l := localParts(t)
	g := &l.g
	for _, phase := range []float64{0, 0.1, 0.35, 0.6, 0.85} {
		s, err := g.StateAt(phase)
		require.NoError(t, err)
		table := poses(g, s)
		slots := l.m.slots()
		meshes := make([]meshgap.Mesh, len(slots))
		for i, sl := range slots {
			meshes[i] = l.posedMesh(t, sl.name, table[sl.name])
		}
		for i := range slots {
			for j := i + 1; j < len(slots); j++ {
				a, b := slots[i].name, slots[j].name
				if touching(a, b) || byOutline(a, b) {
					continue
				}
				gap, separated, err := meshes[i].Exceeds(meshes[j], 0)
				require.NoError(t, err)
				require.True(t, separated, "phase %v: %s and %s are not proven apart (gap lower bound %v)", phase, a, b, gap)
			}
		}
		// Pairs checked by outline.
		wheel := l.posedLoops(t, EscapeWheelName, 23.5, table[EscapeWheelName])
		anchor := l.posedLoops(t, AnchorName, 23.5, table[AnchorName])
		require.False(t, overlap(wheel[0], anchor[0]), "phase %v: escape wheel and anchor", phase)
		pinion := l.posedLoops(t, EscapeWheelName, 3, table[EscapeWheelName])
		great := l.posedLoops(t, GreatWheelName, 3, table[GreatWheelName])
		require.False(t, overlap(pinion[0], great[0]), "phase %v: pinion and great wheel", phase)
		pocket := l.posedLoops(t, GreatWheelName, 7, table[GreatWheelName])[1]
		drum := l.posedLoops(t, PawlCarrierName, 7, table[PawlCarrierName])[0]
		require.True(t, fitsIn(drum, pocket), "phase %v: the carrier's pawls stay inside the ratchet pocket", phase)
		// The arbor turns inside the fixed spring hub's bore without touching it.
		arbor := l.posedLoops(t, GreatWheelArborName, 14.25, table[GreatWheelArborName])[0]
		var bore []Pt
		for _, lp := range l.posedLoops(t, MainspringName, 14.25, table[MainspringName]) {
			near := true
			for _, v := range lp {
				if v.Sub(g.G).Len() > 4 {
					near = false
					break
				}
			}
			if near {
				bore = lp
			}
		}
		require.NotNil(t, bore, "the spring hub's bore")
		require.True(t, fitsIn(arbor, bore), "phase %v: the arbor turns inside the spring's bore", phase)
		// The arbor turns 3.25° across a period against the fixed spring, so the flat corners close the 0.30 mm of the build
		// pose by about 0.084 mm at the worst phase, and the 0.30 mm bore clearance keeps the Tight 0.2 mm minimum.
		require.GreaterOrEqual(t, distPolys(arbor, bore), 0.2, "phase %v: arbor to spring bore", phase)
	}
}
