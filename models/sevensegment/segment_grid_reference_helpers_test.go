package sevensegment

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// nearestWithinWalk is the reference for nearestWithin: it walks every cell inside each search ring and skips the
// inner ones, which costs O(r^2) per ring. It is kept only to check the ring-edge walk against.
func (g *segmentGrid) nearestWithinWalk(p r3.Vec, limit float64) float64 {
	if len(g.segs) == 0 {
		return math.Inf(1)
	}
	dx := math.Max(0, math.Max(g.lo.X-p.X, p.X-g.hi.X))
	dy := math.Max(0, math.Max(g.lo.Y-p.Y, p.Y-g.hi.Y))
	if math.Hypot(dx, dy) > limit+gridMargin {
		return math.Inf(1)
	}
	best := math.Inf(1)
	ci, cj := g.cellOf(p)
	for r := 0; ; r++ {
		i0, i1, j0, j1 := ci-r, ci+r, cj-r, cj+r
		for j := max(j0, 0); j <= min(j1, g.ny-1); j++ {
			for i := max(i0, 0); i <= min(i1, g.nx-1); i++ {
				if j != j0 && j != j1 && i != i0 && i != i1 {
					continue
				}
				for _, k := range g.cells[j*g.nx+i] {
					best = math.Min(best, segDist(p, g.segs[k][0], g.segs[k][1]))
				}
			}
		}
		if i0 <= 0 && j0 <= 0 && i1 >= g.nx-1 && j1 >= g.ny-1 {
			return best
		}
		bound := math.Inf(1)
		if i0 > 0 {
			bound = math.Min(bound, p.X-(g.lo.X+float64(i0)*g.cell))
		}
		if i1 < g.nx-1 {
			bound = math.Min(bound, g.lo.X+float64(i1+1)*g.cell-p.X)
		}
		if j0 > 0 {
			bound = math.Min(bound, p.Y-(g.lo.Y+float64(j0)*g.cell))
		}
		if j1 < g.ny-1 {
			bound = math.Min(bound, g.lo.Y+float64(j1+1)*g.cell-p.Y)
		}
		if bound > math.Min(best, limit)+gridMargin {
			return best
		}
	}
}
