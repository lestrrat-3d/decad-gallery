package spurgear

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

// Params specifies gear lengths in millimeters and its pressure angle in radians.
// Geometric inputs must be finite. Zero fields do not request defaults.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
}

// DefaultParams returns a module-one, 17-tooth gear with 10 mm thickness.
func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15}
}

func validateParams(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("gear parameters: require positive module and thickness, at least 3 teeth and 3 samples")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("gear parameters: pressure angle must be between 0 and pi/2")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("gear parameters: bore diameter and cap chamfer must be nonnegative")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("gear parameters: require positive root and base radii and tip radius above base")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("gear parameters: bore diameter must be less than root diameter %g", 2*d.root)
	}
	if !finite(filletRadius(p, d)) {
		return fmt.Errorf("gear parameters: root fillet radius must be finite")
	}
	return nil
}

// Build lofts and finishes a gear spanning z=0 through p.Thickness.
// ctx and doc must not be nil. The document may already contain bodies.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("build gear: context and document must not be nil")
	}
	if err := validateParams(p); err != nil {
		return nil, err
	}
	d := derive(p)
	world := sketch.NewWorld()
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, world.XY(), p, d)
	if err != nil {
		return nil, fmt.Errorf("lower gear profile: %w", err)
	}
	upperPlane, err := planeAt(world, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("upper gear plane: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upperPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("upper gear profile: %w", err)
	}
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("loft gear: %w", err)
	}
	if radius := filletRadius(p, d); radius > 0 {
		body, err = filletRoots(ctx, body, roots, radius)
		if err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		body, err = CutThroughBore(ctx, doc, body, p)
		if err != nil {
			return nil, err
		}
	}
	if p.CapChamfer > 0 {
		body, err = ChamferOuterCaps(ctx, body, p.CapChamfer)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("build gear: resulting body is not solid")
	}
	return body, nil
}

// BuildProfile returns the solved, fully constrained unfilleted gear outline.
// ctx, world, and plane must not be nil.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("build gear profile: context, world, and plane must not be nil")
	}
	if err := validateParams(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

func planeAt(world *sketch.World, z float64) (*sketch.Plane, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create plane: %w", err)
	}
	return plane, nil
}

func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify sketch: %w", err)
	}
	return nil
}

func singleProfile(s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profile count: got %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, fmt.Errorf("profile is invalid")
	}
	if len(profile.Holes) != 0 {
		return nil, fmt.Errorf("profile hole count: got %d, want 0", len(profile.Holes))
	}
	if len(profile.Outer) != curves {
		return nil, fmt.Errorf("profile boundary count: got %d, want %d", len(profile.Outer), curves)
	}
	return profile, nil
}

func flankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
	l := make([]*sketch.Point, len(left))
	r := make([]*sketch.Point, len(right))
	for i, v := range right {
		r[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
	}
	for i, v := range left {
		if i == len(left)-1 {
			l[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
			continue
		}
		l[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
	}
	return l, r
}

func drawFlanks(s *sketch.Sketch, left, right []*sketch.Point, tooth int) error {
	if _, err := s.CreateFitSpline(right...); err != nil {
		return fmt.Errorf("tooth %d right flank: %w", tooth, err)
	}
	s.CreateArc(s.Origin(), right[len(right)-1], left[len(left)-1])
	if _, err := s.CreateFitSpline(left...); err != nil {
		return fmt.Errorf("tooth %d left flank: %w", tooth, err)
	}
	return nil
}

func rootPoints(s *sketch.Sketch, left, right point, d dimensions) (*sketch.Point, *sketch.Point, point, point) {
	l := point{left.x * d.root / d.base, left.y * d.root / d.base}
	r := point{right.x * d.root / d.base, right.y * d.root / d.base}
	return sketchcheck.DimensionArcEnd(s, l.x, l.y), sketchcheck.DimensionPoint(s, r.x, r.y), l, r
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, previousLeft *sketch.Point
	for tooth := range p.Teeth {
		angle := 2 * math.Pi * float64(tooth) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
		l, r := flankPoints(s, left, right)
		leftRoot, rightRoot, leftContact, rightContact := rootPoints(s, left[0], right[0], d)
		roots = append(roots, rightContact, leftContact)
		if tooth == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(s.Origin(), previousLeft, rightRoot)
		}
		s.CreateLine(rightRoot, r[0])
		if err := drawFlanks(s, l, r, tooth); err != nil {
			return nil, nil, nil, err
		}
		s.CreateLine(l[0], leftRoot)
		previousLeft = leftRoot
	}
	s.CreateArc(s.Origin(), previousLeft, firstRight)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, 6*p.Teeth)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded gear sketch: %w", err)
	}
	for tooth := range p.Teeth {
		angle := 2 * math.Pi * float64(tooth) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
		l, r := flankPoints(s, left, right)
		if err := drawFlanks(s, l, r, tooth); err != nil {
			return nil, nil, nil, err
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded region count: got 0, want at least 1")
	}
	indices := make([]int, len(profiles))
	for i, profile := range profiles {
		if !profile.Valid {
			return nil, nil, nil, fmt.Errorf("embedded region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("embedded union profile is invalid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded hole count: got %d, want 0", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded boundary count: got %d, want %d", len(profile.Outer), 4*p.Teeth)
	}
	roots := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded boundary %d lacks a certified spline trim", i)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded boundary %d root radius: got %.17g, want %.17g", i, math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if len(roots) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded root contact count: got %d, want %d", len(roots), 2*p.Teeth)
	}
	return s, profile, roots, nil
}

func filletRoots(ctx context.Context, body *decad.Body, roots []point, radius float64) (*decad.Body, error) {
	axis := r3.NewVec(0, 0, 1)
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if math.Hypot(start.X-root.x, start.Y-root.y) < 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("axial root edge count: got %d, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("fillet gear roots: %w", err)
	}
	return filleted, nil
}

func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	plane, err := planeAt(world, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	profile, err := singleProfile(s, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("bore profile: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered cylinder through a body spanning z=0 through p.Thickness.
// ctx, doc, and body must not be nil; the diameter and thickness must be positive.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, fmt.Errorf("cut through bore: context, document, and body must not be nil")
	}
	if p.BoreDiameter <= 0 || p.Thickness <= 0 {
		return nil, fmt.Errorf("cut through bore: diameter and thickness must be positive")
	}
	const margin = 1.0
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return cut, nil
}

// ChamferOuterCaps chamfers both outer cap loops while leaving bore rims sharp.
// ctx and gear must not be nil; distance must be positive and both caps must identify outer loops.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("chamfer outer caps: context and body must not be nil")
	}
	if distance <= 0 {
		return nil, fmt.Errorf("chamfer outer caps: distance must be positive")
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return chamfered, nil
}
