// Package pinring constructs a plain flat ring that holds a pin in place.
package pinring

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Default dimensions in millimeters.
const (
	// DefaultRodRadius is the radius of the 5 mm thin rod the ring sits on.
	DefaultRodRadius = 2.5
	// DefaultClearance is the radial clearance between the rod and the ring's bore.
	DefaultClearance = 0.5
	// DefaultWidth is the radial width of the ring, from the bore outward.
	DefaultWidth = 5.0
	// DefaultThickness is the ring's height along its axis.
	DefaultThickness = 5.0
)

// Params specifies the ring. A nil Placement selects the local frame: the axis is the Z axis and the
// ring spans z from 0 to Thickness.
type Params struct {
	// RodRadius is the radius of the rod the ring fits on.
	RodRadius float64
	// Clearance is the radial gap between the rod and the bore.
	Clearance float64
	// Width is the radial width of the ring.
	Width float64
	// Thickness is the ring's height along +Z.
	Thickness float64
	// Placement maps the local frame onto an assembly frame.
	Placement *r3.Frame
}

// DefaultParams returns the 3 mm inner radius and 8 mm outer radius ring, 5 mm thick.
func DefaultParams() Params {
	return Params{
		RodRadius: DefaultRodRadius,
		Clearance: DefaultClearance,
		Width:     DefaultWidth,
		Thickness: DefaultThickness,
	}
}

// InnerRadius returns the bore radius: the rod radius plus the clearance.
func (p Params) InnerRadius() float64 { return p.RodRadius + p.Clearance }

// OuterRadius returns the outer radius: the bore radius plus the width.
func (p Params) OuterRadius() float64 { return p.InnerRadius() + p.Width }

// Validate rejects dimensions that cannot form a ring and an invalid Placement.
func (p Params) Validate() error {
	if !(p.RodRadius > 0) {
		return fmt.Errorf("pinring: RodRadius must be positive, got %g", p.RodRadius)
	}
	if !(p.Clearance >= 0) {
		return fmt.Errorf("pinring: Clearance must not be negative, got %g", p.Clearance)
	}
	if !(p.Width > 0) {
		return fmt.Errorf("pinring: Width must be positive, got %g", p.Width)
	}
	if !(p.Thickness > 0) {
		return fmt.Errorf("pinring: Thickness must be positive, got %g", p.Thickness)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("pinring: Placement must be a valid frame")
	}
	return nil
}

// Model exposes the finished ring and the sketch that built it.
type Model struct {
	Body *decad.Body
	// Profile is the annulus sketch on the XY plane that is extruded along +Z.
	Profile *sketch.Sketch
}

// Build adds one ring to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("pinring: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("pinring: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("pinring: before creating profile: %w", err)
	}

	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("pinring: create profile sketch: %w", err)
	}
	outer := s.CreateCircle(s.Origin(), p.OuterRadius())
	bore := s.CreateCircle(s.Origin(), p.InnerRadius())
	s.AddConstraint(
		sketch.NewDiameter(outer, 2*p.OuterRadius()),
		sketch.NewDiameter(bore, 2*p.InnerRadius()),
	)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("pinring: solve profile sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("pinring: verify profile sketch: %w", err)
	}
	var profile *sketch.Profile
	for _, candidate := range s.Profiles() {
		if !candidate.Valid || len(candidate.Holes) != 1 {
			continue
		}
		if profile != nil {
			return nil, fmt.Errorf("pinring: select profile: multiple valid profiles with one hole")
		}
		profile = candidate
	}
	if profile == nil {
		return nil, fmt.Errorf("pinring: select profile: no valid profile with one hole")
	}
	body, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("pinring: extrude ring: %w", err)
	}
	if p.Placement != nil {
		transform, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("pinring: create placement transform: %w", err)
		}
		if body, err = decad.Transform(ctx, body, transform); err != nil {
			return nil, fmt.Errorf("pinring: place ring body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("pinring: verify ring body: body is not solid")
	}
	return &Model{Body: body, Profile: s}, nil
}
