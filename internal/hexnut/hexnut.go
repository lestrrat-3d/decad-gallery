// Package hexnut constructs an ISO 4032 style hex nut of any size: a hexagonal prism with an optional round bore
// and a 30 degree chamfer on each face that starts at the bearing circle.
package hexnut

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

const (
	// ChamferAngle is the angle of the corner chamfer from the face, in degrees.
	ChamferAngle = 30.0

	// chamferMargin is how far the chamfer tool's cylinder clears the hexagon's corners.
	chamferMargin = 0.2
	// chamferReach is how far the chamfer tool's triangle extends past the nut's face.
	chamferReach = 0.3
)

// Size holds the nut dimensions in millimeters.
type Size struct {
	// Thread is the nominal thread diameter d. It is the bore diameter; the thread is not modeled.
	Thread float64
	// AcrossFlats is the wrench size s.
	AcrossFlats float64
	// Thickness is the nut height m.
	Thickness float64
	// BearingDiameter is the diameter dw of the flat bearing circle on each face.
	BearingDiameter float64
}

// AcrossCorners returns the distance e between opposite corners of the hexagon.
func (s Size) AcrossCorners() float64 {
	return s.AcrossFlats / math.Cos(math.Pi/6)
}

// Validate rejects a size that cannot build a nut.
func (s Size) Validate() error {
	if !(s.Thread > 0) || !(s.AcrossFlats > 0) || !(s.Thickness > 0) || !(s.BearingDiameter > 0) {
		return fmt.Errorf("hexnut: size dimensions must be positive")
	}
	if s.Thread >= s.BearingDiameter {
		return fmt.Errorf("hexnut: thread diameter must be below the bearing diameter")
	}
	if s.BearingDiameter >= s.AcrossFlats {
		return fmt.Errorf("hexnut: bearing diameter must be below the across-flats size")
	}
	return nil
}

// Options selects the optional features.
type Options struct {
	// Bore cuts the round hole of diameter Size.Thread through the nut.
	Bore bool
	// ChamferLower and ChamferUpper chamfer the corners of the lower and upper face.
	ChamferLower, ChamferUpper bool
}

// Result exposes the finished nut and the sketches that built it, all in the local frame. The local frame has
// its origin at the center of the lower face and +Z running from the lower face to the upper one.
type Result struct {
	Body *decad.Body
	// Hexagon is the hexagonal outline, with the bore when it is on, that is extruded along the axis.
	Hexagon *sketch.Sketch
	// LowerChamfer and UpperChamfer are the axial sections of the rings cut from the corners of each face.
	// Each is nil when that chamfer is off.
	LowerChamfer, UpperChamfer *sketch.Sketch
}

// Build adds one nut to doc. A nil placement selects the local frame. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, s Size, o Options, placement *r3.Frame) (*Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("hexnut: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("hexnut: document must not be nil")
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if placement != nil && !placement.IsValid() {
		return nil, fmt.Errorf("hexnut: placement must be a valid frame")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("hexnut: before creating hexagon: %w", err)
	}

	r := &Result{}
	world := sketch.NewWorld()
	body, err := extrudeHexagon(ctx, doc, world, s, o, r)
	if err != nil {
		return nil, err
	}
	for _, upper := range []bool{false, true} {
		if (upper && !o.ChamferUpper) || (!upper && !o.ChamferLower) {
			continue
		}
		if body, err = cutChamfer(ctx, doc, world, s, r, body, upper); err != nil {
			return nil, err
		}
	}
	if placement != nil {
		transform, err := r3.FromFrame(*placement)
		if err != nil {
			return nil, fmt.Errorf("hexnut: create placement transform: %w", err)
		}
		if body, err = decad.Transform(body, transform).Do(ctx); err != nil {
			return nil, fmt.Errorf("hexnut: place nut body: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("hexnut: verify nut body: body is not solid")
	}
	r.Body = body
	return r, nil
}

// extrudeHexagon extrudes the regular hexagon of AcrossFlats, with its round bore when on, along +Z by Thickness.
func extrudeHexagon(ctx context.Context, doc *decad.Document, world *sketch.World, size Size, o Options,
	r *Result) (*decad.Body, error) {
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("hexnut: create hexagon sketch: %w", err)
	}
	r.Hexagon = s
	hex, err := s.CreatePolygon(0, 0, 6, size.AcrossFlats/math.Sqrt(3))
	if err != nil {
		return nil, fmt.Errorf("hexnut: create hexagon: %w", err)
	}
	s.AddConstraint(
		sketch.NewCoincident(hex.Center, s.Origin()),
		// With the first corner on +X, a flat is parallel to X.
		sketch.NewHorizontal(hex.Sides[1]),
		// The center sits half the across-flats size from a flat; regularity makes the other flats follow.
		sketch.NewDistancePointLine(hex.Center, hex.Sides[1], size.AcrossFlats/2),
	)
	holes := 0
	if o.Bore {
		bore := s.CreateCircle(s.Origin(), size.Thread/2)
		s.AddConstraint(sketch.NewDiameter(bore, size.Thread))
		holes = 1
	}
	if err := solveFully(ctx, s); err != nil {
		return nil, fmt.Errorf("hexnut: hexagon sketch: %w", err)
	}
	var profile *sketch.Profile
	for _, candidate := range s.Profiles() {
		if !candidate.Valid || len(candidate.Holes) != holes {
			continue
		}
		if profile != nil {
			return nil, fmt.Errorf("hexnut: select hexagon profile: multiple valid profiles with %d holes", holes)
		}
		profile = candidate
	}
	if profile == nil {
		return nil, fmt.Errorf("hexnut: select hexagon profile: no valid profile with %d holes", holes)
	}
	prism, err := decad.Extrude(doc, s, profile, decad.Distance{D: units.Millimeters(size.Thickness), Dir: decad.Along}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("hexnut: extrude hexagon: %w", err)
	}
	return prism, nil
}

// cutChamfer cuts the 30 degree chamfer from the corners of the lower or upper face. The tool is a triangle
// revolved about the axis, drawn on the XZ plane with U as radius and V as height. Its slanted side lies on the
// chamfer cone, which meets the face at the bearing circle; the triangle reaches past the face so that no tool
// face lies in the nut's face, and its outer side clears the hexagon's corners.
func cutChamfer(ctx context.Context, doc *decad.Document, world *sketch.World, size Size, r *Result,
	body *decad.Body, upper bool) (*decad.Body, error) {
	name := "lower"
	if upper {
		name = "upper"
	}
	s, err := world.CreateSketch(world.XZ())
	if err != nil {
		return nil, fmt.Errorf("hexnut: create %s chamfer sketch: %w", name, err)
	}
	origin := s.Origin()
	slope := math.Tan(ChamferAngle * math.Pi / 180)
	outerR := size.AcrossCorners()/2 + chamferMargin
	// sign points into the nut from the face; face is the face height.
	sign, face, angle := 1.0, 0.0, ChamferAngle
	if upper {
		sign, face, angle = -1.0, size.Thickness, -ChamferAngle
	}
	outerZ := face + sign*(outerR-size.BearingDiameter/2)*slope
	startZ := face - sign*chamferReach
	start := s.CreatePoint(size.BearingDiameter/2-chamferReach/slope, startZ)
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
		sketch.NewHorizontalDistance(origin, start, size.BearingDiameter/2-chamferReach/slope),
		sketch.NewVertical(side),
		sketch.NewAngle(base, cone, angle),
	)
	profile, err := solvedProfile(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("hexnut: %s chamfer sketch: %w", name, err)
	}
	if upper {
		r.UpperChamfer = s
	} else {
		r.LowerChamfer = s
	}
	axis := decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}
	tool, err := decad.Revolve(doc, s, profile, axis, decad.FullRevolution{}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("hexnut: revolve %s chamfer tool: %w", name, err)
	}
	if body, err = decad.Cut(body, tool).Do(ctx); err != nil {
		return nil, fmt.Errorf("hexnut: cut %s chamfer: %w", name, err)
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
