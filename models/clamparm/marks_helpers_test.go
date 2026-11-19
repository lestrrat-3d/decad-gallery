package clamparm_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
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
