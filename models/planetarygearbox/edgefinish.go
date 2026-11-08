package planetarygearbox

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// MinEdgeChamfer is the smallest chamfer on a large flat face's edges that Surface edge finishing allows.
const MinEdgeChamfer = 0.5

// largeFaceArea is the net area, in square millimeters, from which a flat face is large enough to need its edges
// finished.
const largeFaceArea = 400

// edgeFinishTol is how far a face may sit from the height it is looked up at.
const edgeFinishTol = 1e-6

// finishFaceEdges chamfers by EdgeChamfer the edges of the large flat face of body that lies in the plane z, one
// loop at a time. keep decides which of the face's boundary loops are chamfered.
//
// Body edges are matched by their stored start point and direction, which a chamfer leaves in place for the loops
// it does not touch. The selection must pick exactly the edges of the kept loops, or the build fails.
func (r *run) finishFaceEdges(ctx context.Context, body *decad.Body, what string, z float64,
	keep func(loop *decad.Loop, edges []*decad.Edge) bool) (*decad.Body, error) {
	return r.finishFacesEdges(ctx, body, what, []float64{z}, keep)
}

// finishFacesEdges is finishFaceEdges over the large flat faces in several planes, chamfered in one operation. The
// evaluator cannot chamfer a body again after a chamfer of a face's whole outline, so faces that need it are
// finished together.
func (r *run) finishFacesEdges(ctx context.Context, body *decad.Body, what string, zs []float64,
	keep func(loop *decad.Loop, edges []*decad.Edge) bool) (*decad.Body, error) {
	size, err := r.doc.Param("edge_chamfer")
	if err != nil {
		return nil, fmt.Errorf("%s: read edge chamfer: %w", what, err)
	}
	var query *decad.EdgeQuery
	count := 0
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok {
			continue
		}
		origin, normal := plane.Frame.Origin(), plane.Frame.N()
		if math.Abs(math.Abs(normal.Z)-1) > 1e-9 {
			continue
		}
		z, ok := planeHeight(zs, origin.Z)
		if !ok {
			continue
		}
		area, err := f.Area()
		if err != nil {
			return nil, fmt.Errorf("%s: measure flat face in the plane z=%g: %w", what, z, err)
		}
		if area.Value.Base() < largeFaceArea {
			continue
		}
		for _, loop := range f.Loops() {
			edges := loop.Edges()
			if !keep(loop, edges) {
				continue
			}
			for _, e := range edges {
				pred, err := edgePredicates(e)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", what, err)
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
		return nil, fmt.Errorf("%s: no loop to chamfer in the planes z=%g", what, zs)
	}
	out, err := decad.Chamfer(ctx, body, query.Exactly(count), size)
	if err != nil {
		return nil, fmt.Errorf("%s: chamfer %d edges in the planes z=%g: %w", what, count, zs, err)
	}
	if !out.IsSolid() {
		return nil, fmt.Errorf("%s: chamfered result is not a solid", what)
	}
	return out, nil
}

// planeHeight returns the height in zs that z matches, within edgeFinishTol.
func planeHeight(zs []float64, z float64) (float64, bool) {
	for _, want := range zs {
		if math.Abs(z-want) <= edgeFinishTol {
			return want, true
		}
	}
	return 0, false
}

// edgePredicates names one edge by its start point and its kind: a circle or arc, or a line along its direction.
func edgePredicates(e *decad.Edge) ([]decad.EdgePredicate, error) {
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

// outerLoop keeps only the outer boundary of a face: a rim, or a plate's outline.
func outerLoop(loop *decad.Loop, _ []*decad.Edge) bool { return loop.IsOuter() }

// windowsAndRim keeps the outer boundary and every inner loop of more than one edge: a face's windows.
func windowsAndRim(loop *decad.Loop, edges []*decad.Edge) bool {
	return loop.IsOuter() || len(edges) > 1
}

// radiusAbout is the distance from the z axis of the start of e.
func radiusAbout(e *decad.Edge) float64 {
	at := e.Start().Position().Value
	return math.Hypot(at.X, at.Y)
}
