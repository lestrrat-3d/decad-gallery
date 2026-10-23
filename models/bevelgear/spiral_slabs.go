package bevelgear

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// spiralSliceFrame holds the root-cone distances used by the source's spiral steps.
type spiralSliceFrame struct {
	coneVec    r3.Vec
	toe, heel  float64
	mean, span float64
	pitchAngle float64
}

// sliceSpiralPointTooth makes the source's eight parent-plane-parallel cuts.
// The apex scrap is removed and the remaining slabs are returned toe to heel.
func sliceSpiralPointTooth(ctx context.Context, doc *decad.Document, p Params,
	tooth *decad.Body) ([]*decad.Body, spiralSliceFrame, error) {
	if ctx == nil || doc == nil || tooth == nil {
		return nil, spiralSliceFrame{}, fmt.Errorf("context, document, and point tooth must not be nil")
	}
	if err := validateSpiralBuild(p); err != nil {
		return nil, spiralSliceFrame{}, err
	}
	straight := p
	straight.SpiralAngle = 0
	straight.BoreDiameter = 0
	d, err := derive(straight)
	if err != nil {
		return nil, spiralSliceFrame{}, err
	}
	section, err := blankSection(straight, d)
	if err != nil {
		return nil, spiralSliceFrame{}, err
	}
	to3 := func(point axialPoint) r3.Vec { return r3.NewVec(point.station, point.radius, 0) }
	toeCone, heelCone := to3(section[4]), to3(section[3])
	toeMid := toeCone.Add(to3(section[5])).Scale(0.5)
	heelMid := heelCone.Add(to3(section[2])).Scale(0.5)
	if heelMid.Len() < toeMid.Len() {
		heelCone = toeCone
		toeMid, heelMid = heelMid, toeMid
	}
	coneVec := heelCone.Scale(1 / heelCone.Len())
	frame := spiralSliceFrame{
		coneVec:    coneVec,
		toe:        toeMid.Dot(coneVec),
		heel:       heelMid.Dot(coneVec),
		pitchAngle: d.coneAngle,
	}
	frame.mean = (frame.toe + frame.heel) / 2
	frame.span = frame.heel - frame.toe
	if frame.span <= 0 {
		return nil, spiralSliceFrame{}, fmt.Errorf("spiral toe-to-heel span must be positive")
	}

	// The back-cone tooth plane has U=(sinγ,-cosγ,0), V=(0,0,1).
	// Its own normal chooses the offset sign; the root-cone element does not.
	u := r3.NewVec(math.Sin(d.coneAngle), -math.Cos(d.coneAngle), 0)
	normal := u.Cross(r3.NewVec(0, 0, 1))
	parent := r3.NewVec(d.coneDist/math.Cos(d.coneAngle), 0, 0)
	sign := 1.0
	if parent.Scale(-1).Dot(normal) < 0 {
		sign = -1
	}
	for attempt := range 2 {
		pieces := []*decad.Body{tooth}
		for k := 1; k <= 8; k++ {
			location := parent.Add(normal.Scale(sign * float64(k) * frame.span / 6))
			sheet, err := spiralSliceSheet(ctx, doc, location, u, 2*d.coneDist)
			if err != nil {
				return nil, spiralSliceFrame{}, fmt.Errorf("slice plane %d: %w", k, err)
			}
			// The planes move monotonically toward the apex. Only the apex-side
			// piece can meet the next plane; all heel-side pieces stay whole.
			inner := pieces[0]
			cut, err := decad.Split(ctx, inner, sheet)
			if errors.Is(err, decad.ErrDegenerate) {
				if removeErr := decad.Remove(sheet); removeErr != nil {
					return nil, spiralSliceFrame{}, fmt.Errorf("remove missed slice sheet %d: %w", k, removeErr)
				}
				continue
			}
			if err != nil {
				_ = decad.Remove(sheet)
				return nil, spiralSliceFrame{}, fmt.Errorf("split tooth at plane %d: %w", k, err)
			}
			if len(cut) != 2 {
				return nil, spiralSliceFrame{}, fmt.Errorf("slice plane %d made %d pieces; want two", k, len(cut))
			}
			pieces[0] = cut[0]
			pieces = append(pieces, cut[1])
		}
		if len(pieces) == 1 {
			if attempt == 0 {
				sign = -sign
				continue
			}
			return nil, spiralSliceFrame{}, fmt.Errorf(
				"spiral slicing produced %d piece after both signs; span=%g mm, last sign=%g",
				len(pieces), frame.span, sign)
		}
		type rankedSlab struct {
			body     *decad.Body
			distance float64
		}
		ranked := make([]rankedSlab, len(pieces))
		for i, piece := range pieces {
			centroid, err := piece.Centroid()
			if err != nil {
				return nil, spiralSliceFrame{}, fmt.Errorf("measure slice piece %d: %w", i, err)
			}
			ranked[i] = rankedSlab{body: piece, distance: centroid.Value.Dot(coneVec)}
		}
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].distance < ranked[j].distance })
		if err := decad.Remove(ranked[0].body); err != nil {
			return nil, spiralSliceFrame{}, fmt.Errorf("remove apex scrap: %w", err)
		}
		segments := make([]*decad.Body, 0, len(ranked)-1)
		for _, slab := range ranked[1:] {
			segments = append(segments, slab.body)
		}
		return segments, frame, nil
	}
	return nil, spiralSliceFrame{}, fmt.Errorf("spiral slicing exhausted both offset signs")
}

// validateSpiralBuild accepts the spiral members the serial join builds: equal tooth counts at module 1,
// a 90° shaft and 15 samples, with any pressure angle, spiral angle, hand, cutter radius, face width,
// base height, tooth spacing, toe extension and toe radius. Decad takes the serial join only with a
// document made with decad.WithChainTolerance.
func validateSpiralBuild(p Params) error {
	if p.Teeth != p.MateTeeth || p.Module != 1 || p.ShaftAngle != math.Pi/2 || p.Samples != 15 ||
		(p.SpiralHand != "right" && p.SpiralHand != "left") {
		return errSpiralUnsupported
	}
	return nil
}

var errSpiralUnsupported = fmt.Errorf("spiral slicing currently supports the source-default 31/31 case " +
	"and equal tooth counts at module 1 with a 90° shaft and 15 samples")

func spiralSliceSheet(ctx context.Context, doc *decad.Document, origin, along r3.Vec,
	extent float64) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, err
	}
	a := origin.Add(along.Scale(-extent))
	b := origin.Add(along.Scale(extent))
	first := sketchcheck.DimensionPoint(s, a.X, a.Y)
	second := sketchcheck.DimensionPoint(s, b.X, b.Y)
	s.CreateLine(first, second)
	if _, err := s.Solve(ctx); err != nil {
		return nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify slice sheet sketch: %w", err)
	}
	chains := s.Chains()
	if len(chains) != 1 {
		return nil, fmt.Errorf("slice sheet has %d chains; want one", len(chains))
	}
	return decad.Extrude(ctx, doc, s, chains[0], decad.TwoSided{
		One: decad.DistanceSide{D: units.Millimeters(extent)},
		Two: decad.DistanceSide{D: units.Millimeters(extent)},
	})
}
