// Package m2nut constructs an M2 hex nut.
package m2nut

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

// Nominal M2 hex nut dimensions in millimeters, following ISO 4032 and JIS B 1181.
const (
	// BoreDiameter is the nominal thread diameter d. The thread is not modeled.
	BoreDiameter = 2.0
	// AcrossFlats is the wrench size s.
	AcrossFlats = 4.0
	// Thickness is the nut height m.
	Thickness = 1.6
	// BearingDiameter is the diameter dw of the flat bearing circle on each face.
	BearingDiameter = 3.48
	// ChamferAngle is the angle of the corner chamfer from the face, in degrees.
	ChamferAngle = 30.0

	// chamferMargin is how far the chamfer tool's cylinder clears the hexagon's corners.
	chamferMargin = 0.2
	// chamferReach is how far the chamfer tool's triangle extends past the nut's face.
	chamferReach = 0.3
)

// AcrossCorners returns the distance e between opposite corners of the hexagon.
func AcrossCorners() float64 {
	return AcrossFlats / math.Cos(math.Pi/6)
}

// Params specifies the nut. A nil Placement selects the local frame: the origin is the center of the
// lower face and +Z runs from the lower face to the upper one.
type Params struct {
	Placement *r3.Frame
}

// DefaultParams returns the M2 nut in its local frame.
func DefaultParams() Params {
	return Params{}
}

// Validate rejects an invalid Placement.
func (p Params) Validate() error {
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("m2nut: Placement must be a valid frame")
	}
	return nil
}

// Model exposes the finished nut and the sketches that built it, all in the local frame.
type Model struct {
	Body *decad.Body
	// Hexagon is the hexagonal outline and bore that are extruded along the axis.
	Hexagon *sketch.Sketch
	// LowerChamfer and UpperChamfer are the axial sections of the rings cut from the corners of each face.
	LowerChamfer, UpperChamfer *sketch.Sketch
}

// Build adds one nut to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("m2nut: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("m2nut: document must not be nil")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("m2nut: before creating hexagon: %w", err)
	}

	m := &Model{}
	world := sketch.NewWorld()
	prism, err := extrudeHexagon(ctx, doc, world, m)
	if err != nil {
		return nil, err
	}
	body := prism
	for _, upper := range []bool{false, true} {
		if body, err = cutChamfer(ctx, doc, world, m, body, upper); err != nil {
			return nil, err
		}
	}
	if p.Placement != nil {
		transform, err := r3.FromFrame(*p.Placement)
		if err != nil {
			return nil, fmt.Errorf("m2nut: create placement transform: %w", err)
		}
		if body, err = decad.Transform(ctx, body, transform); err != nil {
			return nil, fmt.Errorf("m2nut: place nut body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("m2nut: verify nut body: body is not solid")
	}
	m.Body = body
	return m, nil
}

// extrudeHexagon extrudes the regular hexagon of AcrossFlats, with its round bore, along +Z by Thickness.
func extrudeHexagon(ctx context.Context, doc *decad.Document, world *sketch.World, m *Model) (*decad.Body, error) {
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("m2nut: create hexagon sketch: %w", err)
	}
	m.Hexagon = s
	hex, err := s.CreatePolygon(0, 0, 6, AcrossFlats/math.Sqrt(3))
	if err != nil {
		return nil, fmt.Errorf("m2nut: create hexagon: %w", err)
	}
	bore := s.CreateCircle(s.Origin(), BoreDiameter/2)
	s.AddConstraint(
		sketch.NewDiameter(bore, BoreDiameter),
		sketch.NewCoincident(hex.Center, s.Origin()),
		// With the first corner on +X, a flat is parallel to X.
		sketch.NewHorizontal(hex.Sides[1]),
		// The center sits half the across-flats size from a flat; regularity makes the other flats follow.
		sketch.NewDistancePointLine(hex.Center, hex.Sides[1], AcrossFlats/2),
	)
	if err := solveFully(ctx, s); err != nil {
		return nil, fmt.Errorf("m2nut: hexagon sketch: %w", err)
	}
	var profile *sketch.Profile
	for _, candidate := range s.Profiles() {
		if !candidate.Valid || len(candidate.Holes) != 1 {
			continue
		}
		if profile != nil {
			return nil, fmt.Errorf("m2nut: select hexagon profile: multiple valid profiles with one hole")
		}
		profile = candidate
	}
	if profile == nil {
		return nil, fmt.Errorf("m2nut: select hexagon profile: no valid profile with one hole")
	}
	prism, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(Thickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("m2nut: extrude hexagon: %w", err)
	}
	return prism, nil
}

// cutChamfer cuts the 30 degree chamfer from the corners of the lower or upper face. The tool is a triangle
// revolved about the axis, drawn on the XZ plane with U as radius and V as height. Its slanted side lies on the
// chamfer cone, which meets the face at the bearing circle; the triangle reaches past the face so that no tool
// face lies in the nut's face, and its outer side clears the hexagon's corners.
func cutChamfer(ctx context.Context, doc *decad.Document, world *sketch.World, m *Model, body *decad.Body,
	upper bool) (*decad.Body, error) {
	name := "lower"
	if upper {
		name = "upper"
	}
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, fmt.Errorf("m2nut: create %s chamfer sketch: %w", name, err)
	}
	origin := s.Origin()
	slope := math.Tan(ChamferAngle * math.Pi / 180)
	outerR := AcrossCorners()/2 + chamferMargin
	// sign points into the nut from the face; face is the face height.
	sign, face, angle := 1.0, 0.0, ChamferAngle
	if upper {
		sign, face, angle = -1.0, Thickness, -ChamferAngle
	}
	outerZ := face + sign*(outerR-BearingDiameter/2)*slope
	startZ := face - sign*chamferReach
	start := s.CreatePoint(BearingDiameter/2-chamferReach/slope, startZ)
	foot := s.CreatePoint(outerR, startZ)
	corner := s.CreatePoint(outerR, outerZ)
	base := s.CreateLine(start, foot)
	cone := s.CreateLine(start, corner)
	side := s.CreateLine(foot, corner)
	s.AddConstraint(
		sketch.NewHorizontalDistance(origin, foot, outerR),
		sketch.NewVerticalDistance(origin, foot, startZ),
		sketch.NewHorizontal(base),
		// The cone passes through the bearing circle on the face.
		sketch.NewHorizontalDistance(origin, start, BearingDiameter/2-chamferReach/slope),
		sketch.NewVertical(side),
		sketch.NewAngle(base, cone, angle),
	)
	profile, err := solvedProfile(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("m2nut: %s chamfer sketch: %w", name, err)
	}
	if upper {
		m.UpperChamfer = s
	} else {
		m.LowerChamfer = s
	}
	axis := decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}
	tool, err := decad.Revolve(ctx, doc, s, profile, axis, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("m2nut: revolve %s chamfer tool: %w", name, err)
	}
	if body, err = decad.Cut(ctx, body, tool); err != nil {
		return nil, fmt.Errorf("m2nut: cut %s chamfer: %w", name, err)
	}
	return body, nil
}

// solveFully solves s and requires it fully constrained.
func solveFully(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// solvedProfile solves s, requires it fully constrained and returns its one valid profile.
func solvedProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if err := solveFully(ctx, s); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("select profile: got %d profiles, want one valid region", len(profiles))
	}
	return profiles[0], nil
}
