package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// GearPair holds the driving member and pinion around their common pitch apex.
type GearPair struct {
	Driving *decad.Body
	Pinion  *decad.Body
}

// BuildPair builds the pinion first, then phases the driving gear by half a tooth pitch.
// A pinion with an odd tooth count is also turned by one tooth width about its shaft.
// It then moves both members with Params.Placement.
//
// The pair is a container named "Pair" in the document's root, holding a container "Driving Gear" and a container
// "Pinion Gear". A straight pair revolves both blanks from one shared sketch in "Pair", which holds the common
// apex and both axial sections, the pinion's about its own shaft. Each member then draws its tooth, cone and bore
// sketches in its own container, so its bodies are there too. A spiral member is built in the root container
// about +X and moved to its own when it is done; the pinion is then turned onto its shaft. A document that already
// holds a "Pair" gets "Pair 2" for the next.
func BuildPair(ctx context.Context, doc *decad.Document, p Params) (GearPair, error) {
	if ctx == nil || doc == nil {
		return GearPair{}, fmt.Errorf("context and document must not be nil")
	}
	pinionParams := p
	pinionParams.Teeth, pinionParams.MateTeeth = p.MateTeeth, p.Teeth
	pinionParams.ToeRadius, pinionParams.MateToeRadius = p.MateToeRadius, p.ToeRadius
	pinionParams.BaseHeight, pinionParams.MateBaseHeight = p.MateBaseHeight, p.BaseHeight
	pinionParams.BoreDiameter, pinionParams.MateBoreDiameter = p.MateBoreDiameter, p.BoreDiameter

	// Validate both members before adding either body to the document.
	drivingPlan, err := prepareBuild(p)
	if err != nil {
		return GearPair{}, fmt.Errorf("validate driving gear: %w", err)
	}
	pinionPlan, err := prepareBuild(pinionParams)
	if err != nil {
		return GearPair{}, fmt.Errorf("validate pinion: %w", err)
	}
	if err := recordParameters(doc, p); err != nil {
		return GearPair{}, err
	}
	if drivingPlan.spiral {
		return buildSpiralPair(ctx, doc, p, pinionParams, drivingPlan, pinionPlan)
	}
	drivingSection, err := blankSectionOf(drivingPlan.geometry)
	if err != nil {
		return GearPair{}, fmt.Errorf("driving gear blank: %w", err)
	}
	pinionSection, err := blankSectionOf(pinionPlan.geometry)
	if err != nil {
		return GearPair{}, fmt.Errorf("pinion blank: %w", err)
	}
	containers, err := newPairContainers(doc)
	if err != nil {
		return GearPair{}, err
	}
	shaft := newShaftFrame(p.ShaftAngle)
	drivingBlank, pinionBlank, err := revolveSharedBlanks(ctx, doc, containers.pair, drivingSection, pinionSection, shaft)
	if err != nil {
		return GearPair{}, err
	}
	if drivingBlank, err = moveBody(doc, drivingBlank, containers.driving); err != nil {
		return GearPair{}, err
	}
	if pinionBlank, err = moveBody(doc, pinionBlank, containers.pinion); err != nil {
		return GearPair{}, err
	}
	pinion, err := buildMember(ctx, doc, containers.pinion, shaft, pinionBlank, pinionParams, pinionPlan, true)
	if err != nil {
		return GearPair{}, fmt.Errorf("build pinion: %w", err)
	}
	driving, err := buildMember(ctx, doc, containers.driving, localShaft, drivingBlank, p, drivingPlan, false)
	if err != nil {
		return GearPair{}, fmt.Errorf("build driving gear: %w", err)
	}
	// Both members are built with a tooth centred on the angle toward the mate, and the pinion's mesh point sits half
	// a turn around its shaft from that angle. An odd pinion tooth count puts a gap there instead of a tooth, so the
	// pinion turns by one tooth width to keep the driving gear's gap opposite a pinion tooth.
	if p.MateTeeth%2 == 1 {
		pinionPhase, phaseErr := r3.Rotation(shaft.axis(), units.Radians(math.Pi/float64(p.MateTeeth)))
		if phaseErr != nil {
			return GearPair{}, fmt.Errorf("rotate pinion by one tooth: %w", phaseErr)
		}
		pinion, err = decad.Transform(ctx, pinion, pinionPhase)
		if err != nil {
			return GearPair{}, fmt.Errorf("phase pinion: %w", err)
		}
	}
	return placePair(ctx, doc, p, containers, driving, pinion)
}

// buildSpiralPair builds a spiral pair. Each member is built about +X in the root container; the pinion is then
// turned onto its shaft.
func buildSpiralPair(ctx context.Context, doc *decad.Document, p, pinionParams Params,
	drivingPlan, pinionPlan memberBuildPlan) (GearPair, error) {
	containers, err := newPairContainers(doc)
	if err != nil {
		return GearPair{}, err
	}
	pinion, err := buildMember(ctx, doc, containers.pinion, localShaft, nil, pinionParams, pinionPlan, true)
	if err != nil {
		return GearPair{}, fmt.Errorf("build pinion: %w", err)
	}
	var driving *decad.Body
	if p == pinionParams {
		// The equal spiral members share a symmetric tooth profile, blank, trims, and bore.
		// Reflecting across the shaft's XY plane reverses the spiral hand.
		plane, frameErr := r3.NewFrame(r3.Vec{}, r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
		if frameErr != nil {
			return GearPair{}, fmt.Errorf("create spiral mirror plane: %w", frameErr)
		}
		driving, err = decad.MirroredCopy(ctx, pinion, decad.MirrorFrame{Frame: plane})
		if err != nil {
			return GearPair{}, fmt.Errorf("mirror identical spiral driving gear: %w", err)
		}
	}
	if p.MateTeeth%2 == 1 {
		pinionPhase, phaseErr := r3.Rotation(r3.NewVec(1, 0, 0), units.Radians(math.Pi/float64(p.MateTeeth)))
		if phaseErr != nil {
			return GearPair{}, fmt.Errorf("rotate pinion by one tooth: %w", phaseErr)
		}
		pinion, err = decad.Transform(ctx, pinion, pinionPhase)
		if err != nil {
			return GearPair{}, fmt.Errorf("phase pinion: %w", err)
		}
	}
	pinionTurn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(p.ShaftAngle))
	if err != nil {
		return GearPair{}, fmt.Errorf("rotate pinion shaft: %w", err)
	}
	pinion, err = decad.Transform(ctx, pinion, pinionTurn)
	if err != nil {
		return GearPair{}, fmt.Errorf("place pinion: %w", err)
	}
	if driving == nil {
		driving, err = buildMember(ctx, doc, containers.driving, localShaft, nil, p, drivingPlan, false)
		if err != nil {
			return GearPair{}, fmt.Errorf("build driving gear: %w", err)
		}
	}
	return placePair(ctx, doc, p, containers, driving, pinion)
}

// placePair phases the driving gear by half a tooth pitch, moves both members with Params.Placement, and puts
// them in their containers.
func placePair(ctx context.Context, doc *decad.Document, p Params, containers pairContainers,
	driving, pinion *decad.Body) (GearPair, error) {
	drivingTurn, err := r3.Rotation(r3.NewVec(1, 0, 0), units.Radians(math.Pi/float64(p.Teeth)))
	if err != nil {
		return GearPair{}, fmt.Errorf("rotate driving gear by half a tooth pitch: %w", err)
	}
	driving, err = decad.Transform(ctx, driving, drivingTurn)
	if err != nil {
		return GearPair{}, fmt.Errorf("place driving gear: %w", err)
	}
	// Both members are built about the world origin; one rigid move per body keeps their mesh.
	apex, err := apexFrame(p)
	if err != nil {
		return GearPair{}, err
	}
	driving, err = placement.Place(ctx, driving, apex)
	if err != nil {
		return GearPair{}, fmt.Errorf("place driving gear: %w", err)
	}
	pinion, err = placement.Place(ctx, pinion, apex)
	if err != nil {
		return GearPair{}, fmt.Errorf("place pinion: %w", err)
	}
	if driving, err = moveBody(doc, driving, containers.driving); err != nil {
		return GearPair{}, err
	}
	if pinion, err = moveBody(doc, pinion, containers.pinion); err != nil {
		return GearPair{}, err
	}
	return GearPair{Driving: driving, Pinion: pinion}, nil
}
