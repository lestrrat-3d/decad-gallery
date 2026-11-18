package clamparm_test

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/stretchr/testify/require"
)

// TestClampArmCapacity pins the geometric inputs of "Joint capacity" in instructions.md: the mean friction radius r_m
// of the annulus a knuckle face rubs on, the two faces, and the fin's size, which sets F_fin, and then the T_hold table
// that follows from them.
func TestClampArmCapacity(t *testing.T) {
	// The friction face is an annulus from the bolt hole (radius 2.8) to the end radius (10).
	rOut, rIn := clamparm.EndRadius, m5fastener.HoleRadius
	rm := 2.0 / 3.0 * (math.Pow(rOut, 3) - math.Pow(rIn, 3)) / (math.Pow(rOut, 2) - math.Pow(rIn, 2))
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
	const youngs = 3500.0 // N/mm^2
	second := clamparm.Section * math.Pow(clamparm.FinThickness, 3) / 12
	fFin := 3 * youngs * second * clamparm.JointFit / math.Pow(clamparm.SlotDepth, 3)
	require.InDelta(t, 49, fFin, 1, "F_fin")

	// The table: T_hold = 2 mu (F - F_fin) r_m, F = T_k / (0.2 d), d = 5.
	text, err := os.ReadFile("instructions.md")
	require.NoError(t, err)
	row := regexp.MustCompile(`(?m)^\| (Low|Nominal|High) \| ([0-9.]+) \| ([0-9.]+) \| ([0-9.]+) \| ([0-9.]+) \|$`)
	rows := row.FindAllStringSubmatch(string(text), -1)
	require.Len(t, rows, 3, "the T_hold table")
	num := func(s string) float64 {
		v, err := strconv.ParseFloat(s, 64)
		require.NoError(t, err)
		return v
	}
	for _, r := range rows {
		mu, tk, f, hold := num(r[2]), num(r[3]), num(r[4]), num(r[5])
		require.InDelta(t, tk/(0.2*5)*1000, f, 1e-9, "%s: F", r[1])
		require.InDelta(t, hold, 2*mu*(f-fFin)*rm/1000, 0.01, "%s: T_hold", r[1])
	}
}
