// Package spurgear builds a cylindrical spur gear with straight involute teeth
// from Decad operations: a root disc, one lofted tooth, a circular pattern and a
// single union, followed by root fillets, an optional bore and cap chamfers.
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

// toolMargin is the axial overrun of the bore tool on each side, in
// millimeters. It mirrors the tool_margin parameter.
const toolMargin = 1.0

// Params describes one spur gear. Lengths are millimeters and angles radians.
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

// DefaultParams returns the default 17-tooth gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

// RejectPlacement returns an error naming model when p carries a Placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// placement returns the gear's reference frame.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted moves f by dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a point in the frame's plane to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and returns the derived dimensions.
func validate(p Params) (dimensions, error) {
	switch {
	case p.Module <= 0:
		return dimensions{}, fmt.Errorf("invalid module %v: must be positive", p.Module)
	case p.Thickness <= 0:
		return dimensions{}, fmt.Errorf("invalid thickness %v: must be positive", p.Thickness)
	case p.Teeth < 3:
		return dimensions{}, fmt.Errorf("invalid tooth count %d: must be at least 3", p.Teeth)
	case p.Samples < 3:
		return dimensions{}, fmt.Errorf("invalid samples %d: must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return dimensions{}, fmt.Errorf("invalid pressure angle %v: must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return dimensions{}, fmt.Errorf("invalid bore diameter %v: must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return dimensions{}, fmt.Errorf("invalid cap chamfer %v: must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return dimensions{}, fmt.Errorf("invalid placement frame")
	}
	d, err := deriveDimensions(p)
	if err != nil {
		return dimensions{}, fmt.Errorf("derive dimensions: %w", err)
	}
	switch {
	case d.root <= 0:
		return dimensions{}, fmt.Errorf("invalid root radius %v: must be positive", d.root)
	case d.base <= 0:
		return dimensions{}, fmt.Errorf("invalid base radius %v: must be positive", d.base)
	case d.tip <= d.base:
		return dimensions{}, fmt.Errorf("invalid tip radius %v: must exceed base radius %v", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return dimensions{}, fmt.Errorf("invalid bore diameter %v: must be below root diameter %v",
			p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return dimensions{}, fmt.Errorf("invalid fillet radius %v: not finite", d.fillet)
	}
	return d, nil
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth int) (float64, error) {
	beta := flankBeta(d.pitch, d.base, float64(teeth))
	half := func(r float64) float64 {
		q := rotate(mirroredInvolute(d.base, r), beta)
		return math.Atan2(q.y, q.x)
	}
	limit := math.Pi / float64(teeth)
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("no closing radius: tooth flanks overlap neighbours at the root circle "+
			"(radius %v)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws one embedded tooth closed by an arc at the closing
// radius.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(d, p.Teeth)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	left, right := sampleRange(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	var rightPts, leftPts []*sketch.Point
	for _, q := range right {
		rightPts = append(rightPts, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			leftPts = append(leftPts, sketchcheck.DimensionArcEnd(s, q.x, q.y))
			continue
		}
		leftPts = append(leftPts, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	center := s.Origin()
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	if err := solveVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if n := len(profile.Outer); n != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d outer curves, expected 4", n)
	}
	return s, profile, nil
}

// buildGearProfile draws the whole-gear outline on plane. It returns the
// sketch, the outline profile and the actual root contacts, right then left
// for each tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	var roots []point
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		t := dimensionOrdinaryTooth(s, d, left, right)
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(center, prevLeft, t.rightRoot)
		}
		if err := drawOrdinaryTooth(s, t); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d: %w", k, err)
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws every flank and tip arc plus the root circle,
// unions the bounded regions and records the exact root contacts.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		var rightPts, leftPts []*sketch.Point
		for _, q := range right {
			rightPts = append(rightPts, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
		for i, q := range left {
			if i == len(left)-1 {
				leftPts = append(leftPts, sketchcheck.DimensionArcEnd(s, q.x, q.y))
				continue
			}
			leftPts = append(leftPts, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union has %d holes, expected 0", len(profile.Holes))
	}
	if n := len(profile.Outer); n != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, expected %d", n, 4*p.Teeth)
	}
	var contacts []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d spline trim is not certified "+
				"(partial %v, exact %v)", i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d trim radius %v is not the root radius %v",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if len(contacts) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d root contacts, expected %d",
			len(contacts), 2*p.Teeth)
	}
	return s, profile, contacts, nil
}

// BuildProfile draws the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("gear profile: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("gear profile: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("gear profile: nil plane")
	}
	d, err := validate(p)
	if err != nil {
		return nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// buildRootDisc extrudes the root disc from the lower plane.
func buildRootDisc(ctx context.Context, doc *decad.Document, world *sketch.World, plane *sketch.Plane,
	d dimensions, thickness units.Value) (*decad.Body, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	radius := sketch.NewRadius(circle, d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("root disc: bind radius: %w", err)
	}
	if err := solveVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := onlyProfile(s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: thickness, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("root disc: extrude: %w", err)
	}
	return disc, nil
}

// Build constructs the gear in doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spur gear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spur gear: nil document")
	}
	if _, err := validate(p); err != nil {
		return nil, fmt.Errorf("spur gear: %w", err)
	}
	world := sketch.NewWorld()
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spur gear: document parameters: %w", err)
	}
	// World-owned sketches bind dimensions to the world's table, so it carries
	// the same definitions.
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, fmt.Errorf("spur gear: sketch parameters: %w", err)
	}
	d, err := readDimensions(doc)
	if err != nil {
		return nil, fmt.Errorf("spur gear: %w", err)
	}
	teeth, err := doc.ParamInt("teeth")
	if err != nil {
		return nil, fmt.Errorf("spur gear: %w", err)
	}
	samples, err := doc.ParamInt("samples")
	if err != nil {
		return nil, fmt.Errorf("spur gear: %w", err)
	}
	p.Teeth, p.Samples = teeth, samples
	thickness, err := doc.Param("thickness")
	if err != nil {
		return nil, fmt.Errorf("spur gear: %w", err)
	}

	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spur gear: placement: %w", err)
	}
	lower := world.XY()
	if p.Placement != nil {
		if lower, err = world.CreatePlaneFromFrame(frame); err != nil {
			return nil, fmt.Errorf("spur gear: lower plane: %w", err)
		}
	}
	upperFrame, err := shifted(frame, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("spur gear: upper frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(upperFrame)
	if err != nil {
		return nil, fmt.Errorf("spur gear: upper plane: %w", err)
	}

	disc, err := buildRootDisc(ctx, doc, world, lower, d, thickness)
	if err != nil {
		return nil, fmt.Errorf("spur gear: %w", err)
	}
	embedded := d.base < d.root
	var lowerSketch, upperSketch *sketch.Sketch
	var lowerProfile, upperProfile *sketch.Profile
	if embedded {
		lowerSketch, lowerProfile, err = buildEmbeddedTooth(ctx, world, lower, p, d)
		if err == nil {
			upperSketch, upperProfile, err = buildEmbeddedTooth(ctx, world, upper, p, d)
		}
	} else {
		lowerSketch, lowerProfile, err = buildOrdinaryTooth(ctx, world, lower, p, d, 0)
		if err == nil {
			upperSketch, upperProfile, err = buildOrdinaryTooth(ctx, world, upper, p, d, 0)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("spur gear: tooth sections: %w", err)
	}
	tooth, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spur gear: loft tooth: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth,
		decad.CircularPattern{Center: frame.Origin(), Axis: frame.N(), Count: teeth})
	if err != nil {
		return nil, fmt.Errorf("spur gear: pattern teeth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spur gear: join teeth: %w", err)
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("spur gear: joined body is not solid")
	}

	if d.fillet > 0 {
		radius, err := doc.Param("fillet_radius")
		if err != nil {
			return nil, fmt.Errorf("spur gear: %w", err)
		}
		if embedded {
			gear, err = filletEmbedded(ctx, gear, frame, teeth, radius)
		} else {
			var roots []point
			for k := 0; k < teeth; k++ {
				left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(teeth), p.Samples, 0)
				theta := toothAngle(p, k)
				roots = append(roots,
					rotate(scaleToRoot(right[0], d), theta),
					rotate(scaleToRoot(left[0], d), theta))
			}
			gear, err = filletRootsValue(ctx, gear, frame, roots, radius)
		}
		if err != nil {
			return nil, fmt.Errorf("spur gear: %w", err)
		}
	}
	if p.BoreDiameter > 0 {
		gear, err = cutBore(ctx, doc, gear, p)
		if err != nil {
			return nil, fmt.Errorf("spur gear: %w", err)
		}
	}
	if p.CapChamfer > 0 {
		setback, err := doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spur gear: %w", err)
		}
		gear, err = chamferOuterCaps(ctx, gear, setback)
		if err != nil {
			return nil, fmt.Errorf("spur gear: %w", err)
		}
	}
	return gear, nil
}

// filletEmbedded fillets the concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return out, nil
}

// filletRoots fillets the ordinary-branch root edges matching roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if start.Sub(toWorld(frame, root)).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("fillet roots: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore tool circle on a plane one margin below the
// gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore tool: bind diameter: %w", err)
	}
	if err := solveVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore extrudes the bore tool and cuts it from gear. The document's
// parameter table must already hold the gear definitions.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, toolMargin)
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore tool: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool: extrude: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore tool: cut: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the gear's lower cap through its thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("through bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("through bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("through bore: nil body")
	}
	if p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("through bore: diameter %v must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cap chamfer: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("cap chamfer: nil body")
	}
	if !(distance > 0) {
		return nil, fmt.Errorf("cap chamfer: distance %v must be positive", distance)
	}
	return chamferOuterCaps(ctx, gear, units.Millimeters(distance))
}

func chamferOuterCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}
