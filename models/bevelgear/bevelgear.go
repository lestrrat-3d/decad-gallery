// Package bevelgear builds tapered involute bevel gears with Decad.
package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
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
	r := p.Module * float64(p.Teeth) / 2
	rMate := p.Module * float64(p.MateTeeth) / 2
	gamma := math.Atan2(math.Sin(p.ShaftAngle)*r, rMate+math.Cos(p.ShaftAngle)*r)
	otherGamma := p.ShaftAngle - gamma
	if float64(p.Teeth) < 5.27*math.Cos(gamma) || float64(p.MateTeeth) < 5.27*math.Cos(otherGamma) {
		return dimensions{}, fmt.Errorf("tooth count is below the pitch-cone minimum")
	}
	coneDist := r / math.Sin(gamma)
	maxWidth := 0.95 * coneDist * math.Min(
		math.Pow(math.Sin(gamma), 2), math.Pow(math.Sin(otherGamma), 2))
	if !finite(coneDist, maxWidth) || maxWidth <= 0 {
		return dimensions{}, fmt.Errorf("pitch cone has no positive face width")
	}
	width := p.FaceWidth
	if width == 0 {
		width = math.Min(math.Hypot(2*r, 2*rMate)/6, maxWidth)
	} else if width > maxWidth {
		return dimensions{}, fmt.Errorf("face width %.6g exceeds maximum %.6g mm", width, maxWidth)
	}
	scale := 1 - width/coneDist
	if !finite(scale, width*math.Cos(gamma)) || scale <= 0 {
		return dimensions{}, fmt.Errorf("toe section collapses at the pitch apex")
	}
	return dimensions{coneAngle: gamma, coneDist: coneDist, width: width,
		scale: scale, toeZ: width * math.Cos(gamma)}, nil
}

// Build adds one member of a bevel pair in its local shaft frame.
// ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	plan, err := prepareBuild(p)
	if err != nil {
		return nil, err
	}
	return buildMember(ctx, doc, p, plan, false)
}

type memberBuildPlan struct {
	geometry     Params
	boreDiameter float64
	spiral       bool
}

func prepareBuild(p Params) (memberBuildPlan, error) {
	validationParams := p
	validationParams.BoreDiameter = 0
	if _, err := derive(validationParams); err != nil {
		return memberBuildPlan{}, err
	}
	geometryParams := validationParams
	geometryParams.SpiralAngle = 0
	if p.BaseHeight > 0 || p.MateBaseHeight > 0 || p.ToeExtension > 0 ||
		p.ToeRadius > 0 || p.MateToeRadius > 0 || p.ToothSpacing > 0 {
		return memberBuildPlan{}, fmt.Errorf("Build does not yet support base-height, toe, or spacing options")
	}
	spiral := p.SpiralAngle > 0
	if spiral {
		if err := validateSourceDefaultSpiral(p); err != nil {
			return memberBuildPlan{}, err
		}
	} else if p.Module != 1 || p.Teeth != 8 || p.MateTeeth != 8 ||
		p.ShaftAngle != math.Pi/2 || p.FaceWidth != 0 ||
		p.PressureAngle != 20*math.Pi/180 || p.Samples != 5 {
		return memberBuildPlan{}, fmt.Errorf("Build currently supports the verified 8/8 straight and source-default 31/31 spiral cases")
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

func buildMember(ctx context.Context, doc *decad.Document, p Params,
	plan memberBuildPlan, pinion bool) (*decad.Body, error) {
	var joined *decad.Body
	var err error
	if plan.spiral {
		joined, err = buildSpiralMember(ctx, doc, p, plan.geometry, pinion)
	} else {
		joined, err = buildStraightMember(ctx, doc, p, plan.geometry)
	}
	if err != nil {
		return nil, err
	}
	if plan.boreDiameter == 0 {
		return joined, nil
	}
	joined, err = cutBore(ctx, doc, joined, plan.boreDiameter)
	if err != nil {
		return nil, fmt.Errorf("cut joined gear bore: %w", err)
	}
	return joined, nil
}

func buildStraightMember(ctx context.Context, doc *decad.Document, p, geometryParams Params) (*decad.Body, error) {
	blank, err := BuildBlank(ctx, doc, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("build reference blank: %w", err)
	}
	tooth, err := BuildPointTooth(ctx, doc, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("loft point tooth: %w", err)
	}
	toeCone, err := BuildToeCone(ctx, doc, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("build toe cone: %w", err)
	}
	toeKeeper, err := decad.Cut(ctx, tooth, toeCone)
	if err != nil {
		return nil, fmt.Errorf("trim tooth at toe: %w", err)
	}
	heelCone, err := BuildHeelCone(ctx, doc, geometryParams)
	if err != nil {
		return nil, fmt.Errorf("build heel cone: %w", err)
	}
	trimmedTooth, err := decad.Intersect(ctx, toeKeeper, heelCone)
	if err != nil {
		return nil, fmt.Errorf("trim tooth at heel: %w", err)
	}

	// The first Union consumes its tooth, so place all pattern copies beforehand.
	pattern := make([]*decad.Body, p.Teeth)
	pattern[0] = trimmedTooth
	for i := 1; i < p.Teeth; i++ {
		turn, err := r3.Rotation(r3.NewVec(1, 0, 0), units.Radians(2*math.Pi*float64(i)/float64(p.Teeth)))
		if err != nil {
			return nil, fmt.Errorf("rotate tooth %d: %w", i+1, err)
		}
		pattern[i], err = trimmedTooth.PlacedCopy(ctx, turn)
		if err != nil {
			return nil, fmt.Errorf("place tooth %d: %w", i+1, err)
		}
	}

	joined := blank
	for i, instance := range pattern {
		joined, err = decad.Union(ctx, joined, instance)
		if err != nil {
			return nil, fmt.Errorf("join tooth %d: %w", i+1, err)
		}
	}
	return joined, nil
}
