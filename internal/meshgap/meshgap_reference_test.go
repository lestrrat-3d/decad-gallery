package meshgap

// This file is a copy of internal/meshgap/meshgap.go at origin/main 6ccf1056d, before the clearance speedup. It is
// the reference the differential tests compare the production code with. Only the identifiers carry a ref prefix;
// the logic, the evaluation order and the error text are unchanged. Do not tidy it.

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/r3"
)

type refTriangle struct {
	p      [3]r3.Vec
	lo, hi r3.Vec
}

type refNode struct {
	lo, hi      r3.Vec
	start, end  int
	left, right int
}

// With coordinate magnitudes, displacement bounds, and clearance at most C=2^16,
// coordinate deltas are at most 2C=2^17 and squared box gaps are below 3*2^34<2^36.
// The charged limit is below 3C+1e-6<2^18, so its square is below 2^36. Triangle
// distance dot products stay below 2^36, cross components below 2^35, plane-dot
// terms below 2^54, and their products below 2^108. The point-triangle interior
// denominator is used only when its magnitude is at least 1e-30, keeping its
// quotient below 2^173. A near-parallel squared-segment quotient may overflow,
// but the code clamps it to [0,1] before using it.

// Mesh caches triangles by value so later changes to the input slices cannot
// change a distance check. The caller must verify the source mesh and its bound.
type refMesh struct {
	triangles         []refTriangle
	nodes             []refNode
	lo, hi            r3.Vec
	bound             float64
	clearanceEligible bool
}

// New prepares a verified mesh whose displacement bound is in millimeters.
func refNew(vertices []r3.Vec, faces [][3]int, bound float64) (refMesh, error) {
	if len(vertices) == 0 || len(faces) == 0 {
		return refMesh{}, fmt.Errorf("meshgap: empty mesh")
	}
	if math.IsNaN(bound) || math.IsInf(bound, 0) || bound < 0 {
		return refMesh{}, fmt.Errorf("meshgap: invalid displacement bound")
	}
	eligible := bound <= clearanceFastPathLimit
	for i, p := range vertices {
		if !refFinite(p.X) || !refFinite(p.Y) || !refFinite(p.Z) {
			return refMesh{}, fmt.Errorf("meshgap: nonfinite vertex %d", i)
		}
		if math.Abs(p.X) > clearanceFastPathLimit || math.Abs(p.Y) > clearanceFastPathLimit ||
			math.Abs(p.Z) > clearanceFastPathLimit {
			eligible = false
		}
	}
	m := refMesh{bound: bound, lo: vertices[0], hi: vertices[0], clearanceEligible: eligible}
	for i, face := range faces {
		var t refTriangle
		for j, index := range face {
			if index < 0 || index >= len(vertices) {
				return refMesh{}, fmt.Errorf("meshgap: invalid vertex index at face %d", i)
			}
			t.p[j] = vertices[index]
		}
		t.lo, t.hi = t.p[0], t.p[0]
		for _, p := range t.p {
			t.lo, t.hi = refExtend(t.lo, t.hi, p)
			m.lo, m.hi = refExtend(m.lo, m.hi, p)
		}
		m.triangles = append(m.triangles, t)
	}
	m.refBuild(0, len(m.triangles))
	return m, nil
}

func (m *refMesh) refBuild(start, end int) int {
	n := refNode{lo: m.triangles[start].lo, hi: m.triangles[start].hi, start: start, end: end, left: -1, right: -1}
	for _, t := range m.triangles[start:end] {
		n.lo, n.hi = refExtend(n.lo, n.hi, t.lo)
		n.lo, n.hi = refExtend(n.lo, n.hi, t.hi)
	}
	index := len(m.nodes)
	m.nodes = append(m.nodes, n)
	if end-start <= 8 {
		return index
	}
	refSpan := n.hi.Sub(n.lo)
	axis := 0
	if refSpan.Y > refSpan.X {
		axis = 1
	}
	if refSpan.Z > math.Max(refSpan.X, refSpan.Y) {
		axis = 2
	}
	centre := func(t refTriangle) float64 {
		switch axis {
		case 0:
			return t.lo.X/2 + t.hi.X/2
		case 1:
			return t.lo.Y/2 + t.hi.Y/2
		default:
			return t.lo.Z/2 + t.hi.Z/2
		}
	}
	slices.SortFunc(m.triangles[start:end], func(a, b refTriangle) int { return cmp.Compare(centre(a), centre(b)) })
	middle := start + (end-start)/2
	left, right := m.refBuild(start, middle), m.refBuild(middle, end)
	m.nodes[index].left, m.nodes[index].right = left, right
	return index
}

func refSpan(n refNode) float64 {
	d := n.hi.Sub(n.lo)
	return math.Max(d.X, math.Max(d.Y, d.Z))
}

func refFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

func refExtend(lo, hi, p r3.Vec) (r3.Vec, r3.Vec) {
	return r3.Vec{X: math.Min(lo.X, p.X), Y: math.Min(lo.Y, p.Y), Z: math.Min(lo.Z, p.Z)},
		r3.Vec{X: math.Max(hi.X, p.X), Y: math.Max(hi.Y, p.Y), Z: math.Max(hi.Z, p.Z)}
}

// Exceeds returns a gap lower bound and whether the boundary gap exceeds clearance after charging both
// verified displacement bounds and an arithmetic margin. False is inconclusive:
// it does not prove contact, retention, or absence of material interference.
// This reads immutable mesh data and may run concurrently on the same Mesh.
func (m refMesh) Exceeds(other refMesh, clearance float64) (float64, bool, error) {
	if len(m.triangles) == 0 || len(other.triangles) == 0 {
		return 0, false, fmt.Errorf("meshgap: empty mesh")
	}
	if !refFinite(clearance) || clearance < 0 {
		return 0, false, fmt.Errorf("meshgap: invalid clearance")
	}
	margin := m.bound + other.bound + 1e-6
	limit := clearance + margin
	if !refFinite(limit * limit) {
		return 0, false, fmt.Errorf("meshgap: clearance exceeds squared-distance range")
	}
	box := refSquaredBoxGap(refTriangle{lo: m.lo, hi: m.hi}, refTriangle{lo: other.lo, hi: other.hi})
	if !refFinite(box) {
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
		gap := refSquaredBoxGap(refTriangle{lo: a.lo, hi: a.hi}, refTriangle{lo: b.lo, hi: b.hi})
		if !refFinite(gap) {
			return 0, false, fmt.Errorf("meshgap: node bounds exceed squared-distance range")
		}
		if gap >= best {
			continue
		}
		if a.left >= 0 || b.left >= 0 {
			if b.left < 0 || (a.left >= 0 && refSpan(a) >= refSpan(b)) {
				pending = append(pending, pair{a.left, current.b}, pair{a.right, current.b})
			} else {
				pending = append(pending, pair{current.a, b.left}, pair{current.a, b.right})
			}
			continue
		}
		for _, ta := range m.triangles[a.start:a.end] {
			for _, tb := range other.triangles[b.start:b.end] {
				if refSquaredBoxGap(ta, tb) >= best {
					continue
				}
				distance := refSquaredTriangleGap(ta.p, tb.p)
				if !refFinite(distance) || distance < 0 {
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

// ExceedsClearance returns whether the boundary gap exceeds clearance after charging both
// verified displacement bounds and the arithmetic margin. False is inconclusive: it does not
// prove contact, retention, or absence of material interference. Meshes outside the bounded
// arithmetic range use Exceeds and discard its gap value.
func (m refMesh) ExceedsClearance(other refMesh, clearance float64) (bool, error) {
	if len(m.triangles) == 0 || len(other.triangles) == 0 ||
		!refFinite(clearance) || clearance < 0 || clearance > clearanceFastPathLimit ||
		!m.clearanceEligible || !other.clearanceEligible {
		_, separated, err := m.Exceeds(other, clearance)
		return separated, err
	}

	margin := m.bound + other.bound + 1e-6
	limit := clearance + margin
	limitSquared := limit * limit
	box := refSquaredBoxGap(refTriangle{lo: m.lo, hi: m.hi}, refTriangle{lo: other.lo, hi: other.hi})
	if !refFinite(limitSquared) || !refFinite(box) {
		_, separated, err := m.Exceeds(other, clearance)
		return separated, err
	}
	if box > limitSquared {
		return true, nil
	}

	separated, decided := m.refExceedsClearanceThreshold(other, clearance, limitSquared)
	if decided {
		return separated, nil
	}
	_, separated, err := m.Exceeds(other, clearance)
	return separated, err
}

// refExceedsClearanceThreshold reports whether the bounded DFS reached a decision. A false
// decision flag asks the caller to repeat the query through Exceeds for identical error handling.
func (m refMesh) refExceedsClearanceThreshold(other refMesh, clearance, limitSquared float64) (bool, bool) {
	threshold, ok := refClearanceThresholdUpper(clearance, m.bound, other.bound)
	if !ok {
		return false, false
	}
	type pair struct{ a, b int }
	pending := []pair{{0, 0}}
	for len(pending) != 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		a, b := m.nodes[current.a], other.nodes[current.b]
		lower, valid := refLowerBoxGap(refTriangle{lo: a.lo, hi: a.hi}, refTriangle{lo: b.lo, hi: b.hi})
		if !valid {
			return false, false
		}
		if lower > threshold {
			continue
		}
		if a.left >= 0 || b.left >= 0 {
			if b.left < 0 || (a.left >= 0 && refSpan(a) >= refSpan(b)) {
				pending = append(pending, pair{a.left, current.b}, pair{a.right, current.b})
			} else {
				pending = append(pending, pair{current.a, b.left}, pair{current.a, b.right})
			}
			continue
		}
		for _, ta := range m.triangles[a.start:a.end] {
			for _, tb := range other.triangles[b.start:b.end] {
				lower, valid = refLowerBoxGap(ta, tb)
				if !valid {
					return false, false
				}
				if lower > threshold {
					continue
				}
				distance := refSquaredTriangleGap(ta.p, tb.p)
				if !refFinite(distance) || distance < 0 {
					return false, false
				}
				if distance <= limitSquared {
					return false, true
				}
			}
		}
	}
	return true, true
}

func refClearanceThresholdUpper(clearance, boundA, boundB float64) (float64, bool) {
	margin := refRoundUp(boundA + boundB)
	margin = refRoundUp(margin + 1e-6)
	limit := refRoundUp(clearance + margin)
	threshold := refRoundUp(limit * limit)
	return threshold, refFinite(threshold) && threshold >= 0
}

func refRoundUp(value float64) float64 {
	if !refFinite(value) {
		return value
	}
	return math.Nextafter(value, math.Inf(1))
}

func refLowerBoxGap(a, b refTriangle) (float64, bool) {
	axisGap := func(aLo, aHi, bLo, bHi float64) (float64, bool) {
		x := refRoundDownNonnegative(aLo - bHi)
		y := refRoundDownNonnegative(bLo - aHi)
		gap := math.Max(0, math.Max(x, y))
		squared := refRoundDownNonnegative(gap * gap)
		return squared, refFinite(squared)
	}
	x, ok := axisGap(a.lo.X, a.hi.X, b.lo.X, b.hi.X)
	if !ok {
		return 0, false
	}
	y, ok := axisGap(a.lo.Y, a.hi.Y, b.lo.Y, b.hi.Y)
	if !ok {
		return 0, false
	}
	z, ok := axisGap(a.lo.Z, a.hi.Z, b.lo.Z, b.hi.Z)
	if !ok {
		return 0, false
	}
	xy := refRoundDownNonnegative(x + y)
	if !refFinite(xy) {
		return 0, false
	}
	squared := refRoundDownNonnegative(xy + z)
	return squared, refFinite(squared)
}

func refRoundDownNonnegative(value float64) float64 {
	if !refFinite(value) {
		return value
	}
	if value <= 0 {
		return 0
	}
	return math.Nextafter(value, math.Inf(-1))
}

// PointDistance returns the distance from p to the nearest triangle of the mesh. It reads the triangles only and
// does not charge the displacement bound; the caller adds Bound where it compares with the denoted body.
// This reads immutable mesh data and may run concurrently on the same Mesh.
func (m refMesh) PointDistance(p r3.Vec) (float64, error) {
	if len(m.triangles) == 0 {
		return 0, fmt.Errorf("meshgap: empty mesh")
	}
	if !refFinite(p.X) || !refFinite(p.Y) || !refFinite(p.Z) {
		return 0, fmt.Errorf("meshgap: nonfinite query point")
	}
	query := refTriangle{lo: p, hi: p}
	best := math.Inf(1)
	pending := []int{0}
	for len(pending) != 0 {
		n := m.nodes[pending[len(pending)-1]]
		pending = pending[:len(pending)-1]
		if refSquaredBoxGap(query, refTriangle{lo: n.lo, hi: n.hi}) >= best {
			continue
		}
		if n.left >= 0 {
			pending = append(pending, n.left, n.right)
			continue
		}
		for _, t := range m.triangles[n.start:n.end] {
			if refSquaredBoxGap(query, t) >= best {
				continue
			}
			best = math.Min(best, refSquaredPointTriangle(p, t.p))
		}
	}
	if !refFinite(best) {
		return 0, fmt.Errorf("meshgap: point distance is not finite")
	}
	return math.Sqrt(best), nil
}

// Bound returns the displacement bound New was given, in millimeters.
func (m refMesh) Bound() float64 { return m.bound }

func refSquaredBoxGap(a, b refTriangle) float64 {
	x := math.Max(0, math.Max(a.lo.X-b.hi.X, b.lo.X-a.hi.X))
	y := math.Max(0, math.Max(a.lo.Y-b.hi.Y, b.lo.Y-a.hi.Y))
	z := math.Max(0, math.Max(a.lo.Z-b.hi.Z, b.lo.Z-a.hi.Z))
	return x*x + y*y + z*z
}

func refSquaredTriangleGap(a, b [3]r3.Vec) float64 {
	best := math.Inf(1)
	for i := range 3 {
		if refSegmentPierces(a[i], a[(i+1)%3], b) || refSegmentPierces(b[i], b[(i+1)%3], a) {
			return 0
		}
		best = math.Min(best, refSquaredPointTriangle(a[i], b))
		best = math.Min(best, refSquaredPointTriangle(b[i], a))
		for j := range 3 {
			best = math.Min(best, refSquaredSegments(a[i], a[(i+1)%3], b[j], b[(j+1)%3]))
		}
	}
	return best
}

func refSegmentPierces(p, q r3.Vec, t [3]r3.Vec) bool {
	n := t[1].Sub(t[0]).Cross(t[2].Sub(t[0]))
	a, b := p.Sub(t[0]).Dot(n), q.Sub(t[0]).Dot(n)
	if a*b > 0 || math.Abs(a-b) < 1e-20 {
		return false
	}
	u := a / (a - b)
	if u < 0 || u > 1 {
		return false
	}
	return refSquaredPointTriangle(p.Add(q.Sub(p).Scale(u)), t) < 1e-16
}

func refSquaredPointTriangle(p r3.Vec, t [3]r3.Vec) float64 {
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
		d := p.Sub(a.Add(ab.Scale(refRatio(d1, d1-d3))))
		return d.Dot(d)
	}
	cp := p.Sub(c)
	d5, d6 := ab.Dot(cp), ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return cp.Dot(cp)
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		d := p.Sub(a.Add(ac.Scale(refRatio(d2, d2-d6))))
		return d.Dot(d)
	}
	va := d3*d6 - d5*d4
	if va <= 0 && (d4-d3) >= 0 && (d5-d6) >= 0 {
		w := refRatio(d4-d3, (d4-d3)+(d5-d6))
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

// refRatio returns num/den, or 0 when den is 0. A sliver triangle whose vertices nearly coincide has such a denominator;
// the vertex the edge parameter 0 selects is then as close as the sliver gets.
func refRatio(num, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

func refSquaredSegments(p1, q1, p2, q2 r3.Vec) float64 {
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
