// Package bevelgear builds tapered involute bevel gears with Decad.
package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Params describes one member of a bevel pair. Angles are radians and lengths are millimeters.
type Params struct {
	Module        float64
	Teeth         int
	MateTeeth     int
	ShaftAngle    float64
	FaceWidth     float64
	PressureAngle float64
	Samples       int
	BoreDiameter  float64
	SpiralAngle   float64
	SpiralHand    string
	CutterRadius  float64
	ToothSpacing  float64
	ToeExtension  float64
	ToeRadius     float64
	BaseHeight    float64
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
		p.ToeExtension, p.ToeRadius, p.BaseHeight) {
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
		p.ToothSpacing < 0 || p.ToeRadius < 0 || p.BaseHeight < 0 {
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
	maxWidth := 0.95 * coneDist * (1 - 1.25*p.Module/r)
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
	if p.BoreDiameter > 0 || p.SpiralAngle > 0 || p.ToothSpacing > 0 ||
		p.ToeExtension > 0 || p.ToeRadius > 0 || p.BaseHeight > 0 {
		return dimensions{}, fmt.Errorf("bore, spiral, spacing, toe, and base-height features are not implemented")
	}
	return dimensions{coneAngle: gamma, coneDist: coneDist, width: width,
		scale: scale, toeZ: width * math.Cos(gamma)}, nil
}

// Build adds one straight bevel gear to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	world := sketch.NewWorld()
	toeFrame, err := r3.NewFrame(r3.NewVec(0, 0, d.toeZ),
		r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create toe frame: %w", err)
	}
	toePlane, err := world.CreatePlaneFromFrame(toeFrame)
	if err != nil {
		return nil, fmt.Errorf("create toe plane: %w", err)
	}
	spurParams := spurgear.Params{
		Module: p.Module, Teeth: p.Teeth, PressureAngle: p.PressureAngle,
		Thickness: 1, Samples: p.Samples,
	}
	heelSketch, heelProfile, err := spurgear.BuildProfile(ctx, world, world.XY(), spurParams)
	if err != nil {
		return nil, fmt.Errorf("build heel profile: %w", err)
	}
	spurParams.Module *= d.scale
	toeSketch, toeProfile, err := spurgear.BuildProfile(ctx, world, toePlane, spurParams)
	if err != nil {
		return nil, fmt.Errorf("build toe profile: %w", err)
	}
	body, err := doc.Loft(ctx, heelSketch, heelProfile, toeSketch, toeProfile)
	if err != nil {
		return nil, fmt.Errorf("loft tapered teeth: %w", err)
	}
	return body, nil
}
