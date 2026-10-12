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
	return joinSpiralMember(ctx, doc, p, blank, curved)
}

// buildMatchedSpiralMembers reuses the identical blank and untwisted slices.
// The handed twist, crown, trim, pattern, join, and bore still run per member.
func buildMatchedSpiralMembers(ctx context.Context, doc *decad.Document,
	p Params, plan memberBuildPlan) (*decad.Body, *decad.Body, error) {
	blank, err := BuildBlank(ctx, doc, plan.geometry)
	if err != nil {
		return nil, nil, fmt.Errorf("build reference blank: %w", err)
	}
	drivingBlank, err := blank.Duplicate(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("copy identical spiral blank: %w", err)
	}
	slabs, frame, err := buildSpiralSlabs(ctx, doc, p)
	if err != nil {
		return nil, nil, err
	}
	drivingSlabs := make([]*decad.Body, len(slabs))
	for i, slab := range slabs {
		drivingSlabs[i], err = slab.Duplicate(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("copy spiral slice %d: %w", i, err)
		}
	}
	pinionTooth, err := buildSpiralToothFromSlabs(ctx, doc, p, true, slabs, frame)
	if err != nil {
		return nil, nil, fmt.Errorf("build pinion tooth: %w", err)
	}
	pinion, err := joinSpiralMember(ctx, doc, p, blank, pinionTooth)
	if err != nil {
		return nil, nil, fmt.Errorf("join pinion: %w", err)
	}
	if plan.boreDiameter > 0 {
		pinion, err = cutBore(ctx, doc, pinion, plan.boreDiameter)
		if err != nil {
			return nil, nil, fmt.Errorf("cut pinion bore: %w", err)
		}
	}
	drivingTooth, err := buildSpiralToothFromSlabs(ctx, doc, p, false, drivingSlabs, frame)
	if err != nil {
		return nil, nil, fmt.Errorf("build driving tooth: %w", err)
	}
	driving, err := joinSpiralMember(ctx, doc, p, drivingBlank, drivingTooth)
	if err != nil {
		return nil, nil, fmt.Errorf("join driving gear: %w", err)
	}
	if plan.boreDiameter > 0 {
		driving, err = cutBore(ctx, doc, driving, plan.boreDiameter)
		if err != nil {
			return nil, nil, fmt.Errorf("cut driving bore: %w", err)
		}
	}
	return pinion, driving, nil
}

func joinSpiralMember(ctx context.Context, doc *decad.Document,
	p Params, blank, curved *decad.Body) (*decad.Body, error) {
	trimmed, err := trimSpiralTooth(ctx, doc, p, curved)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth: %w", err)
	}
	pattern, err := patternSpiralTooth(ctx, trimmed, p.Teeth)
	if err != nil {
		return nil, err
	}
	joined, err := decad.CombineJoin(ctx, blank, pattern...)
	if err != nil {
		return nil, fmt.Errorf("join spiral tooth pattern: %w", err)
	}
	return joined, nil
}

// buildSpiralTooth lofts the source-default curved tooth before conical trims.
// The pinion uses the opposite hand from the driving member.
func buildSpiralTooth(ctx context.Context, doc *decad.Document,
	p Params, pinion bool) (*decad.Body, error) {
	slabs, frame, err := buildSpiralSlabs(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return buildSpiralToothFromSlabs(ctx, doc, p, pinion, slabs, frame)
}

func buildSpiralSlabs(ctx context.Context, doc *decad.Document,
	p Params) ([]*decad.Body, spiralSliceFrame, error) {
	if ctx == nil || doc == nil || p.SpiralAngle <= 0 {
		return nil, spiralSliceFrame{}, fmt.Errorf("spiral tooth needs a document, context and positive spiral angle")
	}
	if err := validateSourceDefaultSpiral(p); err != nil {
		return nil, spiralSliceFrame{}, err
	}
	straight := p
	straight.SpiralAngle = 0
	straight.BoreDiameter = 0
	tooth, err := BuildPointTooth(ctx, doc, straight)
	if err != nil {
		return nil, spiralSliceFrame{}, fmt.Errorf("build uncut point tooth: %w", err)
	}
	slabs, frame, err := sliceSpiralPointTooth(ctx, doc, p, tooth)
	if err != nil {
		return nil, spiralSliceFrame{}, fmt.Errorf("slice point tooth: %w", err)
	}
	return slabs, frame, nil
}

func buildSpiralToothFromSlabs(ctx context.Context, doc *decad.Document,
	p Params, pinion bool, slabs []*decad.Body, frame spiralSliceFrame) (*decad.Body, error) {
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
		if err := doc.Remove(slab); err != nil {
			return nil, fmt.Errorf("remove spiral slab %d after loft: %w", i, err)
		}
	}
	return curved, nil
}
