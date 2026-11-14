// Package meshgap screens boundary distances between verified gallery meshes.
package meshgap

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/r3"
)

type triangle struct {
	p      [3]r3.Vec
	lo, hi r3.Vec
}

type node struct {
	lo, hi      r3.Vec
	start, end  int
	left, right int
}

// Mesh caches triangles by value so later changes to the input slices cannot
// change a distance check. The caller must verify the source mesh and its bound.
type Mesh struct {
	triangles []triangle
	nodes     []node
	lo, hi    r3.Vec
	bound     float64
}

// New prepares a verified mesh whose displacement bound is in millimeters.
func New(vertices []r3.Vec, faces [][3]int, bound float64) (Mesh, error) {
	if len(vertices) == 0 || len(faces) == 0 {
		return Mesh{}, fmt.Errorf("meshgap: empty mesh")
	}
	if math.IsNaN(bound) || math.IsInf(bound, 0) || bound < 0 {
		return Mesh{}, fmt.Errorf("meshgap: invalid displacement bound")
	}
	for i, p := range vertices {
		if !finite(p.X) || !finite(p.Y) || !finite(p.Z) {
			return Mesh{}, fmt.Errorf("meshgap: nonfinite vertex %d", i)
		}
	}
	m := Mesh{bound: bound, lo: vertices[0], hi: vertices[0]}
	for i, face := range faces {
		var t triangle
		for j, index := range face {
			if index < 0 || index >= len(vertices) {
				return Mesh{}, fmt.Errorf("meshgap: invalid vertex index at face %d", i)
			}
			t.p[j] = vertices[index]
		}
		t.lo, t.hi = t.p[0], t.p[0]
		for _, p := range t.p {
			t.lo, t.hi = extend(t.lo, t.hi, p)
			m.lo, m.hi = extend(m.lo, m.hi, p)
		}
		m.triangles = append(m.triangles, t)
	}
	m.build(0, len(m.triangles))
	return m, nil
}

func (m *Mesh) build(start, end int) int {
	n := node{lo: m.triangles[start].lo, hi: m.triangles[start].hi, start: start, end: end, left: -1, right: -1}
	for _, t := range m.triangles[start:end] {
		n.lo, n.hi = extend(n.lo, n.hi, t.lo)
		n.lo, n.hi = extend(n.lo, n.hi, t.hi)
	}
	index := len(m.nodes)
	m.nodes = append(m.nodes, n)
	if end-start <= 8 {
		return index
	}
	span := n.hi.Sub(n.lo)
	axis := 0
	if span.Y > span.X {
		axis = 1
	}
	if span.Z > math.Max(span.X, span.Y) {
		axis = 2
	}
	centre := func(t triangle) float64 {
		switch axis {
		case 0:
			return t.lo.X/2 + t.hi.X/2
		case 1:
			return t.lo.Y/2 + t.hi.Y/2
		default:
			return t.lo.Z/2 + t.hi.Z/2
		}
	}
	slices.SortFunc(m.triangles[start:end], func(a, b triangle) int { return cmp.Compare(centre(a), centre(b)) })
	middle := start + (end-start)/2
	left, right := m.build(start, middle), m.build(middle, end)
	m.nodes[index].left, m.nodes[index].right = left, right
	return index
}

func span(n node) float64 {
	d := n.hi.Sub(n.lo)
	return math.Max(d.X, math.Max(d.Y, d.Z))
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

func extend(lo, hi, p r3.Vec) (r3.Vec, r3.Vec) {
	return r3.Vec{X: math.Min(lo.X, p.X), Y: math.Min(lo.Y, p.Y), Z: math.Min(lo.Z, p.Z)},
		r3.Vec{X: math.Max(hi.X, p.X), Y: math.Max(hi.Y, p.Y), Z: math.Max(hi.Z, p.Z)}
}

// Exceeds returns a gap lower bound and whether the boundary gap exceeds clearance after charging both
// verified displacement bounds and an arithmetic margin. False is inconclusive:
// it does not prove contact, retention, or absence of material interference.
// This reads immutable mesh data and may run concurrently on the same Mesh.
func (m Mesh) Exceeds(other Mesh, clearance float64) (float64, bool, error) {
	if len(m.triangles) == 0 || len(other.triangles) == 0 {
		return 0, false, fmt.Errorf("meshgap: empty mesh")
	}
	if !finite(clearance) || clearance < 0 {
		return 0, false, fmt.Errorf("meshgap: invalid clearance")
	}
	margin := m.bound + other.bound + 1e-6
	limit := clearance + margin
	if !finite(limit * limit) {
		return 0, false, fmt.Errorf("meshgap: clearance exceeds squared-distance range")
	}
	box := squaredBoxGap(triangle{lo: m.lo, hi: m.hi}, triangle{lo: other.lo, hi: other.hi})
	if !finite(box) {
		return 0, false, fmt.Errorf("meshgap: bounds exceed squared-distance range")
	}
	if box > limit*limit {
		return math.Max(0, math.Sqrt(box)-margin), true, nil
	}
	best := math.Inf(1)
	type pair struct{ a, b int }
	pending := []pair{{0, 0}}
	for len(pending) != 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		a, b := m.nodes[current.a], other.nodes[current.b]
		gap := squaredBoxGap(triangle{lo: a.lo, hi: a.hi}, triangle{lo: b.lo, hi: b.hi})
		if !finite(gap) {
			return 0, false, fmt.Errorf("meshgap: node bounds exceed squared-distance range")
		}
		if gap >= best {
			continue
		}
		if a.left >= 0 || b.left >= 0 {
			if b.left < 0 || (a.left >= 0 && span(a) >= span(b)) {
				pending = append(pending, pair{a.left, current.b}, pair{a.right, current.b})
			} else {
				pending = append(pending, pair{current.a, b.left}, pair{current.a, b.right})
			}
			continue
		}
		for _, ta := range m.triangles[a.start:a.end] {
			for _, tb := range other.triangles[b.start:b.end] {
				if squaredBoxGap(ta, tb) >= best {
					continue
				}
				distance := squaredTriangleGap(ta.p, tb.p)
				if !finite(distance) || distance < 0 {
					return 0, false, fmt.Errorf("meshgap: triangle distance is not finite and nonnegative")
				}
				best = math.Min(best, distance)
				if best <= limit*limit {
					// A close pair bounds the minimum from above. Only the whole-mesh
					// boxes bound it from below when stopping before all pairs are searched.
					return math.Max(0, math.Sqrt(box)-margin), false, nil
				}
			}
		}
	}
	return math.Max(0, math.Sqrt(best)-margin), true, nil
}

func squaredBoxGap(a, b triangle) float64 {
	x := math.Max(0, math.Max(a.lo.X-b.hi.X, b.lo.X-a.hi.X))
	y := math.Max(0, math.Max(a.lo.Y-b.hi.Y, b.lo.Y-a.hi.Y))
	z := math.Max(0, math.Max(a.lo.Z-b.hi.Z, b.lo.Z-a.hi.Z))
	return x*x + y*y + z*z
}

func squaredTriangleGap(a, b [3]r3.Vec) float64 {
	best := math.Inf(1)
	for i := range 3 {
		if segmentPierces(a[i], a[(i+1)%3], b) || segmentPierces(b[i], b[(i+1)%3], a) {
			return 0
		}
		best = math.Min(best, squaredPointTriangle(a[i], b))
		best = math.Min(best, squaredPointTriangle(b[i], a))
		for j := range 3 {
			best = math.Min(best, squaredSegments(a[i], a[(i+1)%3], b[j], b[(j+1)%3]))
		}
	}
	return best
}

func segmentPierces(p, q r3.Vec, t [3]r3.Vec) bool {
	n := t[1].Sub(t[0]).Cross(t[2].Sub(t[0]))
	a, b := p.Sub(t[0]).Dot(n), q.Sub(t[0]).Dot(n)
	if a*b > 0 || math.Abs(a-b) < 1e-20 {
		return false
	}
	u := a / (a - b)
	if u < 0 || u > 1 {
		return false
	}
	return squaredPointTriangle(p.Add(q.Sub(p).Scale(u)), t) < 1e-16
}

func squaredPointTriangle(p r3.Vec, t [3]r3.Vec) float64 {
	a, b, c := t[0], t[1], t[2]
	ab, ac, ap := b.Sub(a), c.Sub(a), p.Sub(a)
	d1, d2 := ab.Dot(ap), ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return ap.Dot(ap)
	}
	bp := p.Sub(b)
	d3, d4 := ab.Dot(bp), ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return bp.Dot(bp)
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		d := p.Sub(a.Add(ab.Scale(ratio(d1, d1-d3))))
		return d.Dot(d)
	}
	cp := p.Sub(c)
	d5, d6 := ab.Dot(cp), ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return cp.Dot(cp)
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		d := p.Sub(a.Add(ac.Scale(ratio(d2, d2-d6))))
		return d.Dot(d)
	}
	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		w := ratio(d4-d3, (d4-d3)+(d5-d6))
		d := p.Sub(b.Add(c.Sub(b).Scale(w)))
		return d.Dot(d)
	}
	denom := va + vb + vc
	if math.Abs(denom) < 1e-30 {
		return 0
	}
	d := p.Sub(a.Add(ab.Scale(vb / denom)).Add(ac.Scale(vc / denom)))
	return d.Dot(d)
}

// ratio returns num/den, or 0 when den is 0. A sliver triangle whose vertices nearly coincide has such a denominator;
// the vertex the edge parameter 0 selects is then as close as the sliver gets.
func ratio(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

func squaredSegments(p1, q1, p2, q2 r3.Vec) float64 {
	d1, d2, r := q1.Sub(p1), q2.Sub(p2), p1.Sub(p2)
	a, e, f := d1.Dot(d1), d2.Dot(d2), d2.Dot(r)
	s, u := 0.0, 0.0
	if a <= 1e-20 && e <= 1e-20 {
		return r.Dot(r)
	}
	if a <= 1e-20 {
		u = math.Max(0, math.Min(1, f/e))
	} else {
		c := d1.Dot(r)
		if e <= 1e-20 {
			s = math.Max(0, math.Min(1, -c/a))
		} else {
			b := d1.Dot(d2)
			denom := a*e - b*b
			if denom != 0 {
				s = math.Max(0, math.Min(1, (b*f-c*e)/denom))
			}
			u = (b*s + f) / e
			if u < 0 {
				u = 0
				s = math.Max(0, math.Min(1, -c/a))
			} else if u > 1 {
				u = 1
				s = math.Max(0, math.Min(1, (b-c)/a))
			}
		}
	}
	d := p1.Add(d1.Scale(s)).Sub(p2.Add(d2.Scale(u)))
	return d.Dot(d)
}
