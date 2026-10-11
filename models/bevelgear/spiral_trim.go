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
	if ctx == nil || doc == nil || curved == nil {
		return nil, fmt.Errorf("spiral trim needs a context, document and curved tooth")
	}
	if err := validateSourceDefaultSpiral(p); err != nil {
		return nil, err
	}
	geometry := p
	geometry.SpiralAngle = 0
	geometry.BoreDiameter = 0
	toeCone, err := BuildToeCone(ctx, doc, geometry)
	if err != nil {
		return nil, fmt.Errorf("build spiral toe cone: %w", err)
	}
	toeKeeper, err := decad.Cut(ctx, curved, toeCone)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth at toe: %w", err)
	}
	heelCone, err := BuildHeelCone(ctx, doc, geometry)
	if err != nil {
		return nil, fmt.Errorf("build spiral heel cone: %w", err)
	}
	trimmed, err := decad.Intersect(ctx, toeKeeper, heelCone)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth at heel: %w", err)
	}
	return trimmed, nil
}
