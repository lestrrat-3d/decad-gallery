package sevensegment_test

import (
	"cmp"
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
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
		if t.hi.X < cx-26 || t.lo.X > cx+26 || t.hi.Y < cy-26 || t.lo.Y > cy+26 {
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
		return 0, 33, true
	case sevensegment.SegD:
		return 0, -33, true
	case sevensegment.SegG:
		return 0, 0, true
	case sevensegment.SegB:
		return 16.5, 16.5, false
	case sevensegment.SegC:
		return 16.5, -16.5, false
	case sevensegment.SegE:
		return -16.5, -16.5, false
	}
	return -16.5, 16.5, false
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
	for u := -9.9; u <= 9.9; u += step {
		for sv := -2.9; sv <= 2.9; sv += step { // inside the window (half-width 2.95)
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

// TestSevenSegmentVisibility casts rays at the real meshes in the physical geometry (clamped and settled: the cradle and
// the flags 0.25 lower and the flags a further 0.25 lower, the top plate 0.50 lower) with every flag turned to the band
// ends and to the nominal rests. Lit, across the flag, the flag fills at least 90 / 55 / 20 % of the rays that enter at
// |u| <= 9.9 at elevations 90 / 45 / 30 degrees and yellow at least 65 % of those; unlit, no ray hits a yellow body. Azimuths
// along u and the whole-window share are reported, not asserted.
func TestSevenSegmentVisibility(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	topMesh, _ := meshOf(t, parts.TopPlate, 0.05)
	// The top plate and the flags sit 0.50 lower than modeled: the flags' 0.25 clamp and 0.25 settle against the plate's 0.50.
	shift := r3.Vec{Z: -0.5}
	faceZ := sevensegment.FaceZ - 0.5
	plate := trisOf(topMesh, nil, shift, kindPlate)
	type pose struct {
		name string
		lit  bool
		phis []float64
	}
	poses := []pose{
		{"lit", true, []float64{29.39, 45, 63.12}},
		{"unlit", false, []float64{-66.7, -55, -41.22, -37.90}},
	}
	elevs := []float64{90, 45, 30}
	litFlag := map[float64]float64{90: 0.90, 45: 0.55, 30: 0.20}
	for _, s := range sevensegment.Segments() {
		yMesh, _ := meshOf(t, parts.Flags[s].Yellow, 0.05)
		bMesh, _ := meshOf(t, parts.Flags[s].Black, 0.05)
		point, axis := segmentAxis(s)
		sAxis := r3.Vec{Z: 1}.Cross(axis) // s = Z x u
		uAxis := axis
		for _, ps := range poses {
			for _, phi := range ps.phis {
				turn, err := r3.RotationAround(point, axis, units.Degrees(phi-dm.FlagAngle(s, 0)))
				require.NoError(t, err)
				tris := append(append([]rayTri{}, plate...), trisOf(yMesh, &turn, shift, kindYellow)...)
				tris = append(tris, trisOf(bMesh, &turn, shift, kindBlack)...)
				tree := newRayTree(windowNear(tris, s))
				for _, elev := range elevs {
					for _, dir := range []r3.Vec{sAxis, sAxis.Scale(-1)} {
						flag, yellow := windowShares(tree, s, faceZ, elev, dir)
						if ps.lit {
							require.GreaterOrEqual(t, flag, litFlag[elev], "segment %s lit phi %.1f elevation %.0f flag share", s, phi, elev)
							require.GreaterOrEqual(t, yellow, 0.65, "segment %s lit phi %.1f elevation %.0f yellow share", s, phi, elev)
						} else {
							require.Zero(t, yellow, "segment %s unlit phi %.1f elevation %.0f shows yellow", s, phi, elev)
						}
					}
					if s == sevensegment.SegG && ps.lit && phi == 45 {
						flag, yellow := windowShares(tree, s, faceZ, elev, uAxis)
						t.Logf("segment g lit phi 45 elevation %.0f along +u: flag %.2f yellow %.2f (reported, not asserted)", elev, flag, yellow)
					}
				}
			}
		}
	}
}

// TestSevenSegmentWindowLip measures the plate's lip under the window edge of segment g, on the plate mesh, by a ray
// going up from inside the bore 0.01 outside the edge: 0.704 along the long sides (half-width 2.95), and 0.49 at the
// shoulders of the pointed ends. Both are under 0.8 and listed as an exception in instructions.md.
func TestSevenSegmentWindowLip(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	topMesh, _ := meshOf(t, parts.TopPlate, 0.01)
	plate := trisOf(topMesh, nil, r3.Vec{}, kindPlate)
	var near []rayTri
	for _, tr := range plate {
		if tr.hi.X > -14 && tr.lo.X < 14 && tr.hi.Y > -6 && tr.lo.Y < 6 && tr.hi.Z > 82 {
			near = append(near, tr)
		}
	}
	const windowHalf, windowChamfer, windowEnd = 2.95, 12.65, 11.696
	long, shoulder := math.Inf(1), math.Inf(1)
	for u := 0.0; u <= windowEnd; u += 0.01 {
		sv := math.Min(windowHalf, windowChamfer-u) + 0.01
		dist, kind := firstHitDist(near, r3.Vec{X: u, Y: sv, Z: 83}, r3.Vec{X: 1e-9, Y: 1e-9, Z: 1})
		require.Equal(t, kindPlate, kind, "u %.2f", u)
		lip := sevensegment.FaceZ - (83 + dist)
		if u < 9.0 {
			long = math.Min(long, lip)
		} else if u >= 9.7 && u <= 10.7 {
			shoulder = math.Min(shoulder, lip)
		}
	}
	t.Logf("lip along the long sides %.3f, at the shoulders %.3f", long, shoulder)
	require.InDelta(t, 0.704, long, 0.02)
	require.InDelta(t, 0.49, shoulder, 0.02)
}

// openingEntries returns the plan points of segment s's window opening on a 0.05 grid over |s| <= 4 and |u| <= 13: those where
// a ray going straight down from above the display face passes the face plane, that is, where the plate has no material.
// The opening is read from the plate mesh, so a window that is cut wider than designed shows more of the rotor.
func openingEntries(plate *rayTree, s sevensegment.Segment, faceZ float64) [][2]float64 {
	const step = 0.05
	cx, cy, horizontal := segmentCentre(s)
	var out [][2]float64
	// The grid is offset by half a step, so that no ray lies in the plane u = +-10 or s = 0: those planes hold the mesh edges of
	// both bodies and a ray in them can slip through a crack between triangles and report yellow that is not there.
	for u := -13.0 + step/2; u <= 13.0; u += step {
		for sv := -4.0 + step/2; sv <= 4.0; sv += step {
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

// TestSevenSegmentWindowYellow casts rays through the window opening of a type A rotor (segment g) and a type B rotor
// (segment b) in the model pose (the flag and the plate move together in the physical geometry). The opening is read from
// the plate mesh on a 0.05 grid over |s| <= 4, offset by half a step, so a wider window shows more.
//
// Lit, the yellow fills at least 0.95 of the opening at 90 degrees at phi 45, 0.82 at the lit band ends 34.47 (a, g, d) and
// 33.41 (b to f) and 0.76 at the b-f worst-case end 29.39 (computed 0.98, 0.85 to 0.87, 0.80).
//
// Unlit, no ray hits yellow at the design-play band end (a, g, d -40.33, b to f -41.22), the worst-case end (-39.23, -37.90)
// and a 1 degree margin beyond it (-38.23, -36.90), with the rotor centred, 0.16 or 0.25 to either side and 0.30 along the
// axis either way, from elevations 90, 45 and 30 at four azimuths. At the margin poses with the 0.25 side shift the views
// also include elevations 60 and 20 at 24 azimuths, grazing views included: a yellow face seen edge-on still counts.
func TestSevenSegmentWindowYellow(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	topMesh, _ := meshOf(t, parts.TopPlate, 0.05)
	plate := trisOf(topMesh, nil, r3.Vec{}, kindPlate)
	for _, c := range []struct {
		seg    sevensegment.Segment
		unlit  [3]float64 // design end, worst-case end, 1 degree margin
		lit    map[float64]float64
		litEnd string
	}{
		{sevensegment.SegG, [3]float64{-40.33, -39.23, -38.23}, map[float64]float64{45: 0.95, 34.47: 0.82, 33.20: 0.82}, "a, g, d"},
		{sevensegment.SegB, [3]float64{-41.22, -37.90, -36.90}, map[float64]float64{45: 0.95, 33.41: 0.82, 29.39: 0.76}, "b to f"},
	} {
		s := c.seg
		yMesh, _ := meshOf(t, parts.Flags[s].Yellow, 0.02)
		bMesh, _ := meshOf(t, parts.Flags[s].Black, 0.02)
		point, axis := segmentAxis(s)
		sAxis := r3.Vec{Z: 1}.Cross(axis)
		cx, cy := point.X, point.Y
		var near []rayTri
		for _, tr := range plate {
			if tr.hi.X < cx-26 || tr.lo.X > cx+26 || tr.hi.Y < cy-26 || tr.lo.Y > cy+26 || tr.hi.Z < 82 {
				continue
			}
			near = append(near, tr)
		}
		entries := openingEntries(newRayTree(near), s, sevensegment.FaceZ)
		// The opening, read from the mesh, is the designed window: 5.9 wide, 130.0 mm2.
		maxS := 0.0
		for _, q := range entries {
			maxS = math.Max(maxS, math.Abs((r3.Vec{X: q[0], Y: q[1]}).Sub(r3.Vec{X: cx, Y: cy}).Dot(sAxis)))
		}
		require.InDelta(t, 2.95, maxS, 0.03, "segment %s: the window's half-width on the mesh", s)
		require.InDelta(t, 130.0, float64(len(entries))*0.0025, 1.0, "segment %s: the window's area on the mesh", s)
		t.Logf("segment %s (%s): the opening holds %d entry points (%.1f mm2)", s, c.litEnd, len(entries), float64(len(entries))*0.0025)
		pose := func(phi, ds, du float64) []rayTri {
			turn, err := r3.RotationAround(point, axis, units.Degrees(phi-dm.FlagAngle(s, 0)))
			require.NoError(t, err)
			shift := sAxis.Scale(ds).Add(axis.Scale(du))
			tris := append([]rayTri{}, near...)
			tris = append(tris, trisOf(yMesh, &turn, shift, kindYellow)...)
			return indexTris(append(tris, trisOf(bMesh, &turn, shift, kindBlack)...))
		}
		for phi, want := range c.lit {
			tris := pose(phi, 0, 0)
			share := 0.0
			for _, dir := range []r3.Vec{sAxis, sAxis.Scale(-1)} {
				share = math.Max(share, float64(windowRays(tris, entries, sevensegment.FaceZ, 90, dir))/float64(len(entries)))
			}
			t.Logf("segment %s lit phi %.2f: yellow share of the opening %.3f", s, phi, share)
			require.GreaterOrEqual(t, share, want, "segment %s lit phi %.2f", s, phi)
		}
		type job struct {
			phi, ds, du float64
			wide        bool // elevations 90, 60, 45, 30 and 20 at 24 azimuths
		}
		var jobs []job
		for pi, phi := range c.unlit {
			for _, ds := range []float64{0, 0.16, -0.16, 0.25, -0.25} {
				for _, du := range []float64{0, 0.30, -0.30} {
					jobs = append(jobs, job{phi, ds, du, pi == 2 && math.Abs(ds) == 0.25})
				}
			}
		}
		counts := make([]int, len(jobs))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 20)
		for i, j := range jobs {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				tris := pose(j.phi, j.ds, j.du)
				elevs, step := []float64{90, 45, 30}, 90.0
				if j.wide {
					elevs, step = []float64{90, 60, 45, 30, 20}, 15.0
				}
				for _, elev := range elevs {
					for az := 0.0; az < 360; az += step {
						a := az * math.Pi / 180
						dir := sAxis.Scale(math.Cos(a)).Add(axis.Scale(math.Sin(a)))
						counts[i] += windowRays(tris, entries, sevensegment.FaceZ, elev, dir)
					}
				}
			}()
		}
		wg.Wait()
		for i, j := range jobs {
			require.Zero(t, counts[i], "segment %s unlit phi %.2f with side shift %+.2f and axial shift %+.2f: yellow rays", s, j.phi,
				j.ds, j.du)
		}
	}
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
