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

// Params describes one cylindrical gear with straight involute teeth.
// Lengths are in millimeters and angles in radians.
type Params struct {
	// Module is the tooth module.
	Module float64
	// Teeth is the tooth count.
	Teeth int
	// PressureAngle is the pressure angle.
	PressureAngle float64
	// Thickness is the axial thickness.
	Thickness float64
	// Samples is the number of fit points per flank.
	Samples int
	// BoreDiameter is the centered through-bore diameter; zero omits the bore.
	BoreDiameter float64
	// CapChamfer is the equal-distance outer cap setback; zero omits it.
	CapChamfer float64
	// Backlash thins each tooth by this arc at the pitch circle.
	Backlash float64
	// Placement gives the reference point and plane; nil selects the world XY
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
		Backlash:      0.05,
	}
}

// RejectPlacement returns an error naming model when p carries a Placement.
// Gear models that reuse Params without placement support call it so a
// placement is never silently ignored.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// DefineParameters writes the parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// validate applies the input checks and limits.
func validate(p Params) error {
	switch {
	case !(p.Module > 0) || math.IsInf(p.Module, 0):
		return fmt.Errorf("spur gear: module %g must be positive and finite", p.Module)
	case !(p.Thickness > 0) || math.IsInf(p.Thickness, 0):
		return fmt.Errorf("spur gear: thickness %g must be positive and finite", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spur gear: %d teeth, need at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spur gear: %d samples per flank, need at least 3", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("spur gear: pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	case !(p.BoreDiameter >= 0) || math.IsInf(p.BoreDiameter, 0):
		return fmt.Errorf("spur gear: bore diameter %g must be nonnegative and finite", p.BoreDiameter)
	case !(p.CapChamfer >= 0) || math.IsInf(p.CapChamfer, 0):
		return fmt.Errorf("spur gear: cap chamfer %g must be nonnegative and finite", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spur gear: backlash %g must be finite with magnitude at most module/2 (%g)",
			p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spur gear: placement is not a valid frame")
	}
	mn := p.Module * float64(p.Teeth)
	pitch := mn / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (mn - 2.5*p.Module) / 2
	tip := (mn + 2*p.Module) / 2
	involuteAngle := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/float64(p.Teeth) - 2*involuteAngle) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("spur gear: root radius %g must be positive", root)
	case base <= 0:
		return fmt.Errorf("spur gear: base radius %g must be positive", base)
	case tip <= base:
		return fmt.Errorf("spur gear: tip radius %g must exceed base radius %g", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("spur gear: bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*root)
	case math.IsNaN(fillet) || math.IsInf(fillet, 0):
		return fmt.Errorf("spur gear: root fillet radius %g is not finite", fillet)
	}
	return nil
}

// placement returns the placement frame of p.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted returns f moved dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a planar coordinate in the gear frame to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func checkProfileArgs(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	switch {
	case ctx == nil:
		return errors.New("spur gear: nil context")
	case world == nil:
		return errors.New("spur gear: nil sketch world")
	case plane == nil:
		return errors.New("spur gear: nil sketch plane")
	}
	return nil
}

// toothPoints holds the sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// dimensionFlanks creates the dimensioned flank points. Every right point gets
// both coordinates. Every left point does too, except its tip, which gets one,
// and its first point when closing is set.
func dimensionFlanks(s *sketch.Sketch, left, right []point, closing bool) ([]*sketch.Point, []*sketch.Point) {
	rp := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	lp := make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == len(left)-1 || (closing && i == 0) {
			lp[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
	}
	return lp, rp
}

// dimensionTooth creates the flank and root points of one ordinary tooth.
func dimensionTooth(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var tp toothPoints
	tp.left, tp.right = dimensionFlanks(s, left, right, false)
	tp.leftRootXY = rootContact(left[0], d)
	tp.rightRootXY = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootXY.x, tp.leftRootXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootXY.x, tp.rightRootXY.y)
	return tp
}

// solveVerify solves s and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// singleProfile requires s to hold exactly one valid profile.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%d profiles, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is not valid")
	}
	return profiles[0], nil
}

// solveProfile solves and verifies s, then returns its single valid profile.
func solveProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if err := solveVerify(ctx, s); err != nil {
		return nil, err
	}
	return singleProfile(s)
}

// Build builds the gear described by p into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear described by p into doc.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spur gear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spur gear: nil document")
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

// assemble builds the gear body from a prepared run.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
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
		return nil, fmt.Errorf("tooth join: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("tooth join: result is not a solid")
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
			return nil, fmt.Errorf("bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("read parameter cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, fmt.Errorf("cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spur gear: result is not a solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root disc on the lower cap plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind radius: %w", err)
	}
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile returns the whole unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 {
		return nil, nil, errors.New("gear profile: cannot derive gear dimensions")
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile returns the whole unfilleted gear outline with every
// tooth turned by an extra twist about the sketch origin, with its 2N root
// contacts: the right root then the left root of each tooth, in tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("gear profile: twist %g is not finite", twist)
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 {
		return nil, nil, nil, errors.New("gear profile: cannot derive gear dimensions")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

// buildGearProfile builds the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with every tooth turned
// by an extra twist, for both the ordinary and the embedded branch.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		tp := dimensionTooth(s, left, right, d)
		roots = append(roots, tp.rightRootXY, tp.leftRootXY)
		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, prevLeft, tp.rightRoot)
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, tp.right[len(tp.right)-1], tp.left[len(tp.left)-1])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		prevLeft = tp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: %d boundary curves, expected %d", len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline builds the whole-gear outline when the flanks start
// inside the root circle. It returns the exact trimmed root contacts.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline sketch: %w", err)
	}
	center := s.Origin()
	slots := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	splines := make([]*sketch.FitSpline, 2*p.Teeth)
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		lp, rp := dimensionFlanks(s, left, right, false)
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		slots[rs] = 2 * k
		slots[ls] = 2*k + 1
		splines[2*k] = rs
		splines[2*k+1] = ls
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("embedded outline: no profile regions")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline union: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, errors.New("embedded outline: union is not a valid hole-free region")
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, known := slots[spline]
		if !known {
			return nil, nil, nil, errors.New("embedded outline: boundary spline belongs to no tooth")
		}
		tooth := slot / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root trim is not certified", tooth)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank trimmed twice", tooth)
		}
		x, y := splines[slot].Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact is off the root circle", tooth)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has no root contact", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// buildEmbeddedTooth draws one tooth whose flanks start on a closing arc inside
// the root disc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	lp, rp := dimensionFlanks(s, left, right, true)
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: %d outer curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// buildBoreProfile builds the bore tool sketch on a plane margin below the
// lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the parameters already defined on doc.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read parameter bore_tool_length: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through p.Thickness of the placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("bore: nil context")
	case doc == nil:
		return nil, errors.New("bore: nil document")
	case body == nil:
		return nil, errors.New("bore: nil body")
	case !(p.BoreDiameter > 0):
		return nil, fmt.Errorf("bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("chamfer: nil context")
	case gear == nil:
		return nil, errors.New("chamfer: nil body")
	case !(distance > 0) || math.IsInf(distance, 0):
		return nil, fmt.Errorf("chamfer: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers both outer cap loops of gear by setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return chamfered, nil
}

// filletRoots fillets the ordinary root edges with a radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end at a saved root
// contact on the lower cap.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	onRoot := func(pos r3.Vec) bool {
		for _, w := range world {
			if w.Sub(pos).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if onRoot(start) {
			corners = append(corners, start)
			continue
		}
		end := edge.End().Position().Value
		if onRoot(end) {
			corners = append(corners, end)
		}
	}
	if len(corners) == 0 || len(corners) != len(roots) {
		return nil, fmt.Errorf("root edges: matched %d of %d root contacts", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2*teeth concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return filleted, nil
}
