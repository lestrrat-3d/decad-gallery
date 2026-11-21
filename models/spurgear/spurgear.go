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
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters and angles radians.
// Zero fields are taken literally; use DefaultParams for defaults.
type Params struct {
	// Module is the tooth module m.
	Module float64
	// Teeth is the tooth count N.
	Teeth int
	// PressureAngle is the pressure angle α.
	PressureAngle float64
	// Thickness is the axial thickness h.
	Thickness float64
	// Samples is the number of fit points per flank.
	Samples int
	// BoreDiameter is the centered through-bore diameter; zero omits the bore.
	BoreDiameter float64
	// CapChamfer is the equal-distance outer cap setback; zero omits it.
	CapChamfer float64
	// Backlash is the circumferential tooth thinning at the pitch circle.
	// Zero keeps the full half-pitch tooth; a negative value thickens it.
	Backlash float64
	// Placement is the reference point and plane. Its origin lies on the gear
	// axis, its U and V axes span the lower cap plane, and the axis runs along
	// its normal. Nil selects the world XY frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth, module 1 gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
		BoreDiameter:  0,
		CapChamfer:    0,
		Backlash:      0.05,
	}
}

// RejectPlacement returns an error naming model when p carries a Placement.
// Models that reuse Params without placement support call it.
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

// DefineParameters writes the spur gear parameter table of p into t. It does
// not validate p.
func DefineParameters(t *param.Table, p Params) error {
	if t == nil {
		return errors.New("spurgear: nil parameter table")
	}
	return defineParameters(t, p)
}

func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// validate applies the input checks and limits of the construction contract.
func validate(p Params) error {
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spurgear: module %g must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spurgear: thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spurgear: pressure angle %g must lie strictly between 0 and π/2", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %g must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("spurgear: backlash %g must be finite", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash %g exceeds half the module %g", p.Backlash, p.Module)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: placement is not a valid frame")
	}
	d, err := computeDimensions(p)
	if err != nil {
		return fmt.Errorf("spurgear: dimensions: %w", err)
	}
	switch {
	case d.root <= 0:
		return fmt.Errorf("spurgear: root radius %g must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("spurgear: base radius %g must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %g must be smaller than the root circle diameter %g",
			p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spurgear: root fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// Build constructs the gear described by p in doc and returns its solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear described by p in doc through the variant's
// hooks and returns its solid body.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	if v.LocalBuild {
		return v.buildLocal(ctx, doc, p)
	}
	r, err := v.begin(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return v.assemble(ctx, r)
}

// assemble builds tooth zero and the root disc, patterns the tooth, joins
// everything once, and applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("tooth pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join teeth: result is not a solid")
	}

	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
	} else if r.d.fillet > 0 {
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}

	if r.p.BoreDiameter > 0 {
		body, err = cutBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		var setback units.Value
		setback, err = r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("read cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: final body is not a solid")
	}
	return body, nil
}

// buildRootDisc draws the root circle on the lower plane, bound to
// root_radius, and extrudes it through the gear thickness.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("root disc: bind root_radius: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root disc: extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the whole unfilleted gear outline on plane and returns
// the solved, verified sketch and its profile.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	d, err := profileGuards(ctx, world, plane, p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole unfilleted gear outline with every
// tooth turned by an extra twist about the sketch origin. It also returns the
// 2N root contacts, right then left for each tooth in ascending order, in
// sketch coordinates.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: profile: nonfinite twist %g", twist)
	}
	d, err := profileGuards(ctx, world, plane, p)
	if err != nil {
		return nil, nil, nil, err
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (dimensions, error) {
	if ctx == nil {
		return dimensions{}, errors.New("spurgear: profile: nil context")
	}
	if world == nil {
		return dimensions{}, errors.New("spurgear: profile: nil world")
	}
	if plane == nil {
		return dimensions{}, errors.New("spurgear: profile: nil plane")
	}
	if err := validate(p); err != nil {
		return dimensions{}, err
	}
	d, err := computeDimensions(p)
	if err != nil {
		return dimensions{}, fmt.Errorf("spurgear: profile: %w", err)
	}
	return d, nil
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	return buildOrdinaryProfile(ctx, world, plane, p, d, twist)
}

// toothPoints holds the dimensioned sketch points of one ordinary tooth.
type toothPoints struct {
	left, right         []*sketch.Point
	leftRoot, rightRoot *sketch.Point
	leftXY, rightXY     point
}

// dimensionFlanks creates the flank points: every right point with both
// coordinate dimensions, every left point with both except its tip.
func dimensionFlanks(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
	last := len(right) - 1
	rightPts := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == last {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		leftPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return leftPts, rightPts
}

// dimensionOrdinaryTooth creates the flank points and both root contacts of
// an ordinary tooth. The left root receives one dimension; the joining root
// arc supplies its radius.
func dimensionOrdinaryTooth(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var tp toothPoints
	tp.left, tp.right = dimensionFlanks(s, left, right)
	tp.leftXY = rootContact(left[0], d)
	tp.rightXY = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftXY.x, tp.leftXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightXY.x, tp.rightXY.y)
	return tp
}

// buildOrdinaryProfile draws the 6N-curve whole-gear outline for Rb >= Rr.
func buildOrdinaryProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, previousLeft *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		tp := dimensionOrdinaryTooth(s, left, right, d)
		roots = append(roots, tp.rightXY, tp.leftXY)
		last := len(tp.right) - 1

		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, previousLeft, tp.rightRoot)
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear profile: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, tp.right[last], tp.left[last])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear profile: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		previousLeft = tp.leftRoot
	}
	s.CreateArc(center, previousLeft, firstRight)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("gear profile: %d boundary curves, want %d", got, want)
	}
	return s, profile, roots, nil
}

// flankID names one flank spline of an embedded outline.
type flankID struct {
	tooth int
	right bool
}

// buildEmbeddedProfile draws the 4N-curve whole-gear outline for Rb < Rr:
// the original flanks and tip arcs plus one root circle, united into one
// region whose flanks are trimmed at certified root-circle parameters.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: create sketch: %w", err)
	}
	center := s.Origin()
	flanks := make(map[*sketch.FitSpline]flankID, 2*p.Teeth)
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		leftPts, rightPts := dimensionFlanks(s, left, right)
		last := len(rightPts) - 1
		rightFlank, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d right flank: %w", k, err)
		}
		flanks[rightFlank] = flankID{tooth: k, right: true}
		s.CreateArc(center, rightPts[last], leftPts[last])
		leftFlank, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d left flank: %w", k, err)
		}
		flanks[leftFlank] = flankID{tooth: k, right: false}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("embedded profile: no region")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded profile: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded profile: united outline is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded profile: united outline has %d holes, want 0", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded profile: %d boundary curves, want %d", got, want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		id, known := flanks[flank]
		if !known {
			return nil, nil, nil, errors.New("embedded profile: boundary holds an unknown flank spline")
		}
		side := "left"
		slot := 2*id.tooth + 1
		if id.right {
			side = "right"
			slot = 2 * id.tooth
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d %s flank trim is not certified (partial %t, exact %t)",
				id.tooth, side, edge.Partial, edge.TExact)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d %s flank appears twice on the boundary", id.tooth, side)
		}
		x, y := flank.Eval(edge.TStart)
		if diff := math.Abs(math.Hypot(x, y) - d.root); diff > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d %s root contact lies %g off the root circle",
				id.tooth, side, diff)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			side := "left"
			if slot%2 == 0 {
				side = "right"
			}
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d %s flank has no root contact", slot/2, side)
		}
	}
	return s, profile, contacts, nil
}

// buildEmbeddedTooth draws the four-curve natural tooth section for Rb < Rr:
// both flanks from the closing radius to the tip, the tip arc, and the
// closing arc about the gear axis.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	closing, err := closingRadius(d, p.Teeth)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, closing, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	last := len(right) - 1
	rightPts := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == 0 || i == last {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		leftPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}

	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: %d boundary curves, want 4", got)
	}
	return s, profile, nil
}

// solveAndVerify solves s and requires it to be fully constrained with no
// redundant or conflicting constraints.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// singleProfile requires exactly one valid profile in s.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%d profiles, want 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is not valid")
	}
	return profiles[0], nil
}

func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// filletRoots fillets the ordinary-branch axial root edges matched to roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets every axial edge with an endpoint on a saved root
// contact, all together with the given radius.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet: select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	matches := func(pos r3.Vec) bool {
		for _, target := range targets {
			if pos.Sub(target).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range edges {
		if start := edge.Start().Position().Value; matches(start) {
			corners = append(corners, start)
			continue
		}
		if end := edge.End().Position().Value; matches(end) {
			corners = append(corners, end)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d axial root edges, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return filleted, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through p.Thickness in p's placement frame. It defines
// the standard spur parameters on doc first.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: bore: nil document")
	}
	if body == nil {
		return nil, errors.New("spurgear: bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool from the document's existing parameter
// table and cuts it from gear.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("bore: read tool_margin: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore: read bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore: extrude tool: %w", err)
	}
	body, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore: cut: %w", err)
	}
	return body, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the lower
// cap, bound to bore_diameter.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore profile: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore profile: placement: %w", err)
	}
	lowered, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore profile: tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(lowered)
	if err != nil {
		return nil, nil, fmt.Errorf("bore profile: tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore profile: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore profile: bind bore_diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore profile: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore profile: %w", err)
	}
	return s, profile, nil
}

// ChamferOuterCaps bevels the outer loop of both caps of gear with an equal
// setback of distance millimeters. Bore rims stay sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body,
	distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: cap chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: cap chamfer: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	body, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return body, nil
}
