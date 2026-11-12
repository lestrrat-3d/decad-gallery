// Package spurgear builds a cylindrical gear with straight involute teeth.
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

// Params describes one spur gear. Lengths are millimeters and angles are
// radians. Zero fields are not default requests; use DefaultParams.
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
	// CapChamfer is the equal-distance outer cap setback; zero omits the chamfer.
	CapChamfer float64
	// Backlash thins each tooth circumferentially at the pitch circle. Zero
	// keeps the full half-pitch tooth; a negative value thickens it.
	Backlash float64
	// Placement gives the reference point (origin) and reference plane
	// (U and V). The gear axis runs along the frame normal from the origin.
	// Nil selects the world XY frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default spur gear parameters.
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
// Gear models that reuse Params without placement support call it so that a
// placement is never silently ignored.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// DefineParameters writes the spur gear parameter table of p into t. It does
// not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

func defineParameters(t *param.Table, p Params) error {
	if t == nil {
		return errors.New("spurgear: define parameters: nil table")
	}
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("spurgear: parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("spurgear: parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("spurgear: parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("spurgear: parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("spurgear: parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("spurgear: parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("spurgear: parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("spurgear: parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180",
		units.One); err != nil {
		return fmt.Errorf("spurgear: parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2",
		units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter bore_tool_length: %w", err)
	}
	return nil
}

// readDimensions reads the derived dimensions from the document's table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, entry := range []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch},
		{"base_radius", &d.base},
		{"root_radius", &d.root},
		{"tip_radius", &d.tip},
		{"fillet_radius", &d.fillet},
		{"backlash", &d.backlash},
	} {
		v, err := doc.Param(entry.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("spurgear: read parameter %s: %w", entry.name, err)
		}
		*entry.dst = v.Base()
	}
	return d, nil
}

// validate applies the gear's input checks and limits.
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
		return fmt.Errorf("spurgear: pressure angle must lie strictly between 0 and π/2, got %g", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter must not be negative, got %g", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer must not be negative, got %g", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("spurgear: backlash must be finite, got %g", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash magnitude %g exceeds half the module %g", p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: placement is not a valid frame")
	}
	d := derive(p)
	switch {
	case !(d.root > 0):
		return fmt.Errorf("spurgear: root radius must be positive, got %g", d.root)
	case !(d.base > 0):
		return fmt.Errorf("spurgear: base radius must be positive, got %g", d.base)
	case !(d.tip > d.base):
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %g must be smaller than the root diameter %g",
			p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spurgear: fillet radius is not finite: %g", d.fillet)
	}
	return nil
}

// placement returns the gear's reference frame.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		if !p.Placement.IsValid() {
			return r3.Frame{}, errors.New("spurgear: placement is not a valid frame")
		}
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted returns f moved dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a sketch coordinate on f's plane to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func checkProfileArgs(ctx context.Context, world *sketch.World, plane *sketch.Plane, stage string) error {
	if ctx == nil {
		return fmt.Errorf("spurgear: %s: nil context", stage)
	}
	if world == nil {
		return fmt.Errorf("spurgear: %s: nil world", stage)
	}
	if plane == nil {
		return fmt.Errorf("spurgear: %s: nil plane", stage)
	}
	return nil
}

// BuildProfile draws the whole unfilleted gear outline on plane and returns
// the solved, verified sketch and its profile.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane, "profile"); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole unfilleted gear outline on plane with
// every tooth turned by an extra twist about the sketch origin. It returns
// the solved, verified sketch, its profile, and the 2N root contacts: the
// right root then the left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := checkProfileArgs(ctx, world, plane, "twisted profile"); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twisted profile: twist must be finite, got %g", twist)
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, toPairs(roots), nil
}

// buildGearProfile draws the whole gear outline and returns its root contacts.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole gear outline with every tooth at
// toothAngle(p,k)+twist and returns its root contacts.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	return buildOrdinaryProfile(ctx, world, plane, p, d, twist)
}

func buildOrdinaryProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	const stage = "ordinary outline"
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: create sketch: %w", stage, err)
	}
	center := s.Origin()
	last := p.Samples - 1
	roots := make([]point, 0, 2*p.Teeth)
	var firstRightRoot, previousLeftRoot *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		fp := dimensionFlanks(s, left, right)
		leftRootXY := rootContact(left[0], d)
		rightRootXY := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
		rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)
		roots = append(roots, rightRootXY, leftRootXY)

		if k == 0 {
			firstRightRoot = rightRoot
		} else {
			s.CreateArc(center, previousLeftRoot, rightRoot)
		}
		s.CreateLine(rightRoot, fp.right[0])
		if _, err := s.CreateFitSpline(fp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d right flank: %w", stage, k, err)
		}
		s.CreateArc(center, fp.right[last], fp.left[last])
		if _, err := s.CreateFitSpline(fp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d left flank: %w", stage, k, err)
		}
		s.CreateLine(fp.left[0], leftRoot)
		previousLeftRoot = leftRoot
	}
	s.CreateArc(center, previousLeftRoot, firstRightRoot)

	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, stage)
	if err != nil {
		return nil, nil, nil, err
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: got %d outer curves, expected %d", stage, got, want)
	}
	return s, profile, roots, nil
}

func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	const stage = "embedded outline"
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: create sketch: %w", stage, err)
	}
	center := s.Origin()
	last := p.Samples - 1
	// slot maps each flank to its root contact position: 2k for the right
	// flank of tooth k and 2k+1 for its left flank.
	slot := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		fp := dimensionFlanks(s, left, right)
		rightFlank, err := s.CreateFitSpline(fp.right...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d right flank: %w", stage, k, err)
		}
		s.CreateArc(center, fp.right[last], fp.left[last])
		leftFlank, err := s.CreateFitSpline(fp.left...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d left flank: %w", stage, k, err)
		}
		slot[rightFlank] = 2 * k
		slot[leftFlank] = 2*k + 1
	}
	rootCircle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(rootCircle, d.root))

	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: no regions", stage)
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: region %d is not valid", stage, i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: union regions: %w", stage, err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: union profile is not valid", stage)
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: union profile has %d holes, expected 0", stage,
			len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: got %d boundary curves, expected %d", stage, got, want)
	}
	roots := make([]point, 2*p.Teeth)
	found := make([]bool, 2*p.Teeth)
	count := 0
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		idx, ok := slot[flank]
		if !ok {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: boundary spline is not a drawn flank", stage)
		}
		tooth := idx / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d flank trim is not certified "+
				"(partial %t, exact %t)", stage, tooth, edge.Partial, edge.TExact)
		}
		if found[idx] {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d flank appears twice on the boundary", stage, tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: %s: tooth %d trimmed flank contact at radius %g, expected %g",
				stage, tooth, math.Hypot(x, y), d.root)
		}
		roots[idx] = point{x: x, y: y}
		found[idx] = true
		count++
	}
	if count != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: %s: got %d root contacts, expected %d", stage, count, 2*p.Teeth)
	}
	return s, profile, roots, nil
}

// buildEmbeddedTooth draws the four-curve embedded tooth section at angle 0:
// flanks from the closing radius to the tip, the tip arc, and the closing arc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	const stage = "embedded tooth"
	closing, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: create sketch: %w", stage, err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, closing, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	last := p.Samples - 1
	rightPoints := make([]*sketch.Point, p.Samples)
	for i, q := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPoints := make([]*sketch.Point, p.Samples)
	for i, q := range left {
		if i == 0 || i == last {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: right flank: %w", stage, err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: left flank: %w", stage, err)
	}
	s.CreateArc(center, rightPoints[0], leftPoints[0])

	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, stage)
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("spurgear: %s: got %d outer curves, expected 4", stage, got)
	}
	return s, profile, nil
}

// Build constructs the spur gear in doc and returns its solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear described by p and this variant in doc.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: build: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: build: nil document")
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

// assemble builds tooth zero and the root disc, patterns and joins them,
// then applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("spurgear: pattern tooth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join disc and %d teeth: %w", r.teeth, err)
	}
	if !gear.IsSolid() {
		return nil, errors.New("spurgear: join disc and teeth: result is not a solid")
	}

	switch {
	case v.RootFillet != nil:
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	case r.d.fillet > 0:
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
		if err != nil {
			return nil, err
		}
	}

	if r.p.BoreDiameter > 0 {
		gear, err = cutBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: cap chamfer: read cap_chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, errors.New("spurgear: build: result is not a solid")
	}
	return gear, nil
}

// buildRootDisc draws the root circle on the lower plane, bound to
// root_radius, and extrudes it through the gear thickness.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	const stage = "root disc"
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: create sketch: %w", stage, err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: %s: bind root_radius: %w", stage, err)
	}
	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s, stage)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: extrude: %w", stage, err)
	}
	return disc, nil
}

// filletRoots fillets the ordinary gear's axial root edges with a radius in
// millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges of an ordinary gear that pass
// through the saved root contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	const stage = "root fillet"
	if len(roots) == 0 {
		return nil, fmt.Errorf("spurgear: %s: no root contacts", stage)
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: select axial edges: %w", stage, err)
	}
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	matches := func(q r3.Vec) bool {
		for _, w := range world {
			if q.Sub(w).Len() <= 1e-6 {
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
		return nil, fmt.Errorf("spurgear: %s: matched %d axial root edges, expected %d", stage,
			len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: fillet %d edges: %w", stage, len(corners), err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: embedded root fillet: fillet %d edges: %w", 2*teeth, err)
	}
	return filleted, nil
}

// buildBoreProfile draws the bore circle on the placement plane shifted
// margin below the lower cap. It defines the spur parameters on the world's
// table and binds the circle diameter to bore_diameter.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	const stage = "bore tool"
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: %s: nil context", stage)
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: %s: nil world", stage)
	}
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: %w", stage, err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: %w", stage, err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: shift frame: %w", stage, err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: create plane: %w", stage, err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: create sketch: %w", stage, err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: bind bore_diameter: %w", stage, err)
	}
	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, stage)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which spans the placement's lower cap through p.Thickness along its
// normal. It defines the spur parameters on doc first.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: through-bore: nil context")
	case doc == nil:
		return nil, errors.New("spurgear: through-bore: nil document")
	case body == nil:
		return nil, errors.New("spurgear: through-bore: nil body")
	case !(p.BoreDiameter > 0):
		return nil, fmt.Errorf("spurgear: through-bore: bore diameter must be positive, got %g", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: through-bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore cuts the bore using the document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	const stage = "through-bore"
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: read tool_margin: %w", stage, err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: read bore_tool_length: %w", stage, err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: extrude tool: %w", stage, err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %s: cut: %w", stage, err)
	}
	return cut, nil
}

// ChamferOuterCaps chamfers the outer loops of both caps of gear with an
// equal setback distance in millimeters. Bore rims stay sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: cap chamfer: nil context")
	case gear == nil:
		return nil, errors.New("spurgear: cap chamfer: nil body")
	case math.IsNaN(distance) || math.IsInf(distance, 0) || !(distance > 0):
		return nil, fmt.Errorf("spurgear: cap chamfer: distance must be positive and finite, got %g", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps with setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return chamfered, nil
}
