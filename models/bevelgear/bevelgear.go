// Package bevelgear builds tapered involute bevel gears with Decad.
package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one member of a bevel pair. Angles are radians and lengths are millimeters.
type Params struct {
	Module           float64
	Teeth            int
	MateTeeth        int
	ShaftAngle       float64
	FaceWidth        float64
	PressureAngle    float64
	Samples          int
	EnableBore       bool
	BoreDiameter     float64
	MateBoreDiameter float64
	SpiralAngle      float64
	SpiralHand       string
	CutterRadius     float64
	ToothSpacing     float64
	ToeExtension     float64
	ToeRadius        float64
	MateToeRadius    float64
	BaseHeight       float64
	MateBaseHeight   float64

	// Placement anchors the pair. Its origin is the common pitch apex, its U and V axes span
	// the plane that holds both shafts, and the driving shaft runs along U. Nil keeps the
	// world origin, with the shafts in the world XY plane and the driving shaft on +X.
	Placement *r3.Frame
}

// DefaultParams returns the values from instructions.md.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         31,
		MateTeeth:     31,
		ShaftAngle:    math.Pi / 2,
		PressureAngle: 20 * math.Pi / 180,
		Samples:       15,
		EnableBore:    true,
		SpiralAngle:   35 * math.Pi / 180,
		SpiralHand:    "right",
	}
}

type dimensions struct {
	coneAngle float64
	coneDist  float64
	width     float64
	scale     float64
	toeZ      float64

	pitchRadius, matePitchRadius, mateConeAngle, dedendum            float64
	virtualPitch, virtualBase, virtualTip, virtualRoot, virtualTeeth float64
}

func finite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func derive(p Params) (dimensions, error) {
	if !finite(p.Module, p.ShaftAngle, p.FaceWidth, p.PressureAngle,
		p.BoreDiameter, p.SpiralAngle, p.CutterRadius, p.ToothSpacing,
		p.ToeExtension, p.ToeRadius, p.MateToeRadius, p.BaseHeight, p.MateBaseHeight) {
		return dimensions{}, fmt.Errorf("gear parameters must be finite")
	}
	if p.Module <= 0 || p.Teeth < 3 || p.MateTeeth < 3 || p.Samples < 3 {
		return dimensions{}, fmt.Errorf("module must be positive; teeth, mate teeth, and samples must be at least 3")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return dimensions{}, fmt.Errorf("pressure angle must lie between 0 and pi/2 radians")
	}
	if p.ShaftAngle < math.Pi/6 || p.ShaftAngle > 5*math.Pi/6 {
		return dimensions{}, fmt.Errorf("shaft angle must lie between 30 and 150 degrees")
	}
	lesser, greater := math.Min(float64(p.Teeth), float64(p.MateTeeth)),
		math.Max(float64(p.Teeth), float64(p.MateTeeth))
	if p.ShaftAngle >= math.Acos(-lesser/greater) {
		return dimensions{}, fmt.Errorf("shaft angle reaches the pitch-cone limit")
	}
	if p.FaceWidth < 0 || p.BoreDiameter < 0 || p.CutterRadius < 0 ||
		p.ToothSpacing < 0 || p.ToeRadius < 0 || p.MateToeRadius < 0 ||
		p.BaseHeight < 0 || p.MateBaseHeight < 0 {
		return dimensions{}, fmt.Errorf("length inputs cannot be negative")
	}
	if p.SpiralAngle < 0 || p.SpiralAngle >= math.Pi/3 {
		return dimensions{}, fmt.Errorf("spiral angle must lie in [0, 60) degrees")
	}
	if p.ToeExtension < 0 || p.ToeExtension > 100 {
		return dimensions{}, fmt.Errorf("toe extension must lie in [0, 100] percent")
	}
	if p.SpiralAngle > 0 && p.SpiralHand != "right" && p.SpiralHand != "left" {
		return dimensions{}, fmt.Errorf("spiral hand must be right or left")
	}
	t := param.New()
	if err := defineParameters(t, p, p.FaceWidth == 0); err != nil {
		return dimensions{}, fmt.Errorf("define parameters: %w", err)
	}
	read := func(name string) (float64, error) {
		v, err := t.GetValue(name)
		if err != nil {
			return 0, fmt.Errorf("read parameter %s: %w", name, err)
		}
		return v.Base(), nil
	}
	gamma, err := read("cone_angle")
	if err != nil {
		return dimensions{}, err
	}
	otherGamma, err := read("mate_cone_angle")
	if err != nil {
		return dimensions{}, err
	}
	if float64(p.Teeth) < 5.27*math.Cos(gamma) || float64(p.MateTeeth) < 5.27*math.Cos(otherGamma) {
		return dimensions{}, fmt.Errorf("tooth count is below the pitch-cone minimum")
	}
	coneDist, err := read("cone_distance")
	if err != nil {
		return dimensions{}, err
	}
	maxWidth, err := read("max_face_width")
	if err != nil {
		return dimensions{}, err
	}
	if !finite(coneDist, maxWidth) || maxWidth <= 0 {
		return dimensions{}, fmt.Errorf("pitch cone has no positive face width")
	}
	if p.FaceWidth > maxWidth {
		return dimensions{}, fmt.Errorf("face width %.6g exceeds maximum %.6g mm", p.FaceWidth, maxWidth)
	}
	width, err := read("face_width")
	if err != nil {
		return dimensions{}, err
	}
	scale, err := read("toe_scale")
	if err != nil {
		return dimensions{}, err
	}
	toeZ, err := read("toe_height")
	if err != nil {
		return dimensions{}, err
	}
	if !finite(scale, toeZ) || scale <= 0 {
		return dimensions{}, fmt.Errorf("toe section collapses at the pitch apex")
	}
	d := dimensions{coneAngle: gamma, coneDist: coneDist, width: width, scale: scale, toeZ: toeZ}
	for _, e := range []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitchRadius}, {"mate_pitch_radius", &d.matePitchRadius},
		{"mate_cone_angle", &d.mateConeAngle}, {"dedendum", &d.dedendum},
		{"virtual_pitch_radius", &d.virtualPitch}, {"virtual_base_radius", &d.virtualBase},
		{"virtual_tip_radius", &d.virtualTip}, {"virtual_root_radius", &d.virtualRoot},
		{"virtual_teeth", &d.virtualTeeth},
	} {
		if *e.dst, err = read(e.name); err != nil {
			return dimensions{}, err
		}
	}
	return d, nil
}

// Build adds one member of a bevel pair in its local shaft frame, then moves it to
// Params.Placement. ctx and doc must not be nil.
//
// The member is built at the world origin and moved afterwards, because the tooth
// Unions fail on bodies built at an offset.
//
// A spiral member is built by a serial join of overshooting teeth that decad trims once
// with a region bounded by the end planes and cap cones. Decad refuses those serial joins
// unless doc was made with a chain tolerance:
//
//	doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
//
// A 14.5° pressure angle with a 20° spiral angle builds a solid whose mesh VerifyAll cannot
// verify at 0.1 mm, because the flanks cross the end cones at a very shallow angle. When
// decad refuses a boolean, the build retries with the next tooth overshoot and the other
// join order (see spiralOvershoots), so a spiral member can take several builds of about
// a minute each.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	plan, err := prepareBuild(p)
	if err != nil {
		return nil, err
	}
	if err := recordParameters(doc, p); err != nil {
		return nil, err
	}
	member, err := buildMember(ctx, doc, nil, localShaft, nil, p, plan, false)
	if err != nil {
		return nil, err
	}
	return placement.Place(ctx, member, p.Placement)
}

type memberBuildPlan struct {
	geometry     Params
	boreDiameter float64
	spiral       bool
}

func prepareBuild(p Params) (memberBuildPlan, error) {
	if p.Placement != nil && !p.Placement.IsValid() {
		return memberBuildPlan{}, fmt.Errorf("placement frame is not valid")
	}
	validationParams := p
	validationParams.BoreDiameter = 0
	if _, err := derive(validationParams); err != nil {
		return memberBuildPlan{}, err
	}
	geometryParams := validationParams
	geometryParams.SpiralAngle = 0
	spiral := p.SpiralAngle > 0
	if spiral {
		if err := validateSpiralBuild(p); err != nil {
			return memberBuildPlan{}, err
		}
	}
	boreDiameter := 0.0
	if p.EnableBore {
		var err error
		boreDiameter, err = ResolveBoreDiameter(p)
		if err != nil {
			return memberBuildPlan{}, err
		}
	}
	return memberBuildPlan{geometry: geometryParams, boreDiameter: boreDiameter, spiral: spiral}, nil
}

// buildMember builds one member. A straight member draws its sketches in c, a nil c meaning the document's root
// container, and its bodies land there. Its shaft is that of f, and a non-nil blank replaces the blank it would
// revolve itself; the blank must lie in c and revolve about that shaft. A spiral member builds in the root
// container about +X, and f and blank must be localShaft and nil.
func buildMember(ctx context.Context, doc *decad.Document, c *decad.Container, f shaftFrame, blank *decad.Body,
	p Params, plan memberBuildPlan, pinion bool) (*decad.Body, error) {
	var joined *decad.Body
	var err error
	if plan.spiral {
		joined, err = buildSpiralMember(ctx, doc, p, plan.geometry, pinion)
	} else {
		joined, err = buildStraightMember(ctx, doc, c, f, blank, p, plan.geometry)
	}
	if err != nil {
		return nil, err
	}
	if plan.boreDiameter == 0 {
		return joined, nil
	}
	joined, err = cutBore(ctx, doc, joined, f, plan.boreDiameter)
	if err != nil {
		return nil, fmt.Errorf("cut joined gear bore: %w", err)
	}
	return joined, nil
}

func buildStraightMember(ctx context.Context, doc *decad.Document, c *decad.Container, f shaftFrame,
	blank *decad.Body, p, geometryParams Params) (*decad.Body, error) {
	var err error
	if blank == nil {
		blank, err = revolveBlankIn(ctx, doc, c, f, geometryParams)
		if err != nil {
			return nil, fmt.Errorf("build reference blank: %w", err)
		}
	}
	tooth, err := buildPointToothIn(ctx, doc, c, f, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("loft point tooth: %w", err)
	}
	toeCone, err := buildToeConeIn(ctx, doc, c, f, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("build toe cone: %w", err)
	}
	toeKeeper, err := decad.Cut(ctx, tooth, toeCone, decad.WithMatchingFaces())
	if err != nil {
		return nil, fmt.Errorf("trim tooth at toe: %w", err)
	}
	heelCone, err := buildHeelConeIn(ctx, doc, c, f, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("build heel cone: %w", err)
	}
	trimmedTooth, err := decad.Intersect(ctx, toeKeeper, heelCone, decad.WithMatchingFaces())
	if err != nil {
		return nil, fmt.Errorf("trim tooth at heel: %w", err)
	}

	// The first Union consumes its tooth, so place all pattern copies beforehand.
	pattern := make([]*decad.Body, p.Teeth)
	pattern[0] = trimmedTooth
	for i := 1; i < p.Teeth; i++ {
		turn, err := r3.Rotation(f.axis(), units.Radians(2*math.Pi*float64(i)/float64(p.Teeth)))
		if err != nil {
			return nil, fmt.Errorf("rotate tooth %d: %w", i+1, err)
		}
		pattern[i], err = decad.Copy(ctx, trimmedTooth, decad.WithCopyPlacement(turn))
		if err != nil {
			return nil, fmt.Errorf("place tooth %d: %w", i+1, err)
		}
	}
	joined, err := decad.Union(ctx, blank, pattern...)
	if err != nil {
		return nil, fmt.Errorf("join straight tooth pattern: %w", err)
	}
	return joined, nil
}
