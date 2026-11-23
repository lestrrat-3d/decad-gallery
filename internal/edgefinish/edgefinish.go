// Package edgefinish chamfers the edges of a body's large planar faces, as docs/model-pipeline.md "Surface edge
// finishing" requires.
package edgefinish

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

const (
	// MinChamfer is the setback, in mm, of the rule's equal-setback chamfer.
	MinChamfer = 0.5
	// LargeFaceArea is the net area, in mm², from which a planar face is large.
	LargeFaceArea = 400.0
	// LongEdgeRun is the arc length, in mm, from which a boundary run is long.
	LongEdgeRun = 20.0

	// planeTolerance is how far a face may sit from an offset it is looked up at.
	planeTolerance = 1e-6
	// normalTolerance is how far a face's unit normal may differ from the requested one.
	normalTolerance = 1e-9
)

// ErrNoEdge is the error, wrapped, that Chamfer and ChamferRuns return when no face in the requested planes reaches
// LargeFaceArea or no loop of one passes the filters, so there is nothing to chamfer. Callers whose faces may all be
// small test for it with errors.Is.
var ErrNoEdge = errors.New("no edge to chamfer")

// LoopFilter decides whether the loop of a face, with its edges, is chamfered.
type LoopFilter func(loop *decad.Loop, edges []*decad.Edge) bool

// All keeps every loop.
func All(*decad.Loop, []*decad.Edge) bool { return true }

// Outer keeps a face's outer boundary.
func Outer(loop *decad.Loop, _ []*decad.Edge) bool { return loop.IsOuter() }

// Inner keeps the loops that are holes.
func Inner(loop *decad.Loop, _ []*decad.Edge) bool { return !loop.IsOuter() }

// Long keeps the loops f keeps whose edges add up to at least LongEdgeRun. Decad refuses the length of a rim a Boolean
// cut into a curved source, so an edge it refuses is measured from its curve.
func Long(f LoopFilter) LoopFilter {
	return func(loop *decad.Loop, edges []*decad.Edge) bool {
		if !f(loop, edges) {
			return false
		}
		total := 0.0
		for _, e := range edges {
			total += edgeLength(e)
		}
		return total >= LongEdgeRun
	}
}

// edgeLength returns the length of e in mm: Decad's proven length, or the curve's own length when Decad refuses one.
func edgeLength(e *decad.Edge) float64 {
	if length, err := e.Length(); err == nil {
		return length.Value.Base()
	}
	start, end := e.Start().Position().Value, e.End().Position().Value
	switch c := e.Curve().(type) {
	case decad.Circle3:
		return 2 * math.Pi * c.Radius.Base()
	case decad.Arc3:
		a, b := start.Sub(c.Center), end.Sub(c.Center)
		// The edge's direction about Axis is unknown, so take the shorter way round. The arcs a Boolean cut leaves
		// in a loop are at most a half turn.
		sweep := math.Abs(math.Atan2(c.Axis.Dot(a.Cross(b)), a.Dot(b)))
		return c.Radius.Base() * sweep
	}
	return end.Sub(start).Len()
}

// Chamfer chamfers by size, in one decad.Chamfer call, every edge of every kept loop on every planar face of body whose
// normal is ±normal, whose plane lies at signed distance offsets[i] (±1e-6) along normal from the world origin, and
// whose area is at least LargeFaceArea. The evaluator cannot chamfer a body again after it chamfered a face's whole
// outline, so the faces that need it go together. Chamfer returns the new body and the edge count. It fails when no
// edge is selected or the result is not a solid. body, keep and normal must not be nil or zero.
//
// Body edges are matched by their stored start point and kind, which a chamfer leaves in place for the loops it does
// not touch. The query demands exactly the counted edges, so a mismatch fails the build.
func Chamfer(ctx context.Context, body *decad.Body, normal r3.Vec, offsets []float64, keep LoopFilter,
	size float64) (*decad.Body, int, error) {
	return chamfer(ctx, body, normal, offsets, size, func(loop *decad.Loop) ([]*decad.Edge, []*decad.Edge) {
		edges := loop.Edges()
		if !keep(loop, edges) {
			return nil, nil
		}
		return edges, nil
	})
}

// ChamferExcept is Chamfer that leaves out the edges of a kept loop for which skip is true. A loop that touches a
// fillet or another smooth wall has edges where the cap meets that wall tangent, and decad cannot chamfer a smooth edge;
// the caller names them with skip. skip may be nil.
func ChamferExcept(ctx context.Context, body *decad.Body, normal r3.Vec, offsets []float64, keep LoopFilter,
	skip func(*decad.Edge) bool, size float64) (*decad.Body, int, error) {
	return chamfer(ctx, body, normal, offsets, size, func(loop *decad.Loop) ([]*decad.Edge, []*decad.Edge) {
		edges := loop.Edges()
		if !keep(loop, edges) {
			return nil, nil
		}
		var out []*decad.Edge
		for _, e := range edges {
			if skip == nil || !skip(e) {
				out = append(out, e)
			}
		}
		return out, nil
	})
}

// ChamferRuns is Chamfer for faces whose outline mixes long and short tangent-continuous runs. It splits each loop
// that keep accepts into runs (see Runs) and chamfers the edges of the runs of LongEdgeRun or more, so the short
// runs, such as a tooth head, stay sharp. A loop that whole accepts is chamfered whole instead, however its runs
// divide: a D bore's arc and flat are short runs but a long loop.
//
// Decad takes a part of a cap loop only when each run ends at a concave corner of up to 90 degrees beside a line, or
// at a convex corner of 45 degrees or more, and refuses a call that also holds a whole loop. So ChamferRuns makes two
// calls: the partial runs first, then the whole loops on that result. Every edge is chosen on body, before the first
// cut, because a cut leaves the cap faces under LargeFaceArea and a Boolean result's D arc reads short. The edges of
// the second call keep their start point and kind through the first. The count is the sum of both calls. Decad's
// refusal comes back unchanged. The other rules are Chamfer's.
func ChamferRuns(ctx context.Context, body *decad.Body, normal r3.Vec, offsets []float64, keep, whole LoopFilter,
	size float64) (*decad.Body, int, error) {
	return chamfer(ctx, body, normal, offsets, size, func(loop *decad.Loop) ([]*decad.Edge, []*decad.Edge) {
		edges := loop.Edges()
		if !keep(loop, edges) {
			return nil, nil
		}
		if whole(loop, edges) {
			return nil, edges
		}
		var out []*decad.Edge
		for _, run := range Runs(loop) {
			if run.Length >= LongEdgeRun {
				out = append(out, run.Edges...)
			}
		}
		return out, nil
	})
}

// Run is a chain of consecutive edges of a loop that meet tangent to each other.
type Run struct {
	// Edges are the run's edges in walk order.
	Edges []*decad.Edge
	// Length is the run's arc length in mm.
	Length float64
	// Closed is true when the run is the whole loop, with no corner.
	Closed bool
}

// tangentCos is the cosine of the angle, 1e-3 rad, below which two edge directions at a joint are tangent.
const tangentCos = 1 - 5e-7

// Runs splits loop at its corners into tangent-continuous runs, in walk order. A loop with no corner is one closed
// run. A curve it cannot take a direction of counts as a corner on both sides.
func Runs(loop *decad.Loop) []Run {
	cos := loop.CoEdges()
	n := len(cos)
	if n == 0 {
		return nil
	}
	breaks := make([]bool, n) // breaks[i]: a corner lies between coedge i and i+1
	corners := 0
	for i, ce := range cos {
		next := cos[(i+1)%n]
		out, ok1 := endTangent(ce, true)
		in, ok2 := endTangent(next, false)
		breaks[i] = !ok1 || !ok2 || out.Dot(in) < tangentCos
		if breaks[i] {
			corners++
		}
	}
	if corners == 0 {
		run := Run{Closed: true}
		for _, ce := range cos {
			run.Edges = append(run.Edges, ce.Edge())
			run.Length += edgeLength(ce.Edge())
		}
		return []Run{run}
	}
	start := 0
	for i := range breaks {
		if breaks[i] {
			start = (i + 1) % n
			break
		}
	}
	var runs []Run
	var cur Run
	for k := 0; k < n; k++ {
		i := (start + k) % n
		cur.Edges = append(cur.Edges, cos[i].Edge())
		cur.Length += edgeLength(cos[i].Edge())
		if breaks[i] {
			runs = append(runs, cur)
			cur = Run{}
		}
	}
	return runs
}

// endTangent returns the unit direction of travel of ce at its end (atEnd) or start, and false when its curve has none.
func endTangent(ce decad.CoEdge, atEnd bool) (r3.Vec, bool) {
	e := ce.Edge()
	// Travel runs from ce.Start to ce.End; the stored curve runs from e.Start to e.End.
	p := ce.Start().Position().Value
	if atEnd {
		p = ce.End().Position().Value
	}
	var t r3.Vec
	switch c := e.Curve().(type) {
	case decad.Line3:
		t = e.End().Position().Value.Sub(e.Start().Position().Value)
	case decad.Arc3:
		t = c.Axis.Cross(p.Sub(c.Center))
	case decad.Circle3:
		t = c.Axis.Cross(p.Sub(c.Center))
	default:
		return r3.Vec{}, false
	}
	if !ce.IsForward() {
		t = t.Scale(-1)
	}
	if t.Len() == 0 {
		return r3.Vec{}, false
	}
	return t.Scale(1 / t.Len()), true
}

// chamfer is the shared body of Chamfer and ChamferRuns. pick returns the edges of a loop for the first call and for
// the second. Both are chosen on body before any cut.
func chamfer(ctx context.Context, body *decad.Body, normal r3.Vec, offsets []float64, size float64,
	pick func(*decad.Loop) ([]*decad.Edge, []*decad.Edge)) (*decad.Body, int, error) {
	if normal.Len() == 0 {
		return nil, 0, fmt.Errorf("edge finish: zero normal")
	}
	normal = normal.Scale(1 / normal.Len())
	var first, second selection
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok {
			continue
		}
		origin, n := plane.Frame.Origin(), plane.Frame.N()
		if math.Abs(math.Abs(n.Dot(normal))-1) > normalTolerance {
			continue
		}
		at, ok := matchOffset(offsets, origin.Dot(normal))
		if !ok {
			continue
		}
		area, err := f.Area()
		if err != nil {
			return nil, 0, fmt.Errorf("edge finish: measure the face at offset %g: %w", at, err)
		}
		if area.Value.Base() < LargeFaceArea {
			continue
		}
		for _, loop := range f.Loops() {
			a, b := pick(loop)
			if err := first.add(a); err != nil {
				return nil, 0, err
			}
			if err := second.add(b); err != nil {
				return nil, 0, err
			}
		}
	}
	if first.count+second.count == 0 {
		return nil, 0, fmt.Errorf("edge finish: %w in the planes at %g along %v", ErrNoEdge, offsets, normal)
	}
	out := body
	for _, sel := range []selection{first, second} {
		if sel.count == 0 {
			continue
		}
		next, err := decad.Chamfer(ctx, out, sel.query.Exactly(sel.count), units.Millimeters(size))
		if err != nil {
			return nil, 0, fmt.Errorf("edge finish: chamfer %d edges at %g along %v: %w", sel.count, offsets, normal, err)
		}
		out = next
	}
	if !out.IsSolid() {
		return nil, 0, fmt.Errorf("edge finish: the chamfered result is not a solid")
	}
	return out, first.count + second.count, nil
}

// selection is the edge query of one chamfer call and the number of edges it names.
type selection struct {
	query *decad.EdgeQuery
	count int
}

// add names each edge of edges in the selection.
func (s *selection) add(edges []*decad.Edge) error {
	for _, e := range edges {
		pred, err := predicates(e)
		if err != nil {
			return fmt.Errorf("edge finish: %w", err)
		}
		if s.query == nil {
			s.query = decad.Edges(pred...)
		} else {
			s.query = s.query.Or(pred...)
		}
		s.count++
	}
	return nil
}

// matchOffset returns the entry of offsets that d matches.
func matchOffset(offsets []float64, d float64) (float64, bool) {
	for _, want := range offsets {
		if math.Abs(d-want) <= planeTolerance {
			return want, true
		}
	}
	return 0, false
}

// predicates names one edge by its start point and its kind: a circle or arc, a line along its direction, or a faceted curve
// by both of its end points.
func predicates(e *decad.Edge) ([]decad.EdgePredicate, error) {
	start := e.Start().Position().Value
	switch c := e.Curve().(type) {
	case decad.Circle3, decad.Arc3:
		return []decad.EdgePredicate{decad.EndpointAt(start), decad.Circular()}, nil
	case decad.Line3:
		dir := e.End().Position().Value.Sub(start)
		if dir.Dot(dir) == 0 {
			return nil, fmt.Errorf("zero-length line edge at %v", start)
		}
		return []decad.EdgePredicate{decad.EndpointAt(start), decad.ParallelTo(dir)}, nil
	case decad.FacetedCurve:
		// A chord run of a faceted round has no direction of its own: its two end points name it.
		return []decad.EdgePredicate{decad.EndpointAt(start), decad.EndpointAt(e.End().Position().Value)}, nil
	default:
		return nil, fmt.Errorf("cannot select a %T edge at %v", c, start)
	}
}
