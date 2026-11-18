package clamparm_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/stretchr/testify/require"
)

// markHoles returns the centres, in the part's local frame, of the loops of the section at local z that are a mark
// hole: a loop whose bounds are about the mark's diameter square.
func markHoles(t *testing.T, p clamparm.Part, z float64) [][2]float64 {
	t.Helper()
	o := asmSectionOutline(t, p.Body, sectionFrame(t, p, 2, z))
	var out [][2]float64
	for _, l := range o.Loops {
		lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
		for _, v := range l {
			lv := p.Frame.ToLocal(v)
			lo[0], hi[0] = math.Min(lo[0], lv.X), math.Max(hi[0], lv.X)
			lo[1], hi[1] = math.Min(lo[1], lv.Y), math.Max(hi[1], lv.Y)
		}
		if math.Abs(hi[0]-lo[0]-1.5) < 0.1 && math.Abs(hi[1]-lo[1]-1.5) < 0.1 {
			out = append(out, [2]float64{(lo[0] + hi[0]) / 2, (lo[1] + hi[1]) / 2})
		}
	}
	return out
}

// TestClampArmMarks checks the look-alike sets: a knob has 1 mark hole and a screw knob 2, each 1.5 across on radius
// MarkRadius at 30 degrees and then 55, and each runs through: it shows at every level from the inner face to the outer
// face, including the levels beside the hex pocket.
func TestClampArmMarks(t *testing.T) {
	_, m := asmBuild(t, asmCase{"default", clamparm.DefaultParams(), -1})
	sets := []struct {
		name   string
		part   clamparm.Part
		height float64
		angles []float64
	}{
		{"knob J1", m.Knobs[0], clamparm.KnobHeight, []float64{30}},
		{"knob J2", m.Knobs[1], clamparm.KnobHeight, []float64{30}},
		{"knob J3", m.Knobs[2], clamparm.KnobHeight, []float64{30}},
		{"knob J4", m.Knobs[3], clamparm.KnobHeight, []float64{30}},
		{"screw knob", m.ScrewKnobs[0], clamparm.ScrewKnobHeight, []float64{30, 55}},
		{"stud knob", m.ScrewKnobs[1], clamparm.ScrewKnobHeight, []float64{30, 55}},
	}
	for _, s := range sets {
		t.Run(s.name, func(t *testing.T) {
			for _, z := range []float64{0.3, s.height / 2, s.height - 0.3} {
				holes := markHoles(t, s.part, z)
				require.Len(t, holes, len(s.angles), "z %g", z)
				for i, h := range holes {
					found := false
					for _, a := range s.angles {
						rad := a * math.Pi / 180
						if math.Hypot(h[0]-clamparm.MarkRadius*math.Cos(rad), h[1]-clamparm.MarkRadius*math.Sin(rad)) < 0.1 {
							found = true
						}
					}
					require.True(t, found, "z %g: hole %d at (%.2f, %.2f)", z, i, h[0], h[1])
				}
			}
		})
	}
	require.Less(t, len(markHoles(t, m.Knobs[0], 6)), len(markHoles(t, m.ScrewKnobs[0], 5)))
}
