package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// GearPair holds the driving member and pinion around their common pitch apex.
type GearPair struct {
	Driving *decad.Body
	Pinion  *decad.Body
}

// BuildPair builds the verified straight 8/8 pair in source operation order.
func BuildPair(ctx context.Context, doc *decad.Document, p Params) (GearPair, error) {
	if ctx == nil || doc == nil {
		return GearPair{}, fmt.Errorf("context and document must not be nil")
	}
	pinionParams := p
	pinionParams.Teeth, pinionParams.MateTeeth = p.MateTeeth, p.Teeth
	pinionParams.ToeRadius, pinionParams.MateToeRadius = p.MateToeRadius, p.ToeRadius
	pinionParams.BaseHeight, pinionParams.MateBaseHeight = p.MateBaseHeight, p.BaseHeight

	// Build accepts only the equal 8/8 case today. Check both members before
	// adding the pinion so an unsupported pair leaves the document empty.
	if p.Teeth != 8 || p.MateTeeth != 8 {
		return GearPair{}, fmt.Errorf("BuildPair currently supports the verified 8/8 straight case")
	}
	pinion, err := Build(ctx, doc, pinionParams)
	if err != nil {
		return GearPair{}, fmt.Errorf("build pinion: %w", err)
	}
	pinionTurn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(p.ShaftAngle))
	if err != nil {
		return GearPair{}, fmt.Errorf("rotate pinion shaft: %w", err)
	}
	pinion, err = pinion.Placed(ctx, pinionTurn)
	if err != nil {
		return GearPair{}, fmt.Errorf("place pinion: %w", err)
	}
	driving, err := Build(ctx, doc, p)
	if err != nil {
		return GearPair{}, fmt.Errorf("build driving gear: %w", err)
	}
	drivingTurn, err := r3.Rotation(r3.NewVec(1, 0, 0), units.Radians(math.Pi/float64(p.Teeth)))
	if err != nil {
		return GearPair{}, fmt.Errorf("rotate driving gear by half a tooth pitch: %w", err)
	}
	driving, err = driving.Placed(ctx, drivingTurn)
	if err != nil {
		return GearPair{}, fmt.Errorf("place driving gear: %w", err)
	}
	return GearPair{Driving: driving, Pinion: pinion}, nil
}
