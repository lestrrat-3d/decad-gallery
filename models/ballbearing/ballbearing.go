// Package ballbearing constructs nominal mounting envelopes for purchased bearings.
package ballbearing

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params specifies nominal dimensions in millimeters and an optional placement.
// A nil Placement selects the local XY plane with extrusion along +Z.
type Params struct {
	BoreDiameter  float64
	OuterDiameter float64
	Width         float64
	Placement     *r3.Frame
}

// DefaultParams returns the nominal 608-2RS mounting dimensions.
func DefaultParams() Params {
	return Params{BoreDiameter: 8, OuterDiameter: 22, Width: 7}
}

// Validate rejects nonfinite or nonpositive dimensions, an outer diameter no
// larger than the bore, and an invalid placement frame.
func (p Params) Validate() error {
	for _, dimension := range []struct {
		name  string
		value float64
	}{
		{"BoreDiameter", p.BoreDiameter},
		{"OuterDiameter", p.OuterDiameter},
		{"Width", p.Width},
	} {
		if math.IsNaN(dimension.value) || math.IsInf(dimension.value, 0) || dimension.value <= 0 {
			return fmt.Errorf("ballbearing: %s must be finite and positive", dimension.name)
		}
	}
	if p.OuterDiameter <= p.BoreDiameter {
		return fmt.Errorf("ballbearing: OuterDiameter must be larger than BoreDiameter")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("ballbearing: Placement must be a valid frame")
	}
	return nil
}

// Model exposes the final envelope body and the local sketch used to extrude it.
type Model struct {
	Body   *decad.Body
	Sketch *sketch.Sketch
}

// Build adds one annular mounting envelope to doc. ctx and doc must not be nil.
// A supplied placement maps the lower cap center to its origin and the bearing
// axis to its normal. Sketch always refers to the actual local extrusion section.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ballbearing: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("ballbearing: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("ballbearing: before creating section: %w", err)
	}

	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("ballbearing: create section sketch: %w", err)
	}
	outer := s.CreateCircle(s.Origin(), p.OuterDiameter/2)
	bore := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(
		sketch.NewDiameter(outer, p.OuterDiameter),
		sketch.NewDiameter(bore, p.BoreDiameter),
	)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("ballbearing: solve section sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("ballbearing: verify section sketch: %w", err)
	}

	var annulus *sketch.Profile
	for _, profile := range s.Profiles() {
		if !profile.Valid || len(profile.Holes) != 1 {
			continue
		}
		if annulus != nil {
			return nil, fmt.Errorf("ballbearing: select annular profile: multiple valid profiles with one hole")
		}
		annulus = profile
	}
	if annulus == nil {
		return nil, fmt.Errorf("ballbearing: select annular profile: no valid profile with one hole")
	}

	body, err := decad.Extrude(ctx, doc, s, annulus,
		decad.Distance{D: units.Millimeters(p.Width), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("ballbearing: extrude annular section: %w", err)
	}
	if p.Placement != nil {
		transform, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("ballbearing: create placement transform: %w", err)
		}
		body, err = decad.Transform(ctx, body, transform)
		if err != nil {
			return nil, fmt.Errorf("ballbearing: place envelope body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("ballbearing: verify envelope body: body is not solid")
	}
	return &Model{Body: body, Sketch: s}, nil
}
