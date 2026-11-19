package onewayclutch_test

import (
	"cmp"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
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

// efKindOf classifies a loop of the hub's outline: a bolt hole's centre lies on the bolt circle, the bore lies inside
// the bolt holes, and the outer loop is the rest.
func efKindOf(loop []r3.Vec, dm onewayclutch.Dimensions) efKind {
	lo, hi := r3.Vec{X: math.Inf(1), Y: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1)}
	far := 0.0
	for _, p := range loop {
		lo, hi = r3.Vec{X: math.Min(lo.X, p.X), Y: math.Min(lo.Y, p.Y)}, r3.Vec{X: math.Max(hi.X, p.X), Y: math.Max(hi.Y, p.Y)}
		far = math.Max(far, math.Hypot(p.X, p.Y))
	}
	switch {
	case math.Abs(math.Hypot((lo.X+hi.X)/2, (lo.Y+hi.Y)/2)-dm.BoltCircle) < 0.3 && hi.X-lo.X < 4:
		return efHole
	case far < dm.BoltCircle-m2fastener.HoleRadius-0.3:
		return efBore
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
func efMeasure(mid []r3.Vec, face *efNode, chamfer float64) efShift {
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
			case math.Abs(d-(chamfer-efDepth)) < efOffset:
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
func efArea(loops [][]r3.Vec, dm onewayclutch.Dimensions) float64 {
	total := 0.0
	for _, loop := range loops {
		a := 0.0
		for i, p := range loop {
			q := loop[(i+1)%len(loop)]
			a += p.X*q.Y - q.X*p.Y
		}
		a = math.Abs(a) / 2
		if efKindOf(loop, dm) == efOuter {
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
				if b := loop[(i+1)%len(loop)]; on(a) && on(b) && math.Hypot(a.X+b.X, a.Y+b.Y)/2 < dm.ArmOuter-0.02 {
					total += b.Sub(a).Len()
				}
			}
		}
	}
	return total
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
