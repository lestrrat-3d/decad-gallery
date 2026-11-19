package clamparm_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

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
				require.InDelta(t, m5nut.Thickness, highest, asmPlaneTol, names[i])
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
				require.LessOrEqual(t, low+m5nut.Thickness, s.tip(), names[i])
			}
		})
	}
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
	flat := m5bolt.AcrossFlats / 2
	bearing := m5bolt.BearingDiameter / 2
	chamfer := math.Tan(m5bolt.ChamferAngle * math.Pi / 180)
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
