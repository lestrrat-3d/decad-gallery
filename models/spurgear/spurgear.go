// Package spurgear builds a parametric cylindrical spur gear with straight
// involute teeth.
package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters and angles radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement is the reference point (origin) and plane (U, V) of the gear.
	// The axis runs along its normal and the lower cap lies on the plane.
	// Nil selects the world XY frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

// RejectPlacement returns an error naming model when Placement is set.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// validate applies the contract's input checks and limits.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("spurgear: module must be positive, got %g", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("spurgear: thickness must be positive, got %g", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("spurgear: pressure angle must be in (0, pi/2), got %g", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter must not be negative, got %g", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer must not be negative, got %g", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: invalid placement frame")
	}
	d := derive(p)
	switch {
	case !(d.root > 0):
		return fmt.Errorf("spurgear: root radius %g must be positive", d.root)
	case !(d.base > 0):
		return fmt.Errorf("spurgear: base radius %g must be positive", d.base)
	case !(d.tip > d.base):
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*d.root)
	case !finite(filletRadius(p, d)):
		return errors.New("spurgear: root fillet radius is not finite")
	}
	return nil
}

func solveAndVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s: %w", what, err)
	}
	return nil
}

// toothPoints are the dimensioned points of one tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootR, rootL        point
}

// dimensionTooth creates the flank points, and the root points when withRoots.
func dimensionTooth(s *sketch.Sketch, left, right [][2]float64, d dimensions, withRoots bool) toothPoints {
	n := len(left)
	tp := toothPoints{right: make([]*sketch.Point, n), left: make([]*sketch.Point, n)}
	for i := range n {
		tp.right[i] = sketchcheck.DimensionPoint(s, right[i][0], right[i][1])
		if i == n-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, left[i][0], left[i][1])
		} else {
			tp.left[i] = sketchcheck.DimensionPoint(s, left[i][0], left[i][1])
		}
	}
	if withRoots {
		k := d.root / d.base
		tp.rootR = point{right[0][0] * k, right[0][1] * k}
		tp.rootL = point{left[0][0] * k, left[0][1] * k}
		tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rootR.x, tp.rootR.y)
		tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.rootL.x, tp.rootL.y)
	}
	return tp
}

// buildGearProfile draws the complete unfilleted outline on plane.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create profile sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := range p.Teeth {
		angle := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
		tp := dimensionTooth(s, left, right, d, true)
		roots = append(roots, tp.rootR, tp.rootL)
		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, prevLeft, tp.rightRoot)
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, tp.right[p.Samples-1], tp.left[p.Samples-1])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		prevLeft = tp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s, "gear profile"); err != nil {
		return nil, nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile has %d regions, want 1 valid", len(profiles))
	}
	return s, profiles[0], roots, nil
}

// buildEmbeddedProfile handles Rb < Rr: flanks and tip arcs plus a root circle.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create embedded sketch: %w", err)
	}
	center := s.Origin()
	for k := range p.Teeth {
		angle := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
		tp := dimensionTooth(s, left, right, d, false)
		_, err := s.CreateFitSpline(tp.right...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, tp.right[p.Samples-1], tp.left[p.Samples-1])
		_, err = s.CreateFitSpline(tp.left...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded gear profile"); err != nil {
		return nil, nil, nil, err
	}

	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded union valid=%t holes=%d", profile.Valid, len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d curves, want %d", len(profile.Outer), want)
	}

	var contacts []point
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, errors.New("spurgear: embedded flank trim is not certified")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: flank trim radius %g differs from root radius %g",
				math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if want := 2 * p.Teeth; len(contacts) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d root contacts, want %d", len(contacts), want)
	}
	return s, profile, contacts, nil
}

// BuildProfile returns the unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: nil context, world or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// Build constructs the gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("spurgear: nil context or document")
	}
	if err := validate(p); err != nil {
		return nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: placement: %w", err)
	}
	world := sketch.NewWorld()
	lowerPlane := world.XY()
	if p.Placement != nil {
		if lowerPlane, err = world.CreatePlaneFromFrame(frame); err != nil {
			return nil, fmt.Errorf("spurgear: lower plane: %w", err)
		}
	}
	upperFrame, err := shifted(frame, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper frame: %w", err)
	}
	upperPlane, err := world.CreatePlaneFromFrame(upperFrame)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper plane: %w", err)
	}
	d := derive(p)
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lowerPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: lower profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upperPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper profile: %w", err)
	}
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: loft: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: lofted gear is not solid")
	}
	if radius := filletRadius(p, d); radius > 0 {
		if body, err = filletRoots(ctx, body, frame, roots, radius); err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		if body, err = cutBore(ctx, doc, body, p); err != nil {
			return nil, err
		}
	}
	if p.CapChamfer > 0 {
		if body, err = ChamferOuterCaps(ctx, body, p.CapChamfer); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// filletRoots fillets the axial edges that start at the saved root contacts.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if start.Sub(toWorld(frame, root)).Len() <= 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: matched %d root edges, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return filleted, nil
}

const boreMargin = 1.0

// buildBoreProfile draws the bore tool circle on a plane one margin below the gear.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s, "bore sketch"); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("spurgear: bore sketch has %d regions, want 1 valid", len(profiles))
	}
	return s, profiles[0], nil
}

func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, boreMargin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D:   units.Millimeters(p.Thickness + 2*boreMargin),
		Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through p.Thickness in the placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, errors.New("spurgear: nil context, document or body")
	}
	if !(p.BoreDiameter > 0) || !finite(p.BoreDiameter) {
		return nil, fmt.Errorf("spurgear: bore diameter must be positive, got %g", p.BoreDiameter)
	}
	if !(p.Thickness > 0) || !finite(p.Thickness) {
		return nil, fmt.Errorf("spurgear: thickness must be positive, got %g", p.Thickness)
	}
	return cutBore(ctx, doc, body, p)
}

// ChamferOuterCaps chamfers both outer cap loops by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, errors.New("spurgear: nil context or body")
	}
	if !(distance > 0) || !finite(distance) {
		return nil, fmt.Errorf("spurgear: chamfer distance must be positive, got %g", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer caps: %w", err)
	}
	return chamfered, nil
}
