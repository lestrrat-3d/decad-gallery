package bevelgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// buildSpiralMember follows the source's blank, tooth, trim, pattern, and
// single Combine-Join order. The bore is cut by buildMember afterward.
func buildSpiralMember(ctx context.Context, doc *decad.Document,
	p, geometry Params, pinion bool) (*decad.Body, error) {
	blank, err := BuildBlank(ctx, doc, geometry)
	if err != nil {
		return nil, fmt.Errorf("build spiral blank: %w", err)
	}
	curved, err := buildSpiralTooth(ctx, doc, p, pinion)
	if err != nil {
		return nil, err
	}
	trimmed, err := trimSpiralTooth(ctx, doc, p, curved)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth: %w", err)
	}
	pattern, err := patternSpiralTooth(ctx, trimmed, p.Teeth)
	if err != nil {
		return nil, err
	}
	joined, err := decad.Union(ctx, blank, pattern...)
	if err != nil {
		return nil, fmt.Errorf("join spiral tooth pattern: %w", err)
	}
	return joined, nil
}

// buildSpiralTooth lofts the source-default curved tooth before conical trims.
// The pinion uses the opposite hand from the driving member.
func buildSpiralTooth(ctx context.Context, doc *decad.Document,
	p Params, pinion bool) (*decad.Body, error) {
	if ctx == nil || doc == nil || p.SpiralAngle <= 0 {
		return nil, fmt.Errorf("spiral tooth needs a document, context and positive spiral angle")
	}
	if err := validateSourceDefaultSpiral(p); err != nil {
		return nil, err
	}
	straight := p
	straight.SpiralAngle = 0
	straight.BoreDiameter = 0
	tooth, err := BuildPointTooth(ctx, doc, straight)
	if err != nil {
		return nil, fmt.Errorf("build uncut point tooth: %w", err)
	}
	slabs, frame, err := sliceSpiralPointTooth(ctx, doc, p, tooth)
	if err != nil {
		return nil, fmt.Errorf("slice point tooth: %w", err)
	}
	trace, err := buildSpiralTrace(frame, p, pinion)
	if err != nil {
		return nil, fmt.Errorf("build spiral trace: %w", err)
	}
	twisted, err := twistSpiralSlabs(ctx, slabs, frame, trace)
	if err != nil {
		return nil, fmt.Errorf("twist spiral slabs: %w", err)
	}
	crowned, err := crownSpiralSlabs(ctx, twisted, frame, trace)
	if err != nil {
		return nil, fmt.Errorf("crown spiral slabs: %w", err)
	}
	faces, err := spiralLoftFaces(ctx, crowned, frame)
	if err != nil {
		return nil, fmt.Errorf("select spiral loft sections: %w", err)
	}
	curved, err := doc.LoftFaces(ctx, faces...)
	if err != nil {
		return nil, fmt.Errorf("loft spiral tooth: %w", err)
	}
	for i, slab := range crowned {
		if err := decad.Remove(slab); err != nil {
			return nil, fmt.Errorf("remove spiral slab %d after loft: %w", i, err)
		}
	}
	return curved, nil
}
