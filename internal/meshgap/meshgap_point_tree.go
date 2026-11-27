package meshgap

import (
	"cmp"
	"math"
	"slices"

	"github.com/lestrrat-3d/r3"
)

// pointLeafSize is the most points a leaf of a pointTree holds.
const pointLeafSize = 8

// pointNode is a box of query points: the points[start:end] of its tree, with children unless it is a leaf.
type pointNode struct {
	lo, hi      r3.Vec
	start, end  int
	left, right int
}

// pointTree is a tree of boxes over query points, built once per LeastPointDistance call.
type pointTree struct {
	points []r3.Vec
	nodes  []pointNode
}

func newPointTree(points []r3.Vec) *pointTree {
	t := &pointTree{points: points}
	t.build(0, len(points))
	return t
}

func (t *pointTree) build(start, end int) int {
	n := pointNode{lo: t.points[start], hi: t.points[start], start: start, end: end, left: -1, right: -1}
	for _, p := range t.points[start:end] {
		n.lo, n.hi = extend(n.lo, n.hi, p)
	}
	index := len(t.nodes)
	t.nodes = append(t.nodes, n)
	if end-start <= pointLeafSize {
		return index
	}
	span := n.hi.Sub(n.lo)
	axis := 0
	if span.Y > span.X {
		axis = 1
	}
	if span.Z > max(span.X, span.Y) {
		axis = 2
	}
	slices.SortFunc(t.points[start:end], func(a, b r3.Vec) int { return cmp.Compare(axisOf(a, axis), axisOf(b, axis)) })
	mid := start + (end-start)/2
	left := t.build(start, mid)
	right := t.build(mid, end)
	t.nodes[index].left, t.nodes[index].right = left, right
	return index
}

func axisOf(v r3.Vec, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	}
	return v.Z
}

// boxPair is a mesh node and a point node, with the squared gap between their boxes.
type boxPair struct {
	mesh, points int
	gap          float64
}

// leastSquaredDistance returns the least squared distance from the tree's points to the mesh triangles. A pair of
// boxes whose gap is at least the best squared distance so far holds no closer pair of point and triangle, so it is
// skipped; the nearer pair of the two children is visited first so the best shrinks early.
func (m Mesh) leastSquaredDistance(pt *pointTree) float64 {
	best := math.Inf(1)
	pending := make([]boxPair, 0, 128)
	gapOf := func(mi, pi int) float64 {
		a, b := m.nodes[mi], pt.nodes[pi]
		return squaredBoxGap(triangle{lo: a.lo, hi: a.hi}, triangle{lo: b.lo, hi: b.hi})
	}
	pending = append(pending, boxPair{gap: gapOf(0, 0)})
	for len(pending) != 0 {
		pair := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if pair.gap >= best {
			continue
		}
		n, q := m.nodes[pair.mesh], pt.nodes[pair.points]
		meshLeaf, pointLeaf := n.left < 0, q.left < 0
		if meshLeaf && pointLeaf {
			for _, p := range pt.points[q.start:q.end] {
				query := triangle{lo: p, hi: p}
				for _, t := range m.triangles[n.start:n.end] {
					if squaredBoxGap(query, t) >= best {
						continue
					}
					// A triangle whose squared distance is NaN (overflowing products) never lowers best, so it
					// cannot poison the distances of the points that follow.
					if d := squaredPointTriangle(p, t.p); d < best {
						best = d
					}
				}
			}
			continue
		}
		// Split the larger box that has children.
		splitMesh := !meshLeaf
		if !meshLeaf && !pointLeaf {
			a, b := n.hi.Sub(n.lo), q.hi.Sub(q.lo)
			splitMesh = a.Dot(a) >= b.Dot(b)
		}
		var first, second boxPair
		if splitMesh {
			first = boxPair{mesh: n.left, points: pair.points}
			second = boxPair{mesh: n.right, points: pair.points}
		} else {
			first = boxPair{mesh: pair.mesh, points: q.left}
			second = boxPair{mesh: pair.mesh, points: q.right}
		}
		first.gap, second.gap = gapOf(first.mesh, first.points), gapOf(second.mesh, second.points)
		if first.gap > second.gap { // push the farther pair first so the nearer one pops first
			first, second = second, first
		}
		pending = append(pending, second, first)
	}
	return best
}
