package bevelgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// trimSpiralTooth cuts the curved tooth flush with the same two conical
// surfaces that bound the source's straight tooth.
func trimSpiralTooth(ctx context.Context, doc *decad.Document,
	p Params, curved *decad.Body) (*decad.Body, error) {
	return trimSpiralToothBy(ctx, doc, p, curved, 0)
}

// trimSpiralToothBy cuts the curved tooth with the toe cone shrunk and the heel cone grown by
// overshoot millimeters along the shaft, so the tooth ends just outside the blank's toe and heel caps.
func trimSpiralToothBy(ctx context.Context, doc *decad.Document,
	p Params, curved *decad.Body, overshoot float64) (*decad.Body, error) {
	if ctx == nil || doc == nil || curved == nil {
		return nil, fmt.Errorf("spiral trim needs a context, document and curved tooth")
	}
	if err := validateSpiralBuild(p); err != nil {
		return nil, err
	}
	geometry := p
	geometry.SpiralAngle = 0
	geometry.BoreDiameter = 0
	toeCone, err := buildShiftedEndCone(ctx, doc, geometry, false, -overshoot)
	if err != nil {
		return nil, fmt.Errorf("build spiral toe cone: %w", err)
	}
	toeKeeper, err := decad.Cut(ctx, curved, toeCone)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth at toe: %w", err)
	}
	heelCone, err := buildShiftedEndCone(ctx, doc, geometry, true, overshoot)
	if err != nil {
		return nil, fmt.Errorf("build spiral heel cone: %w", err)
	}
	trimmed, err := decad.Intersect(ctx, toeKeeper, heelCone)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth at heel: %w", err)
	}
	return trimmed, nil
}
