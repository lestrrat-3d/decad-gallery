package bevelgear

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

type axialPoint struct{ station, radius float64 }

func resolvedBaseHeight(p Params, d dimensions, fallback float64) (float64, error) {
	r := d.pitchRadius
	dedendum := d.dedendum
	minBase := 1.05 * dedendum * math.Sin(d.coneAngle)
	maxBase := 0.95 * (r - dedendum*math.Cos(d.coneAngle)) * math.Tan(d.coneAngle)
	baseHeight := p.BaseHeight
	if baseHeight == 0 {
		baseHeight = math.Min(math.Max(fallback, minBase), maxBase)
	} else if baseHeight < minBase || baseHeight > maxBase {
		return 0, fmt.Errorf("base height %.6g mm is outside [%.6g, %.6g] mm",
			baseHeight, minBase, maxBase)
	}
	return baseHeight, nil
}

func blankSection(p Params, d dimensions) ([]axialPoint, error) {
	r := d.pitchRadius
	dedendum := d.dedendum
	baseHeight, err := resolvedBaseHeight(p, d, p.Module*float64(p.Teeth)/8)
	if err != nil {
		return nil, err
	}
	heelStation := d.coneDist*math.Cos(d.coneAngle) + baseHeight
	heelRadius := r - baseHeight/math.Tan(d.coneAngle)
	ded := axialPoint{
		station: d.coneDist*math.Cos(d.coneAngle) + dedendum*math.Sin(d.coneAngle),
		radius:  r - dedendum*math.Cos(d.coneAngle),
	}
	mateGamma := d.mateConeAngle
	rMate := d.matePitchRadius
	toeRadius := p.ToeRadius
	if toeRadius == 0 {
		toeRadius = r - d.width/math.Sin(d.coneAngle)
	}
	mateToeRadius := p.MateToeRadius
	if mateToeRadius == 0 {
		mateToeRadius = rMate - d.width/math.Sin(mateGamma)
	}
	if toeRadius <= 0 || mateToeRadius <= 0 {
		return nil, fmt.Errorf("resolved toe radius must be positive for both pair members")
	}
	apexToDed := math.Hypot(d.coneDist, dedendum)
	rootLength := d.width * apexToDed / d.coneDist
	if p.ToeExtension > 0 {
		ownCeiling := (r - dedendum*math.Cos(d.coneAngle)) * (1 - d.width/d.coneDist)
		mateCeiling := (rMate - dedendum*math.Cos(mateGamma)) * (1 - d.width/d.coneDist)
		if toeRadius >= ownCeiling || mateToeRadius >= mateCeiling {
			return nil, fmt.Errorf("toe extension requires radii below the two toe-radius ceilings")
		}
		rootAngle := d.coneAngle - math.Atan2(dedendum, d.coneDist)
		mateRootAngle := mateGamma - math.Atan2(dedendum, d.coneDist)
		ownLimit := apexToDed - toeRadius/math.Sin(rootAngle)
		mateLimit := apexToDed - mateToeRadius/math.Sin(mateRootAngle)
		limit := math.Min(ownLimit, mateLimit)
		if limit <= rootLength {
			return nil, fmt.Errorf("toe extension has no positive root-length window")
		}
		rootLength += p.ToeExtension / 100 * 0.99 * (limit - rootLength)
	}
	toeFactor := 1 - rootLength/apexToDed
	toeRoot := axialPoint{station: toeFactor * ded.station, radius: toeFactor * ded.radius}
	toeSlide := (toeRoot.radius - toeRadius) / math.Cos(d.coneAngle)
	toeStation := toeRoot.station + toeSlide*math.Sin(d.coneAngle)
	section := []axialPoint{
		{station: toeStation, radius: 0},
		{station: heelStation, radius: 0},
		{station: heelStation, radius: heelRadius},
		ded,
		toeRoot,
		{station: toeStation, radius: toeRadius},
	}
	for _, point := range section {
		if !finite(point.station, point.radius) || point.station <= 0 || point.radius < 0 {
			return nil, fmt.Errorf("revolved blank section has invalid station or radius")
		}
	}
	if heelRadius <= 0 || toeStation >= heelStation || ded.radius <= heelRadius {
		return nil, fmt.Errorf("revolved blank section crosses an end face or shaft axis")
	}
	return section, nil
}

// BuildBlank revolves the reference's six-sided frustum before teeth or a bore are added.
// ctx and doc must not be nil.
func BuildBlank(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	section, err := blankSection(p, d)
	if err != nil {
		return nil, err
	}
	return revolveAxialSection(ctx, doc, section, "blank")
}

func revolveAxialSection(ctx context.Context, doc *decad.Document,
	section []axialPoint, label string) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("create %s sketch: %w", label, err)
	}
	points := make([]*sketch.Point, len(section))
	for i, xy := range section {
		points[i] = sketchcheck.DimensionPoint(s, xy.station, xy.radius)
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve %s sketch: %w", label, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify %s sketch: %w", label, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("%s sketch has %d profiles; want one valid region", label, len(profiles))
	}
	body, err := decad.Revolve(ctx, doc, s, profiles[0],
		decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 1, V: 0}},
		decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("revolve %s section: %w", label, err)
	}
	return body, nil
}

// BlankPair holds the driving blank and its pinion around their common pitch apex.
type BlankPair struct {
	Driving *decad.Body
	Pinion  *decad.Body
}

// BuildBlankPair revolves both reference blanks and rotates the pinion about the shared apex.
func BuildBlankPair(ctx context.Context, doc *decad.Document, p Params) (BlankPair, error) {
	if ctx == nil || doc == nil {
		return BlankPair{}, fmt.Errorf("context and document must not be nil")
	}
	drivingDimensions, err := derive(p)
	if err != nil {
		return BlankPair{}, err
	}
	drivingBase, err := resolvedBaseHeight(p, drivingDimensions, p.Module*float64(p.Teeth)/8)
	if err != nil {
		return BlankPair{}, err
	}
	pinionParams := p
	pinionParams.Teeth, pinionParams.MateTeeth = p.MateTeeth, p.Teeth
	pinionParams.ToeRadius, pinionParams.MateToeRadius = p.MateToeRadius, p.ToeRadius
	pinionParams.BaseHeight = p.MateBaseHeight
	pinionParams.MateBaseHeight = p.BaseHeight
	pinionParams.FaceWidth = drivingDimensions.width
	pinionDimensions, err := derive(pinionParams)
	if err != nil {
		return BlankPair{}, fmt.Errorf("pinion: %w", err)
	}
	pinionBase, err := resolvedBaseHeight(pinionParams, pinionDimensions,
		drivingBase*float64(p.MateTeeth)/float64(p.Teeth))
	if err != nil {
		return BlankPair{}, fmt.Errorf("pinion: %w", err)
	}
	pinionParams.BaseHeight = pinionBase
	if _, err := blankSection(p, drivingDimensions); err != nil {
		return BlankPair{}, fmt.Errorf("driving: %w", err)
	}
	if _, err := blankSection(pinionParams, pinionDimensions); err != nil {
		return BlankPair{}, fmt.Errorf("pinion: %w", err)
	}
	turn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(p.ShaftAngle))
	if err != nil {
		return BlankPair{}, fmt.Errorf("rotate pinion: %w", err)
	}
	driving, err := BuildBlank(ctx, doc, p)
	if err != nil {
		return BlankPair{}, fmt.Errorf("build driving blank: %w", err)
	}
	pinion, err := BuildBlank(ctx, doc, pinionParams)
	if err != nil {
		return BlankPair{}, fmt.Errorf("build pinion blank: %w", err)
	}
	pinion, err = decad.Transform(ctx, pinion, turn)
	if err != nil {
		return BlankPair{}, fmt.Errorf("place pinion blank: %w", err)
	}
	return BlankPair{Driving: driving, Pinion: pinion}, nil
}
