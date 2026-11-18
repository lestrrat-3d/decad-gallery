package dovetail

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// ChamferSize is the setback in mm of the equal-setback chamfer on each convex joint edge: the tail's two wide-end edges
// and the socket's two lip edges. It is the standard chamfer of the edge-finishing rule.
const ChamferSize = edgefinish.MinChamfer

// edgeTolerance is how far an edge end may lie from the point it is looked up at, in mm.
const edgeTolerance = 1e-5

// convexEdges returns the world start and end points of the two convex joint edges that run along V, given the
// frame, the tail direction s and the two local (U, N) positions u, n of the edges: (-u, n) and (+u, n).
func (p Params) convexEdges(f r3.Frame, u, n float64) [][2]r3.Vec {
	out := make([][2]r3.Vec, 2)
	for i, su := range []float64{-1, 1} {
		out[i][0] = f.ToWorld(r3.Vec{X: su * u, Y: 0, Z: n})
		out[i][1] = f.ToWorld(r3.Vec{X: su * u, Y: p.Length, Z: n})
	}
	return out
}

// chamferEdges chamfers by ChamferSize the edges of body that run between each of the given point pairs, in one call. It
// finds each edge on the body by its end points, so it names the edge decad stored, and it fails when an edge is
// missing.
func chamferEdges(ctx context.Context, body *decad.Body, ends [][2]r3.Vec, what string) (*decad.Body, error) {
	var query *decad.EdgeQuery
	for _, want := range ends {
		var found *decad.Edge
		for _, e := range body.Edges() {
			a, b := e.Start().Position().Value, e.End().Position().Value
			if near(a, want[0]) && near(b, want[1]) || near(a, want[1]) && near(b, want[0]) {
				found = e
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("dovetail: chamfer %s: no edge from %v to %v", what, want[0], want[1])
		}
		start := found.Start().Position().Value
		dir := found.End().Position().Value.Sub(start)
		pred := []decad.EdgePredicate{decad.EndpointAt(start), decad.ParallelTo(dir)}
		if query == nil {
			query = decad.Edges(pred...)
		} else {
			query = query.Or(pred...)
		}
	}
	out, err := decad.Chamfer(ctx, body, query.Exactly(len(ends)), units.Millimeters(ChamferSize))
	if err != nil {
		return nil, fmt.Errorf("dovetail: chamfer %s: %w", what, err)
	}
	if !out.IsSolid() {
		return nil, fmt.Errorf("dovetail: chamfer %s: the result is not a solid", what)
	}
	return out, nil
}

func near(a, b r3.Vec) bool { return a.Sub(b).Len() <= edgeTolerance }

// ChamferCut is the area, in mm squared, that one chamfer removes from the profile: a triangle with two sides of
// ChamferSize at the convex corner of the profile, which is 90 - Angle degrees wide.
func (p Params) ChamferCut() float64 {
	return 0.5 * ChamferSize * ChamferSize * math.Cos(p.Angle*math.Pi/180)
}
