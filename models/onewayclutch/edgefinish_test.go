package onewayclutch_test

import (
	"cmp"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

const (
	// efDepth is how far inside a hub face the section is taken. A 45 degree chamfer of HubChamfer offsets the outline
	// by HubChamfer - efDepth there: 0.15 mm.
	efDepth = 0.1
	// efSharp and efOffset are how far a point of the mid-height outline may lie from a face section to read as
	// unchanged, and as offset by the chamfer.
	efSharp  = 0.05
	efOffset = 0.05
	// efStep is the spacing of the points sampled along the mid-height outline.
	efStep = 0.05
)

// efSection is the hub's outline at z, flattened to z = 0 so that outlines at different heights compare in the plane.
func efSection(t *testing.T, body *decad.Body, z float64) [][]r3.Vec {
	t.Helper()
	s := fitSection(t, body, fitPlane(t, z))
	for _, loop := range s.Loops {
		for i := range loop {
			loop[i].Z = 0
		}
	}
	return s.Loops
}

// efKind names a loop of the hub's outline.
type efKind int

const (
	efOuter efKind = iota // pawls, web and core
	efBore                // the D or round shaft bore
	efHole                // a bolt hole
)

func efKindOf(loop []r3.Vec) efKind {
	lo, hi := math.Inf(1), 0.0
	for _, p := range loop {
		r := math.Hypot(p.X, p.Y)
		lo, hi = math.Min(lo, r), math.Max(hi, r)
	}
	switch {
	case hi < 6:
		return efBore
	case lo > 6.5 && hi < 10:
		return efHole
	}
	return efOuter
}

// efShift is the length of a loop, in mm, that lies unchanged, offset by the chamfer, or neither, measured from the
// mid-height outline to the face section.
type efShift struct{ unchanged, offset, other float64 }

func (s efShift) total() float64 { return s.unchanged + s.offset + s.other }

type efSegment struct{ a, b r3.Vec }

// efNode stores a balanced bounding-volume tree over the face section's segments.
type efNode struct {
	lo, hi      r3.Vec
	left, right *efNode
	segments    []efSegment
}

func efIndex(loops [][]r3.Vec) *efNode {
	var segments []efSegment
	for _, loop := range loops {
		for i, a := range loop {
			segments = append(segments, efSegment{a, loop[(i+1)%len(loop)]})
		}
	}
	return efBuildIndex(segments)
}

func efBuildIndex(segments []efSegment) *efNode {
	if len(segments) == 0 {
		return nil
	}
	n := &efNode{lo: r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)},
		hi: r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}}
	for _, s := range segments {
		for _, p := range []r3.Vec{s.a, s.b} {
			n.lo.X, n.lo.Y, n.lo.Z = math.Min(n.lo.X, p.X), math.Min(n.lo.Y, p.Y), math.Min(n.lo.Z, p.Z)
			n.hi.X, n.hi.Y, n.hi.Z = math.Max(n.hi.X, p.X), math.Max(n.hi.Y, p.Y), math.Max(n.hi.Z, p.Z)
		}
	}
	// Expand the box by one representable value to keep its lower bound conservative under rounding.
	n.lo = r3.Vec{X: math.Nextafter(n.lo.X, math.Inf(-1)), Y: math.Nextafter(n.lo.Y, math.Inf(-1)), Z: math.Nextafter(n.lo.Z, math.Inf(-1))}
	n.hi = r3.Vec{X: math.Nextafter(n.hi.X, math.Inf(1)), Y: math.Nextafter(n.hi.Y, math.Inf(1)), Z: math.Nextafter(n.hi.Z, math.Inf(1))}
	if len(segments) <= 8 {
		n.segments = segments
		return n
	}
	span := n.hi.Sub(n.lo)
	axis := func(p r3.Vec) float64 { return p.X }
	if span.Y > span.X && span.Y >= span.Z {
		axis = func(p r3.Vec) float64 { return p.Y }
	} else if span.Z > span.X && span.Z > span.Y {
		axis = func(p r3.Vec) float64 { return p.Z }
	}
	slices.SortFunc(segments, func(a, b efSegment) int {
		return cmp.Compare(axis(a.a)/2+axis(a.b)/2, axis(b.a)/2+axis(b.b)/2)
	})
	middle := len(segments) / 2
	n.left, n.right = efBuildIndex(segments[:middle]), efBuildIndex(segments[middle:])
	return n
}

// efLowerBound uses the largest coordinate gap, a lower bound on Euclidean distance.
func (n *efNode) lowerBound(p r3.Vec) float64 {
	gap := func(x, lo, hi float64) float64 { return math.Max(0, math.Max(lo-x, x-hi)) }
	d := math.Max(gap(p.X, n.lo.X, n.hi.X), math.Max(gap(p.Y, n.lo.Y, n.hi.Y), gap(p.Z, n.lo.Z, n.hi.Z)))
	return math.Nextafter(d, math.Inf(-1))
}

func (n *efNode) nearest(p r3.Vec) float64 {
	if n == nil {
		return math.Inf(1)
	}
	return n.search(p, math.Inf(1))
}

func (n *efNode) search(p r3.Vec, best float64) float64 {
	if n.lowerBound(p) > best {
		return best
	}
	if n.left == nil {
		for _, s := range n.segments {
			best = math.Min(best, fitSegDist(p, s.a, s.b))
		}
		return best
	}
	first, second := n.left, n.right
	if second.lowerBound(p) < first.lowerBound(p) {
		first, second = second, first
	}
	best = first.search(p, best)
	return second.search(p, best)
}

// efMeasure samples the mid-height loop and reads each point's distance to the face section's loops.
func efMeasure(mid []r3.Vec, face *efNode) efShift {
	var out efShift
	for i, a := range mid {
		b := mid[(i+1)%len(mid)]
		n := int(math.Ceil(b.Sub(a).Len() / efStep))
		for k := 0; k < n; k++ {
			p := a.Add(b.Sub(a).Scale(float64(k) / float64(n)))
			d := face.nearest(p)
			l := b.Sub(a).Len() / float64(n)
			switch {
			case d < efSharp:
				out.unchanged += l
			case math.Abs(d-(onewayclutch.HubChamfer-efDepth)) < efOffset:
				out.offset += l
			default:
				out.other += l
			}
		}
	}
	return out
}

// efLengthAt is the length of the loops' segments whose end points both lie within tol of radius r about the axis.
func efLengthAt(loops [][]r3.Vec, r, tol float64) float64 {
	on := func(p r3.Vec) bool { return math.Abs(math.Hypot(p.X, p.Y)-r) < tol }
	total := 0.0
	for _, loop := range loops {
		for i, a := range loop {
			if b := loop[(i+1)%len(loop)]; on(a) && on(b) {
				total += b.Sub(a).Len()
			}
		}
	}
	return total
}

// efArea is the area of the hub face in mm²: the outer loop less the D bore and the three bolt holes.
func efArea(loops [][]r3.Vec) float64 {
	total := 0.0
	for _, loop := range loops {
		a := 0.0
		for i, p := range loop {
			q := loop[(i+1)%len(loop)]
			a += p.X*q.Y - q.X*p.Y
		}
		a = math.Abs(a) / 2
		if efKindOf(loop) == efOuter {
			total += a
		} else {
			total -= a
		}
	}
	return total
}

// efFlankTaper is the length of the loops' segments that lie on a pawl's flank line inside the arm's outer radius, in
// mm: the bevel's taper, which follows the flank's plane carried back past the arm's corner. The line is the head's
// flank at distance FlankDistance from the axis; the built direction mirrors it and each pawl is turned 120 degrees.
func efFlankTaper(m *onewayclutch.Model, loops [][]r3.Vec) float64 {
	dm := m.Dimensions
	theta := dm.RampAngle * math.Pi / 180
	n := r3.Vec{X: math.Sin(theta), Y: -math.Cos(theta)}
	if n.Dot(dm.FlankEnd) < 0 {
		n = n.Scale(-1)
	}
	n.X *= dm.Sign
	total := 0.0
	for k := range onewayclutch.Pawls {
		s, c := math.Sincos(-120 * float64(k) * math.Pi / 180)
		on := func(p r3.Vec) bool {
			q := r3.Vec{X: p.X*c - p.Y*s, Y: p.X*s + p.Y*c}
			return math.Abs(q.Dot(n)-dm.FlankDistance) < 0.005
		}
		for _, loop := range loops {
			for i, a := range loop {
				if b := loop[(i+1)%len(loop)]; on(a) && on(b) && math.Hypot(a.X+b.X, a.Y+b.Y)/2 < onewayclutch.ArmOuter-0.02 {
					total += b.Sub(a).Len()
				}
			}
		}
	}
	return total
}

// TestOneWayClutchEdgeFinishing checks the hub's chamfer on both faces in sections. The 605ZZ hub's faces are over
// 400 mm², so each has a 0.25 mm chamfer: the section 0.1 mm inside a face is the mid-height outline pulled in 0.15 mm
// along the arm's outer arc and each inner run, with the bevel tapering out on the head flank's plane, 0.911 mm along
// it from the arm's corner, and it leaves the head run, the bolt holes and the short D bore unchanged. The 8 mm hubs'
// faces are under 400 mm², so they are the exact prism and every section reads the same. A volume check would not do;
// the faceted result reads about 0.07 mm3 high per arc run.
func TestOneWayClutchEdgeFinishing(t *testing.T) {
	for _, c := range fitCases()[:6] {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			chamfered := m.Dimensions.Shaft.Name == "605ZZ"
			mid := efSection(t, m.Hub, midHub)
			area := efArea(mid)
			t.Logf("%s hub face %.1f mm2", c.name, area)
			if chamfered {
				require.Greater(t, area, edgefinish.LargeFaceArea, "the 605ZZ hub face is large")
			} else {
				require.Less(t, area, edgefinish.LargeFaceArea, "an 8 mm hub face is small")
			}
			for _, face := range []struct {
				name string
				z    float64
			}{
				{"back face", onewayclutch.HubFrom + efDepth},
				{"front face", onewayclutch.HubFrom + onewayclutch.HubHeight - efDepth},
			} {
				loops := efSection(t, m.Hub, face.z)
				require.Len(t, loops, len(mid), "%s: loop count", face.name)
				index := efIndex(loops)
				if !chamfered {
					for _, loop := range mid {
						got := efMeasure(loop, index)
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: an 8 mm hub is not chamfered", face.name)
					}
					continue
				}
				var outer efShift
				for _, loop := range mid {
					got := efMeasure(loop, index)
					switch efKindOf(loop) {
					case efHole:
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: a bolt hole is not chamfered", face.name)
					case efBore:
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: the short D bore stays sharp", face.name)
					case efOuter:
						outer = got
					}
				}
				// Per pawl: the head run stays sharp, with the bevel's taper out on the flank; the arm's outer arc and
				// the inner run are offset. Three pawls.
				perimeter := outer.total()
				t.Logf("%s %s: outer loop %.1f mm: unchanged %.1f, offset %.1f, other %.1f", c.name, face.name,
					perimeter, outer.unchanged, outer.offset, outer.other)
				require.Greater(t, outer.offset, 3*(20.1+45)-3*6, "%s: long runs offset", face.name)
				require.Greater(t, outer.unchanged, 3*7.1-3*3, "%s: head runs sharp", face.name)
				require.Less(t, outer.other, 0.05*perimeter, "%s: the transitions are short", face.name)

				// The arm's outer arc reads r 13.2 at mid-height and r 13.05 in the face section.
				arm := onewayclutch.ArmOuter
				chamfer := onewayclutch.HubChamfer - efDepth
				require.Greater(t, efLengthAt(mid, arm, 0.04), 3*20.0, "arm arcs at r %.2f at mid-height", arm)
				require.Greater(t, efLengthAt(loops, arm-chamfer, 0.04), 3*18.0, "%s: arm arcs at r %.2f", face.name, arm-chamfer)
				require.Less(t, efLengthAt(loops, arm, 0.04), 3*3.0, "%s: no arm arc stays at r %.2f", face.name, arm)

				// The bevel ends on the flank's plane: the section follows the flank line 0.911 mm past the arm's corner.
				taper := efFlankTaper(m, loops)
				require.Zero(t, efFlankTaper(m, mid), "%s: no taper at mid-height", face.name)
				t.Logf("%s %s: taper along the flank's plane, three pawls: %.3f mm", c.name, face.name, taper)
				// The faceted section reads 0.80 to 0.83 mm per pawl, 0.1 mm short of the exact 0.911; a missing taper or a square
				// end reads 0 or 0.5.
				require.InDelta(t, 3*0.911, taper, 3*0.15, "%s: the bevel tapers out along the flank's plane", face.name)
			}
		})
	}
}

// TestEdgeFinishIndex checks real face sections against the exhaustive segment search.
func TestEdgeFinishIndex(t *testing.T) {
	m := fitBuild(t, fitCases()[0])
	mid := efSection(t, m.Hub, midHub)
	for _, z := range []float64{onewayclutch.HubFrom + efDepth, onewayclutch.HubFrom + onewayclutch.HubHeight - efDepth} {
		face := efSection(t, m.Hub, z)
		index := efIndex(face)
		count := 0
		for _, loop := range mid {
			for i, a := range loop {
				b := loop[(i+1)%len(loop)]
				n := int(math.Ceil(b.Sub(a).Len() / efStep))
				for k := 0; k < n; k++ {
					count++
					if count%97 != 0 {
						continue
					}
					p := a.Add(b.Sub(a).Scale(float64(k) / float64(n)))
					got, want := index.nearest(p), fitNearest(p, face)
					require.Equal(t, want, got, "z %.3f sample %d point (%.6f, %.6f, %.6f)", z, count, p.X, p.Y, p.Z)
					require.Equal(t, efClass(want), efClass(got), "z %.3f sample %d", z, count)
				}
			}
		}
		require.Greater(t, count, 97)
	}
}

func efClass(d float64) int {
	if d < efSharp {
		return 0
	}
	if math.Abs(d-efDepth) < efOffset {
		return 1
	}
	return 2
}

func TestEdgeFinishIndexEdgeCases(t *testing.T) {
	cases := []struct {
		name   string
		loops  [][]r3.Vec
		points []r3.Vec
	}{
		{"empty", [][]r3.Vec{nil, {}}, []r3.Vec{{}, {X: 1}}},
		{"point", [][]r3.Vec{{{X: 1, Y: 2, Z: 3}}}, []r3.Vec{{}, {X: 1, Y: 2, Z: 3}}},
		{"tree", [][]r3.Vec{{{X: -5}, {X: -4}, {X: -3}, {X: -2}, {X: -1}, {},
			{X: 1}, {X: 2}, {X: 3}, {X: 4}, {X: 5}}},
			[]r3.Vec{{Y: 1}, {X: 4, Y: 1}, {X: -4, Z: 1}}},
		{"segments", [][]r3.Vec{{{X: -2, Z: 3}, {X: 2, Z: 3}}, {{Y: 2}, {Y: 2}}},
			[]r3.Vec{{}, {Z: 3}, {X: 4, Y: 2, Z: 7}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			index := efIndex(c.loops)
			for _, p := range c.points {
				require.Equal(t, fitNearest(p, c.loops), index.nearest(p), "point (%.3f, %.3f, %.3f)", p.X, p.Y, p.Z)
			}
		})
	}
}
