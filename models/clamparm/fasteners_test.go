package clamparm_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const (
	asmPartTol     = 0.05 // chord tolerance of the printed parts' meshes (some bodies' held meshes bound 0.021)
	asmHardwareTol = 0.01 // chord tolerance of the hardware's meshes
	asmPlaneTol    = 0.01 // a vertex lies on a plane within this distance
	asmHexCorner   = 4.6188
	asmHexAngle    = 1.0 // degrees: a hexagon's corners lie on 60 degree steps from the reference direction
	asmKnobNutDep  = 9.0
	asmAdptNutDep  = 10.3
)

// asmSite is a bolt site moved into the case's world frame.
type asmSite struct {
	head, dir, corner r3.Vec
	length            float64
}

func (s asmSite) tip() float64 { return m5fastener.HeadPocketDepth + s.length }

func asmWorldSite(xf r3.Transform, s m5fastener.Site) asmSite {
	return asmSite{head: xf.Apply(s.Head), dir: xf.ApplyDir(s.Dir), corner: xf.ApplyDir(s.Corner), length: s.Length}
}

// asmAxis measures vertices against an axis: along is the distance along the axis from its origin and lateral the
// distance from the axis line. angle is the direction angle of the vertex round the axis from ref, in degrees.
type asmAxis struct{ origin, dir, ref r3.Vec }

func (a asmAxis) along(v r3.Vec) float64 { return v.Sub(a.origin).Dot(a.dir) }

func (a asmAxis) lateral(v r3.Vec) float64 {
	d := v.Sub(a.origin)
	return d.Sub(a.dir.Scale(d.Dot(a.dir))).Len()
}

func (a asmAxis) angle(v r3.Vec) float64 {
	d := v.Sub(a.origin)
	perp := d.Sub(a.dir.Scale(d.Dot(a.dir)))
	second := a.dir.Cross(a.ref)
	return math.Atan2(perp.Dot(second), perp.Dot(a.ref)) * 180 / math.Pi
}

// requireOnHexagon requires the vertex's angle to be a multiple of 60 degrees from the axis reference.
func requireOnHexagon(t *testing.T, a asmAxis, v r3.Vec, what string) {
	t.Helper()
	off := math.Mod(math.Abs(a.angle(v)), 60)
	off = math.Min(off, 60-off)
	require.Less(t, off, asmHexAngle, "%s corner is off the hexagon's corner directions at %v", what, v)
}

// asmPlaneVertices lists the mesh vertices on the plane along = at with lateral distance in [lo, hi].
func asmPlaneVertices(mesh *decad.Mesh, a asmAxis, at, lo, hi float64) []r3.Vec {
	var out []r3.Vec
	for _, v := range mesh.Vertices() {
		l := a.lateral(v)
		if math.Abs(a.along(v)-at) <= asmPlaneTol && l >= lo && l <= hi {
			out = append(out, v)
		}
	}
	return out
}

// boltPocketPart names the printed part whose pocket holds each bolt's head.
func boltPocketPart(m *clamparm.Model, i int) clamparm.Part {
	return [6]clamparm.Part{m.Clamp, m.Swivel, m.Links[0], m.Links[1], m.ScrewKnobs[1], m.ScrewKnobs[0]}[i]
}

// boltTerminalPart is the printed part past the bolt's tip along its direction.
func boltTerminalPart(m *clamparm.Model, i int) clamparm.Part {
	return [6]clamparm.Part{m.Knobs[0], m.Knobs[1], m.Knobs[2], m.Knobs[3], m.Adapter, m.Pad}[i]
}

// nutPocketPart is the printed part whose pocket holds each nut, and the pocket's depth from its floor.
func nutPocketPart(m *clamparm.Model, i int) (clamparm.Part, float64) {
	parts := [6]clamparm.Part{m.Knobs[0], m.Knobs[1], m.Knobs[2], m.Knobs[3], m.Clamp, m.Adapter}
	depths := [6]float64{asmKnobNutDep, asmKnobNutDep, asmKnobNutDep, asmKnobNutDep, m5fastener.NutCavityHeight,
		asmAdptNutDep}
	return parts[i], depths[i]
}

// asmWorldSites lists the six bolt sites in model order, in the case's world frame.
func asmWorldSites(t testing.TB, c asmCase, d clamparm.Dimensions) [6]asmSite {
	t.Helper()
	xf := asmWorld(t, c.params)
	return [6]asmSite{
		asmWorldSite(xf, d.PanSite), asmWorldSite(xf, d.JointSites[0]), asmWorldSite(xf, d.JointSites[1]),
		asmWorldSite(xf, d.JointSites[2]), asmWorldSite(xf, d.StudSite), asmWorldSite(xf, d.ScrewSite),
	}
}

// asmWorldNutFrames lists the six nut frames in model order, in the case's world frame.
func asmWorldNutFrames(t testing.TB, c asmCase, d clamparm.Dimensions) [6]asmAxis {
	t.Helper()
	xf := asmWorld(t, c.params)
	frames := [6]r3.Frame{d.KnobNutFrames[0], d.KnobNutFrames[1], d.KnobNutFrames[2], d.KnobNutFrames[3],
		d.JawNutFrame, d.AdapterNutFrame}
	var out [6]asmAxis
	for i, f := range frames {
		out[i] = asmAxis{origin: xf.Apply(f.Origin()), dir: xf.ApplyDir(f.N()), ref: xf.ApplyDir(f.U())}
	}
	return out
}

// TestClampArmFastenersBoltHeads requires each bolt head to end flush with the face of the part it joins and to sit
// inside its pocket, and each tip to stay inside a face.
func TestClampArmFastenersBoltHeads(t *testing.T) {
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, m := asmBuild(t, c)
			sites := asmWorldSites(t, c, m.Dimensions)
			_, bolts := asmBolts(m)
			names, _ := asmBolts(m)
			for i, s := range sites {
				a := asmAxis{origin: s.head, dir: s.dir, ref: s.corner}
				bolt := asmTessellate(t, bolts[i], asmHardwareTol)
				pocket := asmTessellate(t, boltPocketPart(m, i).Body, asmPartTol)

				// The head's top face is at the site and its underside on the pocket floor, 3.5 below.
				lowest, highest, headReach := math.Inf(1), math.Inf(-1), 0.0
				for _, v := range bolt.Vertices() {
					along := a.along(v)
					lowest, highest = math.Min(lowest, along), math.Max(highest, along)
					if along < m5fastener.HeadPocketDepth-asmPlaneTol {
						headReach = math.Max(headReach, a.lateral(v))
						requireOnHexagonIfCorner(t, a, v, names[i])
					}
				}
				require.InDelta(t, 0, lowest, asmPlaneTol, names[i])
				require.InDelta(t, s.tip(), highest, asmPlaneTol, names[i])
				// The head's corners stay inside the pocket's corner circle: 4.6188 against 4.9188 or 4.6688.
				require.InDelta(t, asmHexCorner, headReach, 0.01, names[i])

				// The part's face is flush with the head: its pocket rim, the corners of a hexagon, lies on the
				// plane through the head's top face.
				wantCorner := m5fastener.HeadPocketRadius
				if i >= 4 {
					wantCorner = m5fastener.PressPocketRadius
				}
				rim := asmPlaneVertices(pocket, a, 0, wantCorner-0.02, wantCorner+0.02)
				require.GreaterOrEqual(t, len(rim), 6, "%s: pocket rim is not flush with the head", names[i])
				for _, v := range rim {
					requireOnHexagon(t, a, v, names[i])
				}
				// The pocket's floor is the head's underside: the part has its floor boundary on that plane.
				floor := asmPlaneVertices(pocket, a, m5fastener.HeadPocketDepth, 2.7, 4.8)
				require.NotEmpty(t, floor, "%s: no pocket floor under the head", names[i])
				// No part vertex between the face and the floor lies inside the head's corner circle.
				for _, v := range pocket.Vertices() {
					along := a.along(v)
					if along > asmPlaneTol && along < m5fastener.HeadPocketDepth-asmPlaneTol {
						require.Greater(t, a.lateral(v), 2.7, "%s: part inside the head's span at %v", names[i], v)
					}
				}

				// The tip stays inside the terminal part: below its outer face, which is the highest vertex round
				// the axis (a nut pocket's rim or the pad socket's floor).
				terminal := asmTessellate(t, boltTerminalPart(m, i).Body, asmPartTol)
				outer := math.Inf(-1)
				for _, v := range terminal.Vertices() {
					if a.lateral(v) <= 5.1 {
						outer = math.Max(outer, a.along(v))
					}
				}
				require.LessOrEqual(t, s.tip(), outer+asmPlaneTol, "%s tip stands outside its terminal part", names[i])
			}
		})
	}
}

// requireOnHexagonIfCorner checks the direction of a head vertex that lies on the corner circle.
func requireOnHexagonIfCorner(t *testing.T, a asmAxis, v r3.Vec, what string) {
	t.Helper()
	if a.lateral(v) > asmHexCorner-0.02 {
		requireOnHexagon(t, a, v, what)
	}
}

// TestClampArmFastenersNuts requires each nut to sit on its pocket's floor, inside its pocket, below the face.
func TestClampArmFastenersNuts(t *testing.T) {
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, m := asmBuild(t, c)
			frames := asmWorldNutFrames(t, c, m.Dimensions)
			names, nuts := asmNuts(m)
			for i, a := range frames {
				part, depth := nutPocketPart(m, i)
				nut := asmTessellate(t, nuts[i], asmHardwareTol)
				pocket := asmTessellate(t, part.Body, asmPartTol)

				lowest, highest, reach := math.Inf(1), math.Inf(-1), 0.0
				for _, v := range nut.Vertices() {
					along := a.along(v)
					lowest, highest = math.Min(lowest, along), math.Max(highest, along)
					reach = math.Max(reach, a.lateral(v))
					if a.lateral(v) > asmHexCorner-0.02 {
						requireOnHexagon(t, a, v, names[i])
					}
				}
				// The nut's lower face is on the pocket floor and its upper face 4.7 above it.
				require.InDelta(t, 0, lowest, asmPlaneTol, names[i])
				require.InDelta(t, m5fastener.NutSize.Thickness, highest, asmPlaneTol, names[i])
				require.InDelta(t, asmHexCorner, reach, 0.01, names[i])
				// It ends below the face: the pocket is deeper than the nut is thick.
				require.Less(t, highest, depth-asmPlaneTol, names[i])

				// The pocket's floor is under the nut and its rim, a hexagon on the corner circle 0.2 past the
				// nut's corners, stands at the face above it.
				floor := asmPlaneVertices(pocket, a, 0, 2.7, 4.8)
				require.NotEmpty(t, floor, "%s: no pocket floor under the nut", names[i])
				rim := asmPlaneVertices(pocket, a, depth, m5fastener.NutPocketRadius-0.02, m5fastener.NutPocketRadius+0.02)
				require.GreaterOrEqual(t, len(rim), 6, "%s: pocket is not open at its face", names[i])
				for _, v := range rim {
					requireOnHexagon(t, a, v, names[i])
				}
			}
		})
	}
}

// TestClampArmFastenersBoltNutEngagement checks each bolt against its nut. The nut's bore has the shank's diameter,
// so decad's Intersect refuses the pair (coincident cylinders); the check is analytic: the nut is coaxial with the
// bolt, square to it, and between the head's underside and the tip.
func TestClampArmFastenersBoltNutEngagement(t *testing.T) {
	nutOf := [6]int{0, 1, 2, 3, 5, 4}
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, m := asmBuild(t, c)
			sites := asmWorldSites(t, c, m.Dimensions)
			frames := asmWorldNutFrames(t, c, m.Dimensions)
			names, _ := asmBolts(m)
			for i, s := range sites {
				n := frames[nutOf[i]]
				axis := asmAxis{origin: s.head, dir: s.dir, ref: s.corner}
				require.InDelta(t, 0, axis.lateral(n.origin), 1e-6, names[i])
				require.InDelta(t, 1, math.Abs(n.dir.Dot(s.dir)), 1e-9, names[i])
				low := axis.along(n.origin)
				require.GreaterOrEqual(t, low, m5fastener.HeadPocketDepth, names[i])
				require.LessOrEqual(t, low+m5fastener.NutSize.Thickness, s.tip(), names[i])
			}
		})
	}
}

// asmPair is a hardware body against a body it touches or passes.
type asmPair struct{ a, b string }

// asmRefusal is a pair in a case.
type asmRefusal struct{ caseName, hardware, other string }

// asmMeshOnly lists the pairs decad's Intersect still refuses in both operand orders on the pinned decad, each with
// its reason (steps.md section 13 item 5). Every other pair must prove empty.
var asmMeshOnly = map[asmRefusal]string{
	{"pan-max", "bolt J2", "swivel"}: "the swivel's held mesh is coarser than the pair's chord tolerance",
	{"pan-min", "bolt J2", "swivel"}: "the swivel's held mesh is coarser than the pair's chord tolerance",
	{"raised", "nut J4", "knob J4"}:  "the held facets come within the chord tolerance",
}

// TestClampArmFastenersNoOverlap requires every hardware body to share no volume with any other body, using decad's
// Intersect and its BooleanEmpty result. Bodies whose boxes are apart are disjoint without a call. A bolt and its own
// nut are left out (see TestClampArmFastenersBoltNutEngagement). Every pair must prove empty in one operand order,
// except the pairs in asmMeshOnly, which decad still refuses in both orders and which asmMeshOverlap checks instead.
func TestClampArmFastenersNoOverlap(t *testing.T) {
	for _, c := range asmCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m := asmBuildFlat(t, c)
			printedNames, printed := asmPrinted(m)
			boltNames, bolts := asmBolts(m)
			nutNames, nuts := asmNuts(m)
			var names []string
			var bodies []*decad.Body
			for i, p := range printed {
				names, bodies = append(names, printedNames[i]), append(bodies, p.Body)
			}
			firstHardware := len(names)
			names, bodies = append(names, boltNames...), append(bodies, bolts...)
			names, bodies = append(names, nutNames...), append(bodies, nuts...)
			own := map[asmPair]struct{}{}
			for i, n := range []int{0, 1, 2, 3, 5, 4} {
				own[asmPair{boltNames[i], nutNames[n]}] = struct{}{}
				own[asmPair{nutNames[n], boltNames[i]}] = struct{}{}
			}

			// Each hardware body's own axis, in the order of bodies, for the fallback's sections.
			axes := make([]asmAxis, len(bodies))
			for i, site := range asmWorldSites(t, c, m.Dimensions) {
				axes[firstHardware+i] = asmAxis{origin: site.head, dir: site.dir, ref: site.corner}
			}
			for i, f := range asmWorldNutFrames(t, c, m.Dimensions) {
				axes[firstHardware+len(boltNames)+i] = f
			}

			checked := 0
			var fallbacks []string
			for i := firstHardware; i < len(bodies); i++ {
				for j := range i {
					if _, ok := own[asmPair{names[i], names[j]}]; ok {
						continue
					}
					bi, err := bodies[i].Bounds()
					require.NoError(t, err)
					bj, err := bodies[j].Bounds()
					require.NoError(t, err)
					if !asmBoxesMeet(bi, bj, asmPlaneTol) {
						continue
					}
					checked++
					_, meshOnly := asmMeshOnly[asmRefusal{c.name, names[i], names[j]}]
					if !requireDisjointPair(t, names[i], names[j], bodies[i], bodies[j], axes[i], meshOnly) {
						fallbacks = append(fallbacks, names[i]+" and "+names[j])
					}
				}
			}
			if len(fallbacks) > 0 {
				t.Logf("Intersect refused these pairs in both operand orders; checked by asmMeshOverlap: %v", fallbacks)
			}
			// Every bolt meets its part, its knob or its nut's neighbours: at least these pairs were checked.
			require.GreaterOrEqual(t, checked, 24)
		})
	}
}

func asmBoxesMeet(a, b decad.Box, tol float64) bool {
	return a.Min.X <= b.Max.X+tol && b.Min.X <= a.Max.X+tol &&
		a.Min.Y <= b.Max.Y+tol && b.Min.Y <= a.Max.Y+tol &&
		a.Min.Z <= b.Max.Z+tol && b.Min.Z <= a.Max.Z+tol
}

// requireDisjointPair requires a, a hardware body, and b to share no volume. It asks decad's Intersect with the
// hardware first, then with the printed part first, and expects BooleanEmpty from one of them, which proves the
// bodies disjoint. A pair in asmMeshOnly (meshOnly) is the exception: decad refuses it in both orders, so the interim
// proof asmMeshOverlap, whose axis is the hardware's own, decides it and the function returns false. A pair not in
// asmMeshOnly that decad refuses fails.
func requireDisjointPair(t *testing.T, an, bn string, a, b *decad.Body, axis asmAxis, meshOnly bool) bool {
	t.Helper()
	var refusal error
	for _, operands := range [][2]*decad.Body{{a, b}, {b, a}} {
		_, err := decad.Intersect(t.Context(), operands[0], operands[1])
		require.Error(t, err, "%s and %s share volume", an, bn)
		var be *decad.BooleanError
		if errors.As(err, &be) && be.Code == decad.BooleanEmpty {
			return true
		}
		refusal = err
	}
	require.True(t, meshOnly, "%s and %s: decad refused the pair in both orders and it is not in asmMeshOnly: %v", an, bn,
		refusal)
	require.Empty(t, asmMeshOverlap(t, an, bn, a, b, axis), "%s and %s share volume", an, bn)
	return false
}

// asmSectionOutline cuts the body with the plane at the finest chord tolerance that the mesh section accepts.
func asmSectionOutline(t *testing.T, body *decad.Body, f r3.Frame) decad.SectionOutline {
	t.Helper()
	var o decad.SectionOutline
	var err error
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		if o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			return o
		}
	}
	require.NoError(t, err)
	return o
}

// asmMeshOverlap is the interim proof of disjointness for a hardware body a and a body b where decad's Intersect
// refuses the pair (steps.md section 13). It returns a description of the first overlap it finds, or "". It runs
// three tests, each of which alone would miss some overlap: asmVertexOverlap, asmNormalOverlap at seven stations
// along the axis and asmAxialOverlap.
func asmMeshOverlap(t *testing.T, an, bn string, a, b *decad.Body, axis asmAxis) string {
	t.Helper()
	if why := asmVertexOverlap(t, an, bn, a, b); why != "" {
		return why
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range asmTessellate(t, a, asmHardwareTol).Vertices() {
		lo, hi = math.Min(lo, axis.along(v)), math.Max(hi, axis.along(v))
	}
	var stations []float64
	for i := range 7 {
		stations = append(stations, lo+0.3+(hi-lo-0.6)*float64(i)/6)
	}
	if why := asmNormalOverlap(t, an, bn, a, b, axis, stations); why != "" {
		return why
	}
	return asmAxialOverlap(t, an, bn, a, b, axis)
}

// asmDeepInside is the depth below which a vertex that a ray says is inside counts as on the surface: a bolt's head
// rests on its pocket floor, so its underside vertices lie on the part's surface, where a ray's parity is unreliable.
const asmDeepInside = 0.01

// asmVertexOverlap reports a vertex of a's mesh deeper than asmDeepInside inside b's mesh, or a vertex of b's mesh
// inside a's (a hole offset into a shank puts hole-wall vertices inside the shank). decad's LocatePoint answers
// Undecided for points well inside a bolt, so the test casts a ray against the mesh itself (asmInsideMesh).
func asmVertexOverlap(t *testing.T, an, bn string, a, b *decad.Body) string {
	t.Helper()
	ma, mb := asmTessellate(t, a, asmHardwareTol), asmTessellate(t, b, asmPartTol)
	ba, err := a.Bounds()
	require.NoError(t, err)
	bb, err := b.Bounds()
	require.NoError(t, err)
	inside := func(of *decad.Mesh, v r3.Vec, tree *meshgap.Mesh, ready *bool) bool {
		if !asmInsideMesh(of, v) {
			return false
		}
		if !*ready {
			var err error
			*tree, err = meshgap.New(of.Vertices(), of.Triangles(), of.Bound().Base())
			require.NoError(t, err)
			*ready = true
		}
		d, err := tree.PointDistance(v)
		require.NoError(t, err)
		return d > asmDeepInside
	}
	var mbTree, maTree meshgap.Mesh
	var mbTreeReady, maTreeReady bool
	for _, v := range ma.Vertices() {
		if asmBoxesMeet(decad.Box{Min: v, Max: v}, bb, asmPlaneTol) && inside(mb, v, &mbTree, &mbTreeReady) {
			return fmt.Sprintf("%s vertex %v is inside %s", an, v, bn)
		}
	}
	for _, v := range mb.Vertices() {
		if asmBoxesMeet(decad.Box{Min: v, Max: v}, ba, asmPlaneTol) && inside(ma, v, &maTree, &maTreeReady) {
			return fmt.Sprintf("%s vertex %v is inside %s", bn, v, an)
		}
	}
	return ""
}

// asmInsideMesh reports whether the point is inside the closed mesh: a ray from it, along a direction that avoids the
// mesh's edges, crosses an odd number of triangles.
func asmInsideMesh(mesh *decad.Mesh, p r3.Vec) bool {
	dir := r3.NewVec(0.5773, 0.3137, 0.7541)
	vs := mesh.Vertices()
	crossings := 0
	for _, tri := range mesh.Triangles() {
		v0, v1, v2 := vs[tri[0]], vs[tri[1]], vs[tri[2]]
		e1, e2 := v1.Sub(v0), v2.Sub(v0)
		h := dir.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < 1e-12 {
			continue
		}
		s := p.Sub(v0)
		u := s.Dot(h) / det
		if u < 0 || u > 1 {
			continue
		}
		q := s.Cross(e1)
		v := dir.Dot(q) / det
		if v < 0 || u+v > 1 {
			continue
		}
		if e2.Dot(q)/det > 1e-9 {
			crossings++
		}
	}
	return crossings%2 == 1
}

// asmSectionsOverlap reports whether the outlines of a and b in the plane f overlap: whether a vertex of one outline
// lies deeper than asmDeepInside inside the other (by parity over its loops, with holes). It does not use
// SectionClearance.Overlaps: where the plane meets a face's rim, as it does at a bolt's pocket floor or a hole's edge
// on a rotated part, the section holds zero-width spikes that make that flag true for outlines 0.3 mm apart.
func asmSectionsOverlap(t *testing.T, a, b *decad.Body, f r3.Frame) bool {
	t.Helper()
	oa, ob := asmSectionOutline(t, a, f), asmSectionOutline(t, b, f)
	if len(oa.Loops) == 0 || len(ob.Loops) == 0 {
		return false
	}
	flat := func(o decad.SectionOutline) [][][2]float64 {
		out := make([][][2]float64, len(o.Loops))
		for i, l := range o.Loops {
			for _, p := range l {
				d := p.Sub(f.Origin())
				out[i] = append(out[i], [2]float64{d.Dot(f.U()), d.Dot(f.V())})
			}
		}
		return out
	}
	la, lb := flat(oa), flat(ob)
	deepIn := func(points, loops [][][2]float64) bool {
		for _, l := range points {
			for _, p := range l {
				if asmInsideLoops(p, loops) && asmLoopsDistance(p, loops) > asmDeepInside {
					return true
				}
			}
		}
		return false
	}
	return deepIn(la, lb) || deepIn(lb, la)
}

// asmInsideLoops reports whether p is inside the region the closed loops bound: a ray along +x crosses an odd number
// of their segments.
func asmInsideLoops(p [2]float64, loops [][][2]float64) bool {
	in := false
	for _, l := range loops {
		for i, q := range l {
			r := l[(i+1)%len(l)]
			if (q[1] > p[1]) != (r[1] > p[1]) && p[0] < q[0]+(p[1]-q[1])/(r[1]-q[1])*(r[0]-q[0]) {
				in = !in
			}
		}
	}
	return in
}

// asmLoopsDistance is the distance from p to the nearest segment of the loops.
func asmLoopsDistance(p [2]float64, loops [][][2]float64) float64 {
	best := math.Inf(1)
	for _, l := range loops {
		for i, q := range l {
			r := l[(i+1)%len(l)]
			dx, dy := r[0]-q[0], r[1]-q[1]
			u := 0.0
			if n := dx*dx + dy*dy; n > 0 {
				u = math.Max(0, math.Min(1, ((p[0]-q[0])*dx+(p[1]-q[1])*dy)/n))
			}
			best = math.Min(best, math.Hypot(p[0]-q[0]-u*dx, p[1]-q[1]-u*dy))
		}
	}
	return best
}

// asmNormalOverlap cuts a and b by planes normal to the axis at the given distances along it and reports an overlap
// of the two sections (a shank's circle against its hole's circle, a hexagon against its pocket).
func asmNormalOverlap(t *testing.T, an, bn string, a, b *decad.Body, axis asmAxis, stations []float64) string {
	t.Helper()
	side := axis.dir.Cross(axis.ref)
	for _, at := range stations {
		f, err := r3.NewFrame(axis.origin.Add(axis.dir.Scale(at)), axis.ref, side)
		require.NoError(t, err)
		if asmSectionsOverlap(t, a, b, f) {
			return fmt.Sprintf("%s and %s overlap in the section normal to the axis at %.2f", an, bn, at)
		}
	}
	return ""
}

// asmAxialOverlap cuts a and b by planes through the axis at 0, 45, 90 and 135 degrees round it and reports an overlap
// of the two sections (the shank's strip against the hole's wall, as in crankhandle's section checks).
func asmAxialOverlap(t *testing.T, an, bn string, a, b *decad.Body, axis asmAxis) string {
	t.Helper()
	side := axis.dir.Cross(axis.ref)
	for _, deg := range []float64{0, 45, 90, 135} {
		rad := deg * math.Pi / 180
		radial := axis.ref.Scale(math.Cos(rad)).Add(side.Scale(math.Sin(rad)))
		f, err := r3.NewFrame(axis.origin, axis.dir, radial)
		require.NoError(t, err)
		if asmSectionsOverlap(t, a, b, f) {
			return fmt.Sprintf("%s and %s overlap in the section through the axis at %g degrees", an, bn, deg)
		}
	}
	return ""
}

// TestClampArmFastenersMeshOverlapControl requires each test of the interim proof asmMeshOverlap to catch a bolt that
// cuts into its link's tongue hole by more than the 0.3 mm clearance: the J2 bolt through link A's tongue, moved
// 0.6 along its corner direction and 0.35 along a diagonal (0.25 along each of two axes, where an axial section at
// 0 or 90 degrees sees only 0.25). The section normal to the axis is taken in the shank's middle, in the tongue.
// A bolt moved 0.1 stays clear. The honest bolt passes the whole proof.
func TestClampArmFastenersMeshOverlapControl(t *testing.T) {
	c := asmCases(t)[0]
	m := asmBuildFlat(t, c)
	site := asmWorldSites(t, c, m.Dimensions)[1]
	axis := asmAxis{origin: site.head, dir: site.dir, ref: site.corner}
	side := site.dir.Cross(site.corner)
	tongue := m.Links[0].Body
	require.Empty(t, asmMeshOverlap(t, "bolt J2", "link A", m.Bolts[1].Body, tongue, axis))
	// The tongue's middle: 3.5 head, the fin 5.74 and the 0.26 fit, then half the tongue.
	middle := 5.74 + 0.26 + 4.0

	doc := clamparm.NewDocument()
	for _, tc := range []struct {
		name  string
		shift r3.Vec
		hit   bool
	}{
		{"along the corner", site.corner.Scale(0.6), true},
		{"along a diagonal", site.corner.Scale(0.25).Add(side.Scale(0.25)), true},
		{"within the clearance", site.corner.Scale(0.1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			moved := m5fastener.Site{Head: site.head.Add(tc.shift), Dir: site.dir, Corner: site.corner, Length: site.length}
			bolt, err := m5fastener.PlaceBolt(t.Context(), doc, moved)
			require.NoError(t, err)
			vertex := asmVertexOverlap(t, "offset bolt", "link A", bolt.Body, tongue)
			normal := asmNormalOverlap(t, "offset bolt", "link A", bolt.Body, tongue, axis, []float64{middle})
			axial := asmAxialOverlap(t, "offset bolt", "link A", bolt.Body, tongue, axis)
			t.Logf("vertex %q normal %q axial %q", vertex, normal, axial)
			if !tc.hit {
				require.Empty(t, vertex+normal+axial)
				return
			}
			require.NotEmpty(t, normal, "the shank section misses an overlap of %v", tc.shift)
			require.NotEmpty(t, axial, "the axial sections miss an overlap of %v", tc.shift)
			require.NotEmpty(t, vertex, "the vertex test misses an overlap of %v", tc.shift)
		})
	}
}

// TestClampArmFastenersPressHeadClearance checks the two bolt heads pressed into screw knobs (the clamp screw and the
// stud) analytically. In the section through a flat of the hexagon, the head's underside is a 30 degree chamfer from
// the bearing circle to the flat, and the pocket floor is flat out to the floor round, a quarter circle. The distance
// between the two outlines must be at least 0.
func TestClampArmFastenersPressHeadClearance(t *testing.T) {
	flat := m5fastener.BoltSize.Hex.AcrossFlats / 2
	bearing := m5fastener.BoltSize.Hex.BearingDiameter / 2
	chamfer := math.Tan(fastener.HexChamferAngle * math.Pi / 180)
	wall := m5fastener.PressPocketRadius * math.Cos(math.Pi/6)
	round := m5fastener.PressPocketRound
	require.Greater(t, wall, flat, "the pocket's flat must clear the head's flat")
	require.GreaterOrEqual(t, wall-round, bearing, "the round must start beyond the bearing circle")

	headAt := func(r float64) float64 { return (r - bearing) * chamfer }
	// Head outline: the flat chamfer line from the bearing circle to the flat, then the head's flat face up to its
	// height. Floor outline: z = 0 to r = wall - round, then the round's arc, then the wall.
	floorAt := func(r float64) float64 {
		if r <= wall-round {
			return 0
		}
		dr := r - (wall - round)
		return round - math.Sqrt(math.Max(0, round*round-dr*dr))
	}
	minGap := math.Inf(1)
	const steps = 4000
	// Floor points against the head outline, and head points against the floor outline.
	for i := 0; i <= steps; i++ {
		r := bearing + (flat-bearing)*float64(i)/steps
		hp := [2]float64{r, headAt(r)}
		for j := 0; j <= steps; j++ {
			fr := (wall - round) + (round+0.01)*float64(j)/steps
			fp := [2]float64{fr, floorAt(math.Min(fr, wall))}
			minGap = math.Min(minGap, math.Hypot(hp[0]-fp[0], hp[1]-fp[1]))
		}
	}
	// The flat's face above the chamfer against the pocket wall.
	minGap = math.Min(minGap, wall-flat)
	require.GreaterOrEqual(t, minGap, 0.0, "the head's chamfer meets the floor round")
	// The design's figure: +0.010 mm at the 0.5 round.
	require.InDelta(t, 0.010, minGap, 0.003)
}

// TestClampArmFastenersPressHeadsFlush requires the clamp screw's and the stud's head to end flush with the outer
// face of its screw knob, inside the press pocket, in every case. It is part of TestClampArmFastenersBoltHeads; this
// test names the two pairs that rely on the analytic clearance test.
func TestClampArmFastenersPressHeadsFlush(t *testing.T) {
	_, m := asmBuild(t, asmCases(t)[0])
	for i, knob := range map[int]clamparm.Part{4: m.ScrewKnobs[1], 5: m.ScrewKnobs[0]} {
		bolt, err := m.Bolts[i].Body.Bounds()
		require.NoError(t, err)
		kb, err := knob.Body.Bounds()
		require.NoError(t, err)
		// Both axes are vertical in the default pose: the bolt head's top face is the knob's outer face.
		require.InDelta(t, kb.Min.Z, bolt.Min.Z, boundTol)
		// The head stands inside the knob's plan.
		require.Greater(t, bolt.Min.X, kb.Min.X)
		require.Less(t, bolt.Max.X, kb.Max.X)
		require.Greater(t, bolt.Min.Y, kb.Min.Y)
		require.Less(t, bolt.Max.Y, kb.Max.Y)
	}
}
