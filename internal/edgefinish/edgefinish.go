// Package edgefinish chamfers the edges of a body's large planar faces, as docs/model-pipeline.md "Surface edge
// finishing" requires.
package edgefinish

import (
	"context"
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

// LoopFilter decides whether the loop of a face, with its edges, is chamfered.
type LoopFilter func(loop *decad.Loop, edges []*decad.Edge) bool

// All keeps every loop.
func All(*decad.Loop, []*decad.Edge) bool { return true }

// Outer keeps a face's outer boundary.
func Outer(loop *decad.Loop, _ []*decad.Edge) bool { return loop.IsOuter() }

// Inner keeps the loops that are holes.
func Inner(loop *decad.Loop, _ []*decad.Edge) bool { return !loop.IsOuter() }

// Long keeps the loops f keeps whose edges add up to at least LongEdgeRun. An edge whose length Decad cannot bound
// adds nothing.
func Long(f LoopFilter) LoopFilter {
	return func(loop *decad.Loop, edges []*decad.Edge) bool {
		if !f(loop, edges) {
			return false
		}
		total := 0.0
		for _, e := range edges {
			length, err := e.Length()
			if err != nil {
				continue
			}
			total += length.Value.Base()
		}
		return total >= LongEdgeRun
	}
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
	if normal.Len() == 0 {
		return nil, 0, fmt.Errorf("edge finish: zero normal")
	}
	normal = normal.Scale(1 / normal.Len())
	var query *decad.EdgeQuery
	count := 0
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
			edges := loop.Edges()
			if !keep(loop, edges) {
				continue
			}
			for _, e := range edges {
				pred, err := predicates(e)
				if err != nil {
					return nil, 0, fmt.Errorf("edge finish: %w", err)
				}
				if query == nil {
					query = decad.Edges(pred...)
				} else {
					query = query.Or(pred...)
				}
				count++
			}
		}
	}
	if query == nil {
		return nil, 0, fmt.Errorf("edge finish: no edge to chamfer in the planes at %g along %v", offsets, normal)
	}
	out, err := decad.Chamfer(ctx, body, query.Exactly(count), units.Millimeters(size))
	if err != nil {
		return nil, 0, fmt.Errorf("edge finish: chamfer %d edges at %g along %v: %w", count, offsets, normal, err)
	}
	if !out.IsSolid() {
		return nil, 0, fmt.Errorf("edge finish: the chamfered result is not a solid")
	}
	return out, count, nil
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

// predicates names one edge by its start point and its kind: a circle or arc, or a line along its direction.
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
	default:
		return nil, fmt.Errorf("cannot select a %T edge at %v", c, start)
	}
}
