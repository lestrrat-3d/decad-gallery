package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
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

// revolveBlankIn revolves the reference blank of p from a sketch placed in c; a nil c leaves it in the root.
func revolveBlankIn(ctx context.Context, doc *decad.Document, c *decad.Container, p Params) (*decad.Body, error) {
	d, err := derive(p)
	if err != nil {
		return nil, err
	}
	section, err := blankSection(p, d)
	if err != nil {
		return nil, err
	}
	return revolveAxialSectionIn(ctx, doc, c, section, "blank")
}

func revolveAxialSection(ctx context.Context, doc *decad.Document,
	section []axialPoint, label string) (*decad.Body, error) {
	return revolveAxialSectionIn(ctx, doc, nil, section, label)
}

// revolveAxialSectionIn revolves section about the X axis from a sketch placed in c; a nil c leaves the sketch
// in the document's root container.
func revolveAxialSectionIn(ctx context.Context, doc *decad.Document, c *decad.Container,
	section []axialPoint, label string) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, fmt.Errorf("create %s sketch: %w", label, err)
	}
	if err := addSketch(c, s); err != nil {
		return nil, err
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
		decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 1, V: 0}}, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("revolve %s section: %w", label, err)
	}
	return body, nil
}

// cleanCoordinate returns v, or zero when v is only the rounding of a rotation by a quarter turn.
func cleanCoordinate(v float64) float64 {
	if math.Abs(v) < 1e-12 {
		return 0
	}
	return v
}

// revolveSharedBlanks revolves both members' blanks from one sketch in c, which holds the common apex at the
// origin, both shaft axes as construction lines, and both axial sections around the apex. The driving section
// lies on the +X axis; the pinion section is rotated by shaftAngle about the apex and revolves about its own
// shaft axis. Each blank is a revolve of its own profile. The bodies land in c.
func revolveSharedBlanks(ctx context.Context, doc *decad.Document, c *decad.Container,
	driving, pinion []axialPoint, shaftAngle float64) (*decad.Body, *decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, nil, fmt.Errorf("create blank pair sketch: %w", err)
	}
	if err := addSketch(c, s); err != nil {
		return nil, nil, err
	}
	sin, cos := math.Sincos(shaftAngle)
	sin, cos = cleanCoordinate(sin), cleanCoordinate(cos)
	reach := 0.0
	for _, section := range [][]axialPoint{driving, pinion} {
		for _, point := range section {
			reach = math.Max(reach, point.station)
		}
	}
	// Each shaft axis runs from the apex, the sketch origin, past its member's heel.
	for _, direction := range [][2]float64{{1, 0}, {cos, sin}} {
		end := sketchcheck.DimensionPoint(s, cleanCoordinate(reach*direction[0]), cleanCoordinate(reach*direction[1]))
		axis := s.CreateLine(s.Origin(), end)
		axis.SetConstruction(true)
	}
	drive := func(section []axialPoint, rotate bool) []*sketch.Line {
		points := make([]*sketch.Point, len(section))
		for i, xy := range section {
			x, y := xy.station, xy.radius
			if rotate {
				x, y = xy.station*cos-xy.radius*sin, xy.station*sin+xy.radius*cos
			}
			points[i] = sketchcheck.DimensionPoint(s, cleanCoordinate(x), cleanCoordinate(y))
		}
		lines := make([]*sketch.Line, len(points))
		for i := range points {
			lines[i] = s.CreateLine(points[i], points[(i+1)%len(points)])
		}
		return lines
	}
	drivingLines := drive(driving, false)
	pinionLines := drive(pinion, true)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve blank pair sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify blank pair sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 2 {
		return nil, nil, fmt.Errorf("blank pair sketch has %d profiles; want one per member", len(profiles))
	}
	revolve := func(own []*sketch.Line, axis decad.SketchLine, label string) (*decad.Body, error) {
		for _, profile := range profiles {
			if !profile.Valid || !profileUses(profile, own[0]) {
				continue
			}
			body, err := decad.Revolve(ctx, doc, s, profile, axis, decad.FullRevolution{})
			if err != nil {
				return nil, fmt.Errorf("revolve %s blank: %w", label, err)
			}
			return body, nil
		}
		return nil, fmt.Errorf("blank pair sketch has no valid %s profile", label)
	}
	drivingBody, err := revolve(drivingLines,
		decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 1, V: 0}}, "driving")
	if err != nil {
		return nil, nil, err
	}
	pinionBody, err := revolve(pinionLines,
		decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: cos, V: sin}}, "pinion")
	if err != nil {
		return nil, nil, err
	}
	return drivingBody, pinionBody, nil
}

// profileUses reports whether line is on the outer boundary of profile.
func profileUses(profile *sketch.Profile, line *sketch.Line) bool {
	for _, entity := range profile.Entities {
		if entity == sketch.Entity(line) {
			return true
		}
	}
	return false
}

// BlankPair holds the driving blank and its pinion around their common pitch apex.
type BlankPair struct {
	Driving *decad.Body
	Pinion  *decad.Body
}

// BuildBlankPair revolves both reference blanks from one sketch around the shared apex, the pinion's section
// rotated by the shaft angle, and puts each blank in the container of its member.
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
	drivingSection, err := blankSection(p, drivingDimensions)
	if err != nil {
		return BlankPair{}, fmt.Errorf("driving: %w", err)
	}
	pinionSection, err := blankSection(pinionParams, pinionDimensions)
	if err != nil {
		return BlankPair{}, fmt.Errorf("pinion: %w", err)
	}
	containers, err := newPairContainers(doc)
	if err != nil {
		return BlankPair{}, err
	}
	driving, pinion, err := revolveSharedBlanks(ctx, doc, containers.pair, drivingSection, pinionSection, p.ShaftAngle)
	if err != nil {
		return BlankPair{}, err
	}
	if driving, err = moveBody(doc, driving, containers.driving); err != nil {
		return BlankPair{}, err
	}
	if pinion, err = moveBody(doc, pinion, containers.pinion); err != nil {
		return BlankPair{}, err
	}
	return BlankPair{Driving: driving, Pinion: pinion}, nil
}

// serialEnds holds the two bodies the serial spiral route needs besides the teeth.
type serialEnds struct{ blank, region *decad.Body }

// buildSerialEnds revolves the extended blank and the trim region of the serial spiral route. The
// extended blank runs past both end planes and both cap cones, and its corners on the root cone continue
// that cone, so no face of it lies on a face of the region. The region is bounded only by the two end
// planes, the toe and heel cap cones and an outer radius no operand reaches; intersecting a member with
// it leaves the planes and cones as the member's end faces. unit is the length of one module.
func buildSerialEnds(ctx context.Context, doc *decad.Document, p Params, unit float64) (serialEnds, error) {
	d, err := derive(p)
	if err != nil {
		return serialEnds{}, err
	}
	section, err := blankSection(p, d)
	if err != nil {
		return serialEnds{}, err
	}
	toeBack, heelBack, heelEdge, ded, toeRoot, toeEdge :=
		section[0], section[1], section[2], section[3], section[4], section[5]
	slopeToe := (toeEdge.radius - toeRoot.radius) / (toeEdge.station - toeRoot.station)
	slopeHeel := (heelEdge.radius - ded.radius) / (heelEdge.station - ded.station)
	interceptToe := toeEdge.radius - slopeToe*toeEdge.station
	interceptHeel := heelEdge.radius - slopeHeel*heelEdge.station
	rootLength := math.Hypot(ded.station-toeRoot.station, ded.radius-toeRoot.radius)
	rootUnit := [2]float64{(ded.station - toeRoot.station) / rootLength, (ded.radius - toeRoot.radius) / rootLength}
	past := unit
	heelCorner := axialPoint{station: heelBack.station + past}
	heelCorner.radius = slopeHeel*heelCorner.station + interceptHeel + past
	toeCorner := axialPoint{station: toeBack.station - past}
	toeCorner.radius = slopeToe*toeCorner.station + interceptToe - past
	if toeCorner.radius <= 0 {
		toeCorner.radius = 0.5 * (slopeToe*toeCorner.station + interceptToe)
	}
	bigSection := []axialPoint{
		{station: toeBack.station - past, radius: 0},
		{station: heelBack.station + past, radius: 0},
		heelCorner,
		{station: ded.station + 1.5*past*rootUnit[0], radius: ded.radius + 1.5*past*rootUnit[1]},
		{station: toeRoot.station - 1.5*past*rootUnit[0], radius: toeRoot.radius - 1.5*past*rootUnit[1]},
		toeCorner,
	}
	blank, err := revolveAxialSection(ctx, doc, bigSection, "extended blank")
	if err != nil {
		return serialEnds{}, err
	}
	reach := 2.5 * (ded.radius + 3*unit)
	region, err := revolveAxialSection(ctx, doc, []axialPoint{
		{station: toeBack.station, radius: 0},
		{station: heelBack.station, radius: 0},
		heelEdge,
		{station: (reach - interceptHeel) / slopeHeel, radius: reach},
		{station: (reach - interceptToe) / slopeToe, radius: reach},
		toeEdge,
	}, "cap region")
	if err != nil {
		return serialEnds{}, err
	}
	return serialEnds{blank: blank, region: region}, nil
}
