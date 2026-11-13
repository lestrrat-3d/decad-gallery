// Package spiralspring builds a flat spiral (clock or motor) spring as one printable solid: a hub with a D bore at
// the inner end, a two-center arc spiral, and a straight tail ending in a round eye for a housing pin.
//
// The local frame has its origin on the spring axis in the bottom face and +Z along the axis. The spring winds
// counter-clockwise outward seen from +Z.
package spiralspring

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Model is a built spiral spring.
type Model struct {
	Body       *decad.Body
	Profile    *sketch.Sketch
	Dimensions Dimensions
}

// Build adds one spring body to doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, errors.New("spiralspring: context must not be nil")
	}
	if doc == nil {
		return nil, errors.New("spiralspring: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("spiralspring: before creating profile: %w", err)
	}
	s, profile, corners, err := buildProfile(ctx, p, dm)
	if err != nil {
		return nil, err
	}

	h := p.Height
	body, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(h), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spiralspring: extrude spring: %w", err)
	}

	// Fillet the five concave vertical corners.
	var sel *decad.EdgeQuery
	for _, c := range corners {
		preds := []decad.EdgePredicate{decad.Concave(), decad.ParallelTo(r3.Vec{Z: 1}), decad.EndpointAt(c)}
		if sel == nil {
			sel = decad.Edges(preds...)
		} else {
			sel = sel.Or(preds...)
		}
	}
	body, err = decad.Fillet(ctx, body, sel.Exactly(len(corners)), units.Millimeters(dm.FilletRadius))
	if err != nil {
		return nil, fmt.Errorf("spiralspring: fillet corners: %w", err)
	}

	// Edge finishing, when a cap is a large planar face.
	large := false
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok {
			continue
		}
		if n := plane.Frame.N(); n.Z*n.Z < 1-1e-18 {
			continue
		}
		z := plane.Frame.Origin().Z
		if z > 1e-6 && z < h-1e-6 || z < -1e-6 || z > h+1e-6 {
			continue
		}
		area, err := f.Area()
		if err != nil {
			return nil, fmt.Errorf("spiralspring: measure cap: %w", err)
		}
		if area.Value.Base() >= edgefinish.LargeFaceArea {
			large = true
		}
	}
	if large {
		keep := func(loop *decad.Loop, edges []*decad.Edge) bool {
			return edgefinish.Outer(loop, edges) || edgefinish.Long(edgefinish.Inner)(loop, edges)
		}
		body, _, err = edgefinish.Chamfer(ctx, body, r3.Vec{Z: 1}, []float64{0, h}, keep, edgefinish.MinChamfer)
		if err != nil {
			return nil, fmt.Errorf("spiralspring: finish cap edges: %w", err)
		}
	}

	body, err = placement.Place(ctx, body, p.Placement)
	if err != nil {
		return nil, fmt.Errorf("spiralspring: place spring: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spiralspring: spring is not a solid")
	}
	return &Model{Body: body, Profile: s, Dimensions: dm}, nil
}
