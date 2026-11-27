package sevensegment

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// segmentGrid holds the segments of section loops (in x and y) in a uniform grid, so that the distance from a point to
// the nearest segment is found among the cells around the point instead of over every segment. It returns exactly
// what nearest returns: it skips only segments that lie farther away than a distance it has already found, and a
// minimum does not depend on the order it is taken in.
type segmentGrid struct {
	segs   [][2]r3.Vec
	lo, hi r3.Vec // the box of every segment
	cell   float64
	nx, ny int
	cells  [][]int32
}

// gridMargin keeps the cell bounds clear of rounding: a cell is skipped only when it lies this much farther than the
// best distance found.
const gridMargin = 1e-9

func newSegmentGrid(loops [][]r3.Vec) *segmentGrid {
	g := &segmentGrid{lo: r3.Vec{X: math.Inf(1), Y: math.Inf(1)}, hi: r3.Vec{X: math.Inf(-1), Y: math.Inf(-1)}}
	for _, l := range loops {
		for i, a := range l {
			b := l[(i+1)%len(l)]
			g.segs = append(g.segs, [2]r3.Vec{a, b})
			g.lo = r3.Vec{X: math.Min(g.lo.X, math.Min(a.X, b.X)), Y: math.Min(g.lo.Y, math.Min(a.Y, b.Y))}
			g.hi = r3.Vec{X: math.Max(g.hi.X, math.Max(a.X, b.X)), Y: math.Max(g.hi.Y, math.Max(a.Y, b.Y))}
		}
	}
	if len(g.segs) == 0 {
		return g
	}
	// About one segment per cell along the loops, and no more than 256 cells along a side.
	ext := math.Max(g.hi.X-g.lo.X, g.hi.Y-g.lo.Y)
	g.cell = math.Max(ext/256, 1e-3)
	g.nx = int((g.hi.X-g.lo.X)/g.cell) + 1
	g.ny = int((g.hi.Y-g.lo.Y)/g.cell) + 1
	g.cells = make([][]int32, g.nx*g.ny)
	for k, s := range g.segs {
		i0, j0 := g.cellOf(r3.Vec{X: math.Min(s[0].X, s[1].X), Y: math.Min(s[0].Y, s[1].Y)})
		i1, j1 := g.cellOf(r3.Vec{X: math.Max(s[0].X, s[1].X), Y: math.Max(s[0].Y, s[1].Y)})
		for j := j0; j <= j1; j++ {
			for i := i0; i <= i1; i++ {
				g.cells[j*g.nx+i] = append(g.cells[j*g.nx+i], int32(k))
			}
		}
	}
	return g
}

// cellOf returns the cell of p, clamped to the grid.
func (g *segmentGrid) cellOf(p r3.Vec) (int, int) {
	i := int(math.Floor((p.X - g.lo.X) / g.cell))
	j := int(math.Floor((p.Y - g.lo.Y) / g.cell))
	return min(max(i, 0), g.nx-1), min(max(j, 0), g.ny-1)
}

// ringDist returns the distance from p to the nearest segment in the cells on the edge of the block i0..i1, j0..j1,
// clipped to the grid. Cells inside the block were visited at a smaller ring, so only the two end rows and, in the rows
// between, the two end columns are walked: a ring costs O(r), not O(r^2).
func (g *segmentGrid) ringDist(p r3.Vec, i0, i1, j0, j1 int) float64 {
	best := math.Inf(1)
	cell := func(i, j int) {
		for _, k := range g.cells[j*g.nx+i] {
			best = math.Min(best, segDist(p, g.segs[k][0], g.segs[k][1]))
		}
	}
	ia, ib := max(i0, 0), min(i1, g.nx-1)
	row := func(j int) {
		if j < 0 || j >= g.ny {
			return
		}
		for i := ia; i <= ib; i++ {
			cell(i, j)
		}
	}
	row(j0)
	if j1 != j0 {
		row(j1)
	}
	for j := max(j0+1, 0); j <= min(j1-1, g.ny-1); j++ {
		if i0 >= 0 && i0 < g.nx {
			cell(i0, j)
		}
		if i1 != i0 && i1 >= 0 && i1 < g.nx {
			cell(i1, j)
		}
	}
	return best
}

// nearestWithin returns the distance from p to the nearest segment, as nearest does, when it is at most limit; when
// every segment lies farther than limit it may return any value greater than limit.
func (g *segmentGrid) nearestWithin(p r3.Vec, limit float64) float64 {
	if len(g.segs) == 0 {
		return math.Inf(1)
	}
	// The box of every segment: a point farther than limit from it is farther than limit from every segment.
	dx := math.Max(0, math.Max(g.lo.X-p.X, p.X-g.hi.X))
	dy := math.Max(0, math.Max(g.lo.Y-p.Y, p.Y-g.hi.Y))
	if math.Hypot(dx, dy) > limit+gridMargin {
		return math.Inf(1)
	}
	best := math.Inf(1)
	ci, cj := g.cellOf(p)
	for r := 0; ; r++ {
		i0, i1, j0, j1 := ci-r, ci+r, cj-r, cj+r
		best = math.Min(best, g.ringDist(p, i0, i1, j0, j1))
		if i0 <= 0 && j0 <= 0 && i1 >= g.nx-1 && j1 >= g.ny-1 {
			return best // every cell visited
		}
		// A segment not yet visited lies in cells outside the block of rings 0 to r: at least bound from p. The sides of
		// the block that are the grid's own edges have nothing beyond them.
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
