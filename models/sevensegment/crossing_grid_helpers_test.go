package sevensegment_test

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// crossingGrid buckets the triangles of a closed mesh by their extent in y and z, so that pointIn's ray along +X
// tests only the triangles near the ray. It counts the same crossings as pointIn: every triangle that pointIn does not
// skip lies in the cells it reads, and each triangle is counted once.
type crossingGrid struct {
	mesh   *decad.Mesh
	lo, hi r3.Vec // the mesh's box
	cell   float64
	ny, nz int
	cells  [][]int32
}

// pointInMargin is pointIn's skip margin: a triangle lying wholly this far beyond the point in y, z or -x is skipped.
const pointInMargin = 0.05

func newCrossingGrid(mesh *decad.Mesh) *crossingGrid {
	vs, ts := mesh.Vertices(), mesh.Triangles()
	g := &crossingGrid{mesh: mesh, lo: r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)},
		hi: r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}}
	for _, v := range vs {
		g.lo = r3.Vec{X: math.Min(g.lo.X, v.X), Y: math.Min(g.lo.Y, v.Y), Z: math.Min(g.lo.Z, v.Z)}
		g.hi = r3.Vec{X: math.Max(g.hi.X, v.X), Y: math.Max(g.hi.Y, v.Y), Z: math.Max(g.hi.Z, v.Z)}
	}
	g.cell = math.Max(math.Max(g.hi.Y-g.lo.Y, g.hi.Z-g.lo.Z)/128, 0.25)
	g.ny = int((g.hi.Y-g.lo.Y)/g.cell) + 1
	g.nz = int((g.hi.Z-g.lo.Z)/g.cell) + 1
	g.cells = make([][]int32, g.ny*g.nz)
	for k, tri := range ts {
		a, b, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		j0, k0 := g.cellOf(math.Min(a.Y, math.Min(b.Y, c.Y)), math.Min(a.Z, math.Min(b.Z, c.Z)))
		j1, k1 := g.cellOf(math.Max(a.Y, math.Max(b.Y, c.Y)), math.Max(a.Z, math.Max(b.Z, c.Z)))
		for kk := k0; kk <= k1; kk++ {
			for j := j0; j <= j1; j++ {
				g.cells[kk*g.ny+j] = append(g.cells[kk*g.ny+j], int32(k))
			}
		}
	}
	return g
}

// cellOf returns the cell of (y, z), clamped to the grid.
func (g *crossingGrid) cellOf(y, z float64) (int, int) {
	j := int(math.Floor((y - g.lo.Y) / g.cell))
	k := int(math.Floor((z - g.lo.Z) / g.cell))
	return min(max(j, 0), g.ny-1), min(max(k, 0), g.nz-1)
}

// pointIn reports what pointIn(p, g.mesh) reports.
func (g *crossingGrid) pointIn(p r3.Vec) bool {
	// pointIn skips every triangle when p lies beyond the mesh's box by the margin: none crosses its ray.
	if g.hi.X < p.X-pointInMargin || g.lo.Y > p.Y+pointInMargin || g.hi.Y < p.Y-pointInMargin ||
		g.lo.Z > p.Z+pointInMargin || g.hi.Z < p.Z-pointInMargin {
		return false
	}
	// A triangle that pointIn tests reaches within the margin of p in y and z, so it lies in a cell this window reads.
	// A triangle registered in more than one of these cells is counted in the first cell of the window that holds it.
	j0, k0 := g.cellOf(p.Y-pointInMargin, p.Z-pointInMargin)
	j1, k1 := g.cellOf(p.Y+pointInMargin, p.Z+pointInMargin)
	vs, ts := g.mesh.Vertices(), g.mesh.Triangles()
	crossings := 0
	for kk := k0; kk <= k1; kk++ {
		for j := j0; j <= j1; j++ {
			for _, idx := range g.cells[kk*g.ny+j] {
				tri := ts[idx]
				a, b, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
				// Count the triangle only in the first window cell it is registered in.
				tj, tk := g.cellOf(math.Min(a.Y, math.Min(b.Y, c.Y)), math.Min(a.Z, math.Min(b.Z, c.Z)))
				if max(tj, j0) != j || max(tk, k0) != kk {
					continue
				}
				if crossesRay(p, a, b, c) {
					crossings++
				}
			}
		}
	}
	return crossings%2 == 1
}
