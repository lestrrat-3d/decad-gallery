package clamparm_test

// CI: isolated-tests

import (
	"math"
	"os"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/clamparmcapacity"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/stretchr/testify/require"
)

// TestClampArmCapacity pins the geometric inputs of "Joint capacity" in instructions.md: the mean friction radius r_m
// of the annulus a knuckle face rubs on, the two faces, and the fin's size, which sets F_fin, and then the T_hold table
// that follows from them.
func TestClampArmCapacity(t *testing.T) {
	// The friction face is an annulus from the bolt hole (radius 2.8) to the end radius (10).
	rm, fFin := clamparmcapacity.Inputs()
	require.InDelta(t, 7.075, rm, 0.001, "r_m")

	// The tongue's two faces rub against two fins: a link's fork end has two fin loops at a section across the slot.
	_, m := asmBuild(t, asmCase{"default", clamparm.DefaultParams(), -1})
	link := m.Links[0]
	o := asmSectionOutline(t, link.Body, sectionFrame(t, link, 0, 110))
	require.Len(t, o.Loops, 2, "two fins, so two friction faces on the tongue")
	var yLo, yHi [2]float64
	for i, l := range o.Loops {
		yLo[i], yHi[i] = math.Inf(1), math.Inf(-1)
		for _, v := range l {
			y := link.Frame.ToLocal(v).Y
			yLo[i], yHi[i] = math.Min(yLo[i], y), math.Max(yHi[i], y)
		}
	}
	bound := o.OffSurface.Base() + 1e-3
	for i := range yLo {
		require.InDelta(t, clamparm.FinThickness, yHi[i]-yLo[i], bound, "fin %d thickness", i)
	}
	require.InDelta(t, clamparm.ForkGap, math.Max(yLo[0], yLo[1])-math.Min(yHi[0], yHi[1]), bound, "fork gap")
	require.InDelta(t, clamparm.ForkGap, clamparm.TongueThickness+2*clamparm.JointFit, 1e-9)
	require.InDelta(t, clamparm.EndRadius-clamparm.ForkGap/2, clamparm.FinThickness, 1e-9)

	// F_fin: a fin is a cantilever SlotDepth long, Section wide and FinThickness thick, closed by JointFit.
	require.InDelta(t, 49, fFin, 1, "F_fin")

	// The table: T_hold = 2 mu (F - F_fin) r_m, F = T_k / (0.2 d), d = 5.
	text, err := os.ReadFile("instructions.md")
	require.NoError(t, err)
	require.NoError(t, clamparmcapacity.Check(string(text)))
}
