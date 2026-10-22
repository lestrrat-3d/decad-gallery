// Package spurgear builds a cylindrical gear with straight involute teeth.
package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// Params describes one spur gear. Lengths are millimeters and angles are
// radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement gives the reference point and plane. Nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default gear parameters.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
		BoreDiameter:  0,
		CapChamfer:    0,
		Placement:     nil,
	}
}

// RejectPlacement returns an error naming model when p carries a placement.
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

// validate applies the input checks and returns the derived dimensions.
func validate(p Params) (dimensions, error) {
	switch {
	case p.Module <= 0:
		return dimensions{}, fmt.Errorf("spurgear: module %v must be positive", p.Module)
	case p.Thickness <= 0:
		return dimensions{}, fmt.Errorf("spurgear: thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return dimensions{}, fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return dimensions{}, fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return dimensions{}, fmt.Errorf("spurgear: pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return dimensions{}, fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return dimensions{}, fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return dimensions{}, fmt.Errorf("spurgear: invalid placement frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return dimensions{}, fmt.Errorf("spurgear: root radius %v must be positive", d.root)
	case d.base <= 0:
		return dimensions{}, fmt.Errorf("spurgear: base radius %v must be positive", d.base)
	case d.tip <= d.base:
		return dimensions{}, fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return dimensions{}, fmt.Errorf("spurgear: bore diameter %v must be below root diameter %v",
			p.BoreDiameter, 2*d.root)
	}
	if rf := filletRadius(p, d); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return dimensions{}, fmt.Errorf("spurgear: root fillet radius %v is not finite", rf)
	}
	return d, nil
}

// solveVerify solves a sketch and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s sketch: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s sketch: %w", stage, err)
	}
	return nil
}

// solveSingleProfile solves, verifies, and returns the sketch's one profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	if err := solveVerify(ctx, s, stage); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s sketch has %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s profile is not valid", stage)
	}
	return profiles[0], nil
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: nil sketch world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: nil sketch plane")
	}
	d, err := validate(p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	return s, profile, err
}

// buildGearProfile draws the whole-gear outline and returns the root contacts.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create outline sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point

	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		leftPts, rightPts := flankPoints(s, left, right)
		leftRoot := scaleToRoot(d, left[0][0], left[0][1])
		rightRoot := scaleToRoot(d, right[0][0], right[0][1])
		leftRootPt := sketchcheck.DimensionArcEnd(s, leftRoot.x, leftRoot.y)
		rightRootPt := sketchcheck.DimensionPoint(s, rightRoot.x, rightRoot.y)
		roots = append(roots, rightRoot, leftRoot)

		if k == 0 {
			firstRight = rightRootPt
		} else {
			s.CreateArc(center, prevLeft, rightRootPt)
		}
		s.CreateLine(rightRootPt, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[last], leftPts[last])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRootPt)
		prevLeft = leftRootPt
	}
	s.CreateArc(center, prevLeft, firstRight)

	profile, err := solveSingleProfile(ctx, s, "outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile draws the original flanks and tip arcs plus a full root
// circle, then unions the bounded regions into one outline.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create embedded sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	var flanks []*sketch.FitSpline

	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		leftPts, rightPts := flankPoints(s, left, right)
		rightSpline, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[last], leftPts[last])
		leftSpline, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank spline: %w", k, err)
		}
		flanks = append(flanks, rightSpline, leftSpline)
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d holes, expected 0", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d boundary curves, expected %d", got, want)
	}

	known := make(map[*sketch.FitSpline]bool, len(flanks))
	for _, f := range flanks {
		known[f] = true
	}
	var roots []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !known[flank] {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded edge %d lies on an unknown spline", i)
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded edge %d has no certified trim (partial %t, exact %t)",
				i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if r := math.Hypot(x, y); math.Abs(r-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded edge %d trims at radius %v, expected %v", i, r, d.root)
		}
		roots = append(roots, point{x, y})
	}
	if got, want := len(roots), 2*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d root contacts, expected %d", got, want)
	}
	return s, profile, roots, nil
}

// Build constructs the gear in doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	d, err := validate(p)
	if err != nil {
		return nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: placement frame: %w", err)
	}
	world := sketch.NewWorld()
	lowerPlane := world.XY()
	if p.Placement != nil {
		lowerPlane, err = world.CreatePlaneFromFrame(frame)
		if err != nil {
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

	var body *decad.Body
	var roots []point
	if d.base >= d.root {
		body, roots, err = buildOrdinary(ctx, doc, world, lowerPlane, upperPlane, frame, p, d)
	} else {
		body, roots, err = buildEmbedded(ctx, doc, world, lowerPlane, upperPlane, p, d)
	}
	if err != nil {
		return nil, err
	}

	if rf := filletRadius(p, d); rf > 0 {
		body, err = filletRoots(ctx, body, frame, roots, rf)
		if err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		body, err = cutBore(ctx, doc, body, p)
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
		return nil, fmt.Errorf("spurgear: finished gear is not a solid")
	}
	return body, nil
}

// buildOrdinary builds the root disc and one lofted tooth, patterns the tooth,
// and joins everything in one union.
func buildOrdinary(ctx context.Context, doc *decad.Document, world *sketch.World,
	lowerPlane, upperPlane *sketch.Plane, frame r3.Frame, p Params, d dimensions) (*decad.Body, []point, error) {
	ds, err := world.CreateSketch(lowerPlane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create root disc sketch: %w", err)
	}
	circle := ds.CreateCircle(ds.Origin(), d.root)
	ds.AddConstraint(sketch.NewRadius(circle, d.root))
	discProfile, err := solveSingleProfile(ctx, ds, "root disc")
	if err != nil {
		return nil, nil, err
	}
	disc, err := decad.Extrude(ctx, doc, ds, discProfile,
		decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}

	lowerSketch, lowerProfile, err := buildTooth(ctx, world, lowerPlane, p, d, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: lower tooth section: %w", err)
	}
	upperSketch, upperProfile, err := buildTooth(ctx, world, upperPlane, p, d, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: upper tooth section: %w", err)
	}
	tooth, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: loft tooth: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: frame.Origin(),
		Axis:   frame.N(),
		Count:  p.Teeth,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: pattern tooth: %w", err)
	}
	teeth := append([]*decad.Body{tooth}, copies...)
	gear, err := decad.Union(ctx, disc, teeth...)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: join disc and %d teeth: %w", len(teeth), err)
	}
	if !gear.IsSolid() {
		return nil, nil, fmt.Errorf("spurgear: joined gear is not a solid")
	}

	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, 0)
	rightRoot := scaleToRoot(d, right[0][0], right[0][1])
	leftRoot := scaleToRoot(d, left[0][0], left[0][1])
	roots := make([]point, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		theta := toothAngle(p, k)
		rx, ry := rotate(rightRoot.x, rightRoot.y, theta)
		lx, ly := rotate(leftRoot.x, leftRoot.y, theta)
		roots = append(roots, point{rx, ry}, point{lx, ly})
	}
	return gear, roots, nil
}

// buildEmbedded lofts identical complete outlines into one solid.
func buildEmbedded(ctx context.Context, doc *decad.Document, world *sketch.World,
	lowerPlane, upperPlane *sketch.Plane, p Params, d dimensions) (*decad.Body, []point, error) {
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lowerPlane, p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: lower outline: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upperPlane, p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: upper outline: %w", err)
	}
	gear, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: loft embedded gear: %w", err)
	}
	if !gear.IsSolid() {
		return nil, nil, fmt.Errorf("spurgear: lofted gear is not a solid")
	}
	return gear, roots, nil
}

// filletRoots fillets the axial edges that start at a saved root contact.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, target := range targets {
			if start.Sub(target).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if got, want := len(corners), len(roots); got != want {
		return nil, fmt.Errorf("spurgear: matched %d root edges, expected %d", got, want)
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet %d root edges: %w", len(corners), err)
	}
	return filleted, nil
}

// buildBoreProfile draws the centered bore circle on a plane set margin below
// the lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
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
		return nil, nil, fmt.Errorf("spurgear: create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	profile, err := solveSingleProfile(ctx, s, "bore")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore extrudes the bore tool and cuts it from gear.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	const margin = 1.0
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	bored, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return bored, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through p.Thickness in the placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("spurgear: bore diameter %v must be positive", p.BoreDiameter)
	}
	return cutBore(ctx, doc, body, p)
}

// ChamferOuterCaps chamfers the outer loops of both caps of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance %v must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return chamfered, nil
}
