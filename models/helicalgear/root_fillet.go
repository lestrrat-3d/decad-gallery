package helicalgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// filletRootEdges follows the source's root-cylinder face and axial-edge
// selection. A gear with no matching root edge skips the fillet.
func filletRootEdges(ctx context.Context, doc *decad.Document, gear *decad.Body, _ r3.Frame) (*decad.Body, error) {
	rootValue, err := doc.Param("root_radius")
	if err != nil {
		return nil, fmt.Errorf("read root radius: %w", err)
	}
	radius, err := doc.Param("helical_fillet_radius")
	if err != nil {
		return nil, fmt.Errorf("read helical fillet radius: %w", err)
	}
	if radius.Base() <= 0 {
		return gear, nil
	}
	root := rootValue.Base()

	const rootRadiusTolerance = 0.001 // The source's 0.0001 cm, converted to millimeters.
	var roots []*decad.Edge
	seen := make(map[*decad.Edge]struct{})
	for _, face := range gear.Faces() {
		cylinder, ok := face.Surface().(decad.Cylinder)
		if !ok || math.Abs(cylinder.Radius.Base()-root) > rootRadiusTolerance {
			continue
		}
		for _, edge := range face.Edges() {
			if _, ok := edge.Curve().(decad.Line3); !ok {
				continue
			}
			start := edge.Start().Position().Value
			end := edge.End().Position().Value
			dx, dy, dz := end.X-start.X, end.Y-start.Y, end.Z-start.Z
			length := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if length == 0 || math.Abs(math.Abs(dz/length)-1) >= 0.01 {
				continue
			}
			if _, ok := seen[edge]; ok {
				continue
			}
			seen[edge] = struct{}{}
			roots = append(roots, edge)
		}
	}
	if len(roots) == 0 {
		return gear, nil
	}

	var selection *decad.EdgeQuery
	for _, edge := range roots {
		start := edge.Start().Position().Value
		end := edge.End().Position().Value
		if selection == nil {
			selection = decad.Edges(decad.EndpointAt(start), decad.EndpointAt(end))
			continue
		}
		selection.Or(decad.EndpointAt(start), decad.EndpointAt(end))
	}
	rounded, err := decad.Fillet(ctx, gear, selection.Exactly(len(roots)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet helical root edges: %w", err)
	}
	return rounded, nil
}
