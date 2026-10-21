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

// BuildPair builds the pinion first, then phases the pinion by half a tooth pitch (one tooth width).
// It then moves both members with Params.Placement.
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
	pinion, err := buildMember(ctx, doc, pinionParams, pinionPlan, true)
	if err != nil {
		return GearPair{}, fmt.Errorf("build pinion: %w", err)
	}
	var driving *decad.Body
	if p == pinionParams {
		if drivingPlan.spiral {
			// The equal members share a symmetric tooth profile, blank, trims, and bore.
			// Reflecting across the shaft's XY plane reverses the spiral hand.
			plane, frameErr := r3.NewFrame(r3.Vec{}, r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
			if frameErr != nil {
				return GearPair{}, fmt.Errorf("create spiral mirror plane: %w", frameErr)
			}
			driving, err = decad.MirroredCopy(ctx, pinion, decad.MirrorFrame{Frame: plane})
			if err != nil {
				return GearPair{}, fmt.Errorf("mirror identical spiral driving gear: %w", err)
			}
		} else {
			driving, err = decad.Copy(ctx, pinion)
			if err != nil {
				return GearPair{}, fmt.Errorf("copy identical straight driving gear: %w", err)
			}
		}
	}
	// Turning the pinion by one tooth width about its shaft puts a tooth opposite a gap of the driving gear.
	pinionPhase, err := r3.Rotation(r3.NewVec(1, 0, 0), units.Radians(math.Pi/float64(p.MateTeeth)))
	if err != nil {
		return GearPair{}, fmt.Errorf("rotate pinion by one tooth: %w", err)
	}
	pinion, err = decad.Transform(ctx, pinion, pinionPhase)
	if err != nil {
		return GearPair{}, fmt.Errorf("phase pinion: %w", err)
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
		driving, err = buildMember(ctx, doc, p, drivingPlan, false)
		if err != nil {
			return GearPair{}, fmt.Errorf("build driving gear: %w", err)
		}
	}
	// Both members are built about the world origin; one rigid move per body keeps their mesh.
	driving, err = placement.Place(ctx, driving, p.Placement)
	if err != nil {
		return GearPair{}, fmt.Errorf("place driving gear: %w", err)
	}
	pinion, err = placement.Place(ctx, pinion, p.Placement)
	if err != nil {
		return GearPair{}, fmt.Errorf("place pinion: %w", err)
	}
	return GearPair{Driving: driving, Pinion: pinion}, nil
}
