package sevensegment_test

import (
	"cmp"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
)

// rayTri is one triangle with its edges, its bounding box, the kind of body it belongs to and its place in the list it
// was cast against.
type rayTri struct {
	a, e1, e2 r3.Vec // a corner and the edges from it to the other two
	lo, hi    r3.Vec
	kind      int   // 0 plate, 1 black, 2 yellow
	idx       int32 // its index in the list a rayTree was built from: of two hits at one distance the lower index wins
}

const (
	kindPlate = iota
	kindBlack
	kindYellow
)

// trisOf returns the triangles of mesh moved by shift, tagged with kind.
func trisOf(mesh *decad.Mesh, turn *r3.Transform, shift r3.Vec, kind int) []rayTri {
	vs, ts := mesh.Vertices(), mesh.Triangles()
	out := make([]rayTri, 0, len(ts))
	for _, t := range ts {
		p := [3]r3.Vec{vs[t[0]], vs[t[1]], vs[t[2]]}
		for i := range p {
			if turn != nil {
				p[i] = turn.Apply(p[i])
			}
			p[i] = p[i].Add(shift)
		}
		lo := r3.Vec{X: min(p[0].X, p[1].X, p[2].X), Y: min(p[0].Y, p[1].Y, p[2].Y), Z: min(p[0].Z, p[1].Z, p[2].Z)}
		hi := r3.Vec{X: max(p[0].X, p[1].X, p[2].X), Y: max(p[0].Y, p[1].Y, p[2].Y), Z: max(p[0].Z, p[1].Z, p[2].Z)}
		out = append(out, rayTri{a: p[0], e1: p[1].Sub(p[0]), e2: p[2].Sub(p[0]), lo: lo, hi: hi, kind: kind})
	}
	return out
}

// slab returns where the ray from o with inverse direction inv enters and leaves the box lo, hi. The builtin min and
// max keep math.Min's and math.Max's NaN rule, which a ray along an axis plane relies on, and compile inline.
func slab(lo, hi, o, inv r3.Vec) (tmin, tmax float64) {
	t1, t2 := (lo.X-o.X)*inv.X, (hi.X-o.X)*inv.X
	tmin, tmax = min(t1, t2), max(t1, t2)
	t1, t2 = (lo.Y-o.Y)*inv.Y, (hi.Y-o.Y)*inv.Y
	tmin, tmax = max(tmin, min(t1, t2)), min(tmax, max(t1, t2))
	t1, t2 = (lo.Z-o.Z)*inv.Z, (hi.Z-o.Z)*inv.Z
	return max(tmin, min(t1, t2)), min(tmax, max(t1, t2))
}

// hitTri returns the distance along the ray from o along d (inv its componentwise inverse) at which it hits t, and
// false when it misses t or t's box starts beyond best.
func hitTri(t *rayTri, o, d, inv r3.Vec, best float64) (float64, bool) {
	if tmin, tmax := slab(t.lo, t.hi, o, inv); tmax < max(tmin, 0) || tmin > best {
		return 0, false
	}
	h := d.Cross(t.e2)
	det := t.e1.Dot(h)
	if math.Abs(det) < 1e-14 {
		return 0, false
	}
	s := o.Sub(t.a)
	u := s.Dot(h) / det
	// A small tolerance on the barycentric coordinates keeps rays from slipping through the seams between triangles.
	const eps = 1e-7
	if u < -eps || u > 1+eps {
		return 0, false
	}
	q := s.Cross(t.e1)
	v := d.Dot(q) / det
	if v < -eps || u+v > 1+eps {
		return 0, false
	}
	dist := t.e2.Dot(q) / det
	return dist, dist > 1e-9
}

// firstHitDist returns the distance to the nearest triangle the ray from o along d hits and its kind, or +Inf and -1.
// Of two hits at one distance the earlier triangle wins.
func firstHitDist(tris []rayTri, o, d r3.Vec) (float64, int) {
	best, kind := math.Inf(1), -1
	inv := r3.Vec{X: 1 / d.X, Y: 1 / d.Y, Z: 1 / d.Z}
	for i := range tris {
		if dist, ok := hitTri(&tris[i], o, d, inv, best); ok && dist < best {
			best, kind = dist, tris[i].kind
		}
	}
	return best, kind
}

// windowNear returns the triangles of tris near segment s's window, in their order.
func windowNear(tris []rayTri, s sevensegment.Segment) []rayTri {
	cx, cy, _ := windowCentre(s)
	var near []rayTri
	for _, t := range tris {
		if t.hi.X < cx-39.0 || t.lo.X > cx+39.0 || t.hi.Y < cy-39.0 || t.lo.Y > cy+39.0 {
			continue
		}
		near = append(near, t)
	}
	return near
}

// windowCentre is segment s's window centre in plan and whether the window lies along X.
func windowCentre(s sevensegment.Segment) (cx, cy float64, horizontal bool) {
	switch s {
	case sevensegment.SegA:
		return 0, sevensegment.SegmentSpan, true
	case sevensegment.SegD:
		return 0, -sevensegment.SegmentSpan, true
	case sevensegment.SegG:
		return 0, 0, true
	case sevensegment.SegB:
		return sevensegment.SegmentSpan / 2, sevensegment.SegmentSpan / 2, false
	case sevensegment.SegC:
		return sevensegment.SegmentSpan / 2, -sevensegment.SegmentSpan / 2, false
	case sevensegment.SegE:
		return -sevensegment.SegmentSpan / 2, -sevensegment.SegmentSpan / 2, false
	}
	return -sevensegment.SegmentSpan / 2, sevensegment.SegmentSpan / 2, false
}

// windowShares casts a ray grid over segment s's window, entering at the display face, at elevation elev degrees with
// the horizontal part of the ray along dir (a unit horizontal vector), and returns over the rays whose entry point has
// |u| <= 9.9: the share that first hits a flag body, and the share of those that first hit yellow. tree holds the
// triangles near the window (windowNear).
func windowShares(tree *rayTree, s sevensegment.Segment, faceZ, elev float64, dir r3.Vec) (flag, yellow float64) {
	cx, cy, horizontal := windowCentre(s)
	e := elev * math.Pi / 180
	d := r3.Vec{X: -dir.X * math.Cos(e), Y: -dir.Y * math.Cos(e), Z: -math.Sin(e)}
	total, flags, yellows := 0, 0, 0
	const step = 0.4
	for u := -14.85; u <= 14.85; u += step {
		for sv := -(sevensegment.WindowHalfWidth - 0.05); sv <= sevensegment.WindowHalfWidth-0.05; sv += step { // inside the window
			var x, y float64
			if horizontal {
				x, y = cx+u, cy+sv
			} else {
				x, y = cx-sv, cy+u
			}
			entry := r3.Vec{X: x, Y: y, Z: faceZ}
			origin := entry.Sub(d.Scale(8))
			total++
			switch tree.first(origin, d) {
			case kindBlack:
				flags++
			case kindYellow:
				flags++
				yellows++
			}
		}
	}
	return float64(flags) / float64(total), float64(yellows) / math.Max(float64(flags), 1)
}

// openingEntries returns the plan points of segment s's window opening on a 0.05 grid over |s| <= 6 and |u| <= 19.5: those where
// a ray going straight down from above the display face passes the face plane, that is, where the plate has no material.
// The opening is read from the plate mesh, so a window that is cut wider than designed shows more of the rotor.
func openingEntries(plate *rayTree, s sevensegment.Segment, faceZ float64) [][2]float64 {
	const step = 0.05
	cx, cy, horizontal := segmentCentre(s)
	var out [][2]float64
	// The grid is offset by half a step, so that no ray lies in the plane u = +-10 or s = 0: those planes hold the mesh edges of
	// both bodies and a ray in them can slip through a crack between triangles and report yellow that is not there.
	for u := -19.5 + step/2; u <= 19.5; u += step {
		for sv := -6.0 + step/2; sv <= 6.0; sv += step {
			x, y := cx-sv, cy+u
			if horizontal {
				x, y = cx+u, cy+sv
			}
			dist := plate.firstDist(r3.Vec{X: x, Y: y, Z: faceZ + 1}, r3.Vec{X: 1e-9, Y: 1e-9, Z: -1})
			if dist > 1.05 {
				out = append(out, [2]float64{x, y})
			}
		}
	}
	return out
}

// windowRays casts a ray from each entry point at elevation elev with the horizontal part of the ray along dir, and
// returns how many first hit yellow. tris holds the plate and the flag bodies, indexed (indexTris).
func windowRays(tris []rayTri, entries [][2]float64, faceZ, elev float64, dir r3.Vec) int {
	e := elev * math.Pi / 180
	d := r3.Vec{X: -dir.X * math.Cos(e), Y: -dir.Y * math.Cos(e), Z: -math.Sin(e)}
	origins := make([]r3.Vec, len(entries))
	for i, q := range entries {
		entry := r3.Vec{X: q[0], Y: q[1], Z: faceZ}
		origins[i] = entry.Sub(d.Scale(8))
	}
	g := newShadowGrid(tris, d, origins)
	yellow := 0
	for _, o := range origins {
		if g.first(o) == kindYellow {
			yellow++
		}
	}
	return yellow
}

// indexTris numbers tris in their order, as newRayTree does: of two hits at one distance the lower index wins.
func indexTris(tris []rayTri) []rayTri {
	for i := range tris {
		tris[i].idx = int32(i)
	}
	return tris
}

// shadowGrid bins triangles by their shadow on the plane across one ray direction, so that a ray along it tests only
// the triangles whose shadow covers its own point. It finds what a rayTree over the same triangles finds: a triangle
// left out of the ray's cell misses the ray by more than shadowMargin in that plane, far more than hitTri's barycentric
// tolerance lets a hit stray from the triangle, and the cell's triangles are tested with hitTri and the same tie rule.
type shadowGrid struct {
	tris        []rayTri
	d, inv      r3.Vec
	bu, bv      r3.Vec // the plane across d
	lou, lov    float64
	nu, nv      int
	cells       [][]int32
	emptyRegion bool
}

const (
	shadowMargin = 1e-3 // mm; hitTri's tolerance of 1e-7 strays 1e-5 mm on a 100 mm triangle
	shadowCell   = 0.1  // mm
)

func newShadowGrid(tris []rayTri, d r3.Vec, origins []r3.Vec) *shadowGrid {
	g := &shadowGrid{tris: tris, d: d, inv: r3.Vec{X: 1 / d.X, Y: 1 / d.Y, Z: 1 / d.Z}}
	ref := r3.Vec{Z: 1}
	if math.Abs(d.Z) > 0.9 {
		ref = r3.Vec{X: 1}
	}
	g.bu = d.Cross(ref)
	g.bu = g.bu.Scale(1 / g.bu.Len())
	g.bv = d.Cross(g.bu)
	g.bv = g.bv.Scale(1 / g.bv.Len())
	if len(origins) == 0 {
		g.emptyRegion = true
		return g
	}
	lou, lov, hiu, hiv := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, o := range origins {
		u, v := o.Dot(g.bu), o.Dot(g.bv)
		lou, lov, hiu, hiv = min(lou, u), min(lov, v), max(hiu, u), max(hiv, v)
	}
	g.lou, g.lov = lou-shadowMargin, lov-shadowMargin
	hiu, hiv = hiu+shadowMargin, hiv+shadowMargin
	g.nu = int((hiu-g.lou)/shadowCell) + 1
	g.nv = int((hiv-g.lov)/shadowCell) + 1
	g.cells = make([][]int32, g.nu*g.nv)
	for k := range tris {
		t := &tris[k]
		b, c := t.a.Add(t.e1), t.a.Add(t.e2)
		u0, u1, u2 := t.a.Dot(g.bu), b.Dot(g.bu), c.Dot(g.bu)
		v0, v1, v2 := t.a.Dot(g.bv), b.Dot(g.bv), c.Dot(g.bv)
		tlu, thu := min(u0, u1, u2)-shadowMargin, max(u0, u1, u2)+shadowMargin
		tlv, thv := min(v0, v1, v2)-shadowMargin, max(v0, v1, v2)+shadowMargin
		if thu < g.lou || tlu > hiu || thv < g.lov || tlv > hiv {
			continue // no ray's shadow comes near it
		}
		i0, j0 := g.cellOf(tlu, tlv)
		i1, j1 := g.cellOf(thu, thv)
		for j := j0; j <= j1; j++ {
			for i := i0; i <= i1; i++ {
				g.cells[j*g.nu+i] = append(g.cells[j*g.nu+i], int32(k))
			}
		}
	}
	return g
}

// cellOf returns the cell of the shadow point (u, v), clamped to the grid.
func (g *shadowGrid) cellOf(u, v float64) (int, int) {
	i := int(math.Floor((u - g.lou) / shadowCell))
	j := int(math.Floor((v - g.lov) / shadowCell))
	return min(max(i, 0), g.nu-1), min(max(j, 0), g.nv-1)
}

// first returns the kind of the nearest triangle the ray from o along the grid's direction hits, or -1. o must be one
// of the origins the grid was built for.
func (g *shadowGrid) first(o r3.Vec) int {
	if g.emptyRegion {
		return -1
	}
	i, j := g.cellOf(o.Dot(g.bu), o.Dot(g.bv))
	best, kind, bestIdx := math.Inf(1), -1, int32(math.MaxInt32)
	for _, k := range g.cells[j*g.nu+i] {
		tr := &g.tris[k]
		if dist, ok := hitTri(tr, o, g.d, g.inv, best); ok && (dist < best || dist == best && tr.idx < bestIdx) {
			best, kind, bestIdx = dist, tr.kind, tr.idx
		}
	}
	return kind
}

// segmentCentre is segment s's centre in plan and whether it lies across X.
func segmentCentre(s sevensegment.Segment) (cx, cy float64, horizontal bool) {
	point, axis := segmentAxis(s)
	return point.X, point.Y, axis.X != 0
}

// rayTree is a bounding volume tree over triangles, so that the whole-window grids stay fast. It finds what
// firstHitDist finds over the list it was built from: it skips only boxes the ray misses or enters beyond the best hit,
// and of two hits at one distance the lower index wins, as in the list's order.
type rayTree struct {
	tris  []rayTri
	nodes []rayNode
}

type rayNode struct {
	lo, hi      r3.Vec
	left, right int32 // children, -1 for a leaf
	start, n    int32 // the leaf's triangles
	axis        int32 // the axis the children are split along
}

func newRayTree(tris []rayTri) *rayTree {
	t := &rayTree{tris: append([]rayTri{}, tris...)}
	for i := range t.tris {
		t.tris[i].idx = int32(i)
	}
	t.build(0, len(t.tris))
	return t
}

func vecComp(v r3.Vec, axis int32) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	}
	return v.Z
}

func (t *rayTree) build(start, end int) int32 {
	lo := r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	hi := r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for i := start; i < end; i++ {
		q := &t.tris[i]
		lo = r3.Vec{X: min(lo.X, q.lo.X), Y: min(lo.Y, q.lo.Y), Z: min(lo.Z, q.lo.Z)}
		hi = r3.Vec{X: max(hi.X, q.hi.X), Y: max(hi.Y, q.hi.Y), Z: max(hi.Z, q.hi.Z)}
	}
	idx := int32(len(t.nodes))
	t.nodes = append(t.nodes, rayNode{lo: lo, hi: hi, left: -1, right: -1, start: int32(start), n: int32(end - start)})
	if end-start <= 6 {
		return idx
	}
	ext := hi.Sub(lo)
	axis := int32(0)
	if ext.Y > ext.X && ext.Y >= ext.Z {
		axis = 1
	} else if ext.Z > ext.X && ext.Z > ext.Y {
		axis = 2
	}
	sub := t.tris[start:end]
	slices.SortFunc(sub, func(a, b rayTri) int {
		return cmp.Compare(vecComp(a.lo, axis)+vecComp(a.hi, axis), vecComp(b.lo, axis)+vecComp(b.hi, axis))
	})
	mid := (start + end) / 2
	l, r := t.build(start, mid), t.build(mid, end)
	t.nodes[idx].left, t.nodes[idx].right, t.nodes[idx].axis = l, r, axis
	return idx
}

// first returns the kind of the nearest triangle the ray from o along d hits, or -1.
func (t *rayTree) first(o, d r3.Vec) int {
	_, kind := t.nearest(o, d)
	return kind
}

// firstDist returns the distance to the nearest triangle the ray hits, or +Inf.
func (t *rayTree) firstDist(o, d r3.Vec) float64 {
	dist, _ := t.nearest(o, d)
	return dist
}

func (t *rayTree) nearest(o, d r3.Vec) (float64, int) {
	inv := r3.Vec{X: 1 / d.X, Y: 1 / d.Y, Z: 1 / d.Z}
	best, kind, bestIdx := math.Inf(1), -1, int32(math.MaxInt32)
	if len(t.nodes) == 0 {
		return best, kind
	}
	// The tree is balanced, so its depth is about log2 of the triangle count: 64 levels hold any mesh.
	var stack [64]int32
	stack[0] = 0
	n := 1
	for n > 0 {
		n--
		nd := &t.nodes[stack[n]]
		if tmin, tmax := slab(nd.lo, nd.hi, o, inv); tmax < max(tmin, 0) || tmin > best {
			continue
		}
		if nd.left < 0 {
			for i := nd.start; i < nd.start+nd.n; i++ {
				tr := &t.tris[i]
				if dist, ok := hitTri(tr, o, d, inv, best); ok && (dist < best || dist == best && tr.idx < bestIdx) {
					best, kind, bestIdx = dist, tr.kind, tr.idx
				}
			}
			continue
		}
		// Visit the child nearer along the ray first, so the best hit shrinks early and prunes more.
		if vecComp(d, nd.axis) >= 0 {
			stack[n], stack[n+1] = nd.right, nd.left
		} else {
			stack[n], stack[n+1] = nd.left, nd.right
		}
		n += 2
	}
	return best, kind
}
