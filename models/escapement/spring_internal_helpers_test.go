package escapement

import (
	"math"

	"github.com/lestrrat-3d/decad"
)

// springGap is the least distance from the spring's outline vertices that lie between rMin and rMax from centre to the
// polygons other. A mainspring's outline is one long loop whose Clearance bound is wide, so fits are measured from its
// vertices; the outline's own bound is added by the callers' tolerances.
func springGap(so decad.SectionOutline, centre Pt, rMin, rMax float64, other [][]Pt) float64 {
	best := math.Inf(1)
	for _, l := range loopsBy(so) {
		for _, v := range l {
			if d := v.Sub(centre).Len(); d < rMin || d > rMax {
				continue
			}
			for _, o := range other {
				for j := range o {
					best = math.Min(best, distPointSeg(v, o[j], o[(j+1)%len(o)]))
				}
			}
		}
	}
	return best
}
