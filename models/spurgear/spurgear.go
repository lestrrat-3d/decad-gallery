// Package spurgear builds straight-toothed involute gears with decad.
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

// Params holds the dimensions of one spur gear. Angles are radians; lengths are millimeters.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
}

// DefaultParams returns the values from instructions.md.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

func (p Params) validate() error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("module and thickness must be positive; teeth and samples must be at least 3")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle must lie between 0 and pi/2 radians")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer cannot be negative")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("gear radii are invalid")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter must be smaller than the root diameter")
	}
	radius := filletRadius(p, d)
	if math.IsNaN(radius) || math.IsInf(radius, 0) {
		return fmt.Errorf("root fillet radius is not finite")
	}
	return nil
}

func filletRadius(p Params, d dimensions) float64 {
	angle := math.Pi/float64(p.Teeth) - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)
	return 0.9 * d.root * angle / 2
}

// Build adds one gear to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	d := derive(p)
	world := sketch.NewWorld()
	frame, err := r3.NewFrame(r3.NewVec(0, 0, p.Thickness), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create upper gear frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create upper gear plane: %w", err)
	}
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, world.XY(), p, d)
	if err != nil {
		return nil, fmt.Errorf("lower gear profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upper, p, d)
	if err != nil {
		return nil, fmt.Errorf("upper gear profile: %w", err)
	}
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("loft gear profile: %w", err)
	}
	if radius := filletRadius(p, d); radius > 0 {
		body, err = roundRootCorners(ctx, body, roots, radius)
		if err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		body, err = cutThroughBore(ctx, doc, body, p)
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
	return body, nil
}

// BuildProfile draws one involute outline on plane. ctx, world, and plane must not be nil.
// The caller owns the resulting Sketch profile and may use it in a Decad loft.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane must not be nil")
	}
	if err := p.validate(); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// ChamferOuterCaps bevels the outer loops of both gear end caps.
// ctx and gear must not be nil; distance is in millimeters.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).
		Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	body, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("chamfer outer cap loops: %w", err)
	}
	return body, nil
}

func cutThroughBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	const margin = 1.0
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	body, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("cut through-bore: %w", err)
	}
	return body, nil
}

func buildBoreProfile(ctx context.Context, world *sketch.World, p Params, margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, 0, -margin), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, nil, fmt.Errorf("create bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	center := s.Origin()
	circle := s.CreateCircle(center, p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve bore sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify bore sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("bore sketch has %d profiles; want one valid region", len(profiles))
	}
	return s, profiles[0], nil
}

// CutThroughBore cuts a centered bore through a body spanning z=0 to p.Thickness.
// ctx, doc, and body must not be nil.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	return cutThroughBore(ctx, doc, body, p)
}

func roundRootCorners(ctx context.Context, body *decad.Body, roots []point, radius float64) (*decad.Body, error) {
	axis := r3.NewVec(0, 0, 1)
	axial, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial gear edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range axial {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if math.Hypot(start.X-root.x, start.Y-root.y) < 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("found %d axial root corners; want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	rounded, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("fillet root corners: %w", err)
	}
	return rounded, nil
}

// dimensionPoint locates a computed involute sample through editable dimensions.
func dimensionPoint(s *sketch.Sketch, xy point) *sketch.Point {
	return sketchcheck.DimensionPoint(s, xy.x, xy.y)
}

// dimensionArcEnd leaves the radial coordinate to the arc's own radius relation.
func dimensionArcEnd(s *sketch.Sketch, xy point) *sketch.Point {
	return sketchcheck.DimensionArcEnd(s, xy.x, xy.y)
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedGearProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	center := s.Origin()
	var firstRoot, previousRoot *sketch.Point
	var roots []point
	for tooth := range p.Teeth {
		angle := float64(tooth) * 2 * math.Pi / float64(p.Teeth)
		left, right := toothFlanks(p, d, angle)
		leftPoints := make([]*sketch.Point, len(left))
		rightPoints := make([]*sketch.Point, len(right))
		for i := range left {
			if i == len(left)-1 {
				leftPoints[i] = dimensionArcEnd(s, left[i])
			} else {
				leftPoints[i] = dimensionPoint(s, left[i])
			}
			rightPoints[i] = dimensionPoint(s, right[i])
		}
		rightRootXY := point{right[0].x * d.root / d.base, right[0].y * d.root / d.base}
		leftRootXY := point{left[0].x * d.root / d.base, left[0].y * d.root / d.base}
		rightRoot, leftRoot := dimensionPoint(s, rightRootXY), dimensionArcEnd(s, leftRootXY)
		roots = append(roots, rightRootXY, leftRootXY)
		if previousRoot != nil {
			s.CreateArc(center, previousRoot, rightRoot)
		} else {
			firstRoot = rightRoot
		}
		s.CreateLine(rightRoot, rightPoints[0])
		if _, err := s.CreateFitSpline(rightPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", tooth, err)
		}
		s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
		if _, err := s.CreateFitSpline(leftPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", tooth, err)
		}
		s.CreateLine(leftPoints[0], leftRoot)
		previousRoot = leftRoot
	}
	s.CreateArc(center, previousRoot, firstRoot)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve gear profile: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify gear profile: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("gear sketch has %d profiles; want one valid region", len(profiles))
	}
	return s, profiles[0], roots, nil
}

func buildEmbeddedGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded gear sketch: %w", err)
	}
	center := s.Origin()
	for tooth := range p.Teeth {
		angle := float64(tooth) * 2 * math.Pi / float64(p.Teeth)
		left, right := toothFlanks(p, d, angle)
		leftPoints := make([]*sketch.Point, len(left))
		rightPoints := make([]*sketch.Point, len(right))
		for i := range left {
			if i == len(left)-1 {
				leftPoints[i] = dimensionArcEnd(s, left[i])
			} else {
				leftPoints[i] = dimensionPoint(s, left[i])
			}
			rightPoints[i] = dimensionPoint(s, right[i])
		}
		if _, err := s.CreateFitSpline(rightPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", tooth, err)
		}
		s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
		if _, err := s.CreateFitSpline(leftPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", tooth, err)
		}
	}
	rootCircle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(rootCircle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded gear profile: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded gear profile: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded gear region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded gear regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded gear outline has %d edges and %d holes; want %d edges and no holes",
			len(profile.Outer), len(profile.Holes), 4*p.Teeth)
	}
	roots := make([]point, 0, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded flank lacks an exact root-circle trim")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded flank trim misses root circle")
		}
		roots = append(roots, point{x, y})
	}
	if len(roots) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("found %d embedded root corners; want %d", len(roots), 2*p.Teeth)
	}
	return s, profile, roots, nil
}
