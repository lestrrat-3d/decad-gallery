package bevelgear

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
)

// buildSpiralMember builds the blank and the curved tooth, then joins them. The 31/31 members the
// source-order Combine-Join builds keep that route; every other member takes the serial route below.
// The bore is cut by buildMember afterward.
func buildSpiralMember(ctx context.Context, doc *decad.Document,
	p, geometry Params, pinion bool) (*decad.Body, error) {
	if validateSourceDefaultSpiral(p) != nil {
		return buildSpiralMemberSerial(ctx, doc, p, geometry, pinion)
	}
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

// spiralOvershoots lists, in modules, how far the serial route's teeth end past the blank's toe and heel
// caps before the member is trimmed flush. A boolean can refuse a result whose float rounding folds two
// facets through each other, or find two held facets too close to tell. That depends on the exact geometry,
// so a member that decad refuses with one overshoot is built again with the next, and with the other join
// order.
var spiralOvershoots = []float64{0.5, 0.37, 0.81, 0.62, 0.45}

// joinOrder says how the teeth reach the extended blank: one at a time, or first with each other.
type joinOrder int

const (
	joinOneByOne joinOrder = iota
	joinTeethFirst
)

// buildSpiralMemberSerial builds a member with the first overshoot in spiralOvershoots that the final
// intersection accepts. It leaves the document holding only the member.
func buildSpiralMemberSerial(ctx context.Context, doc *decad.Document,
	p, geometry Params, pinion bool) (*decad.Body, error) {
	body, _, err := buildSpiralMemberSerialTrace(ctx, doc, p, geometry, pinion)
	return body, err
}

// buildSpiralMemberSerialTrace is buildSpiralMemberSerial that also returns the overshoot it used, in modules.
func buildSpiralMemberSerialTrace(ctx context.Context, doc *decad.Document,
	p, geometry Params, pinion bool) (*decad.Body, float64, error) {
	var last error
	for _, modules := range spiralOvershoots {
		for _, order := range []joinOrder{joinOneByOne, joinTeethFirst} {
			before := map[*decad.Body]struct{}{}
			for _, body := range doc.Bodies() {
				before[body] = struct{}{}
			}
			body, err := buildSpiralMemberOvershoot(ctx, doc, p, geometry, pinion, modules*p.Module, order)
			if err == nil {
				return body, modules, nil
			}
			last = err
			for _, body := range doc.Bodies() {
				if _, ok := before[body]; !ok {
					_ = decad.Remove(body)
				}
			}
			if !errors.Is(err, decad.ErrUnsupported) {
				return nil, 0, last
			}
		}
	}
	return nil, 0, last
}

// buildSpiralMemberOvershoot extends the blank and trims each tooth past the final toe and heel caps, so no
// tooth face lies on a blank face, joins the teeth one at a time, then intersects the member once with the
// region between the end planes and the toe and heel cones, whose faces become the member's end faces. The
// unions and the intersection are generic booleans whose contacts are all transversal; their chained
// held-mesh bound needs a document made with decad.WithChainTolerance.
func buildSpiralMemberOvershoot(ctx context.Context, doc *decad.Document,
	p, geometry Params, pinion bool, overshoot float64, order joinOrder) (*decad.Body, error) {
	ends, err := buildSerialEnds(ctx, doc, geometry, p.Module)
	if err != nil {
		return nil, fmt.Errorf("build extended spiral blank and cap region: %w", err)
	}
	curved, err := buildSpiralTooth(ctx, doc, p, pinion)
	if err != nil {
		return nil, err
	}
	tooth, err := trimSpiralToothBy(ctx, doc, p, curved, overshoot)
	if err != nil {
		return nil, fmt.Errorf("trim spiral tooth past the caps: %w", err)
	}
	pattern, err := patternSpiralTooth(ctx, tooth, p.Teeth)
	if err != nil {
		return nil, err
	}
	joined := ends.blank
	if order == joinTeethFirst {
		// The teeth are disjoint, so joining them first is trivial and each keeps its own bound.
		combined := pattern[0]
		for i, instance := range pattern[1:] {
			combined, err = decad.Union(ctx, combined, instance)
			if err != nil {
				return nil, fmt.Errorf("join spiral teeth 1 and %d: %w", i+2, err)
			}
		}
		joined, err = decad.Union(ctx, joined, combined)
		if err != nil {
			return nil, fmt.Errorf("join spiral teeth to the extended blank: %w", err)
		}
	} else {
		for i, instance := range pattern {
			joined, err = decad.Union(ctx, joined, instance)
			if err != nil {
				return nil, fmt.Errorf("join spiral tooth %d: %w", i+1, err)
			}
		}
	}
	trimmed, err := decad.Intersect(ctx, joined, ends.region)
	if err != nil {
		return nil, fmt.Errorf("trim spiral member to its caps: %w", err)
	}
	return trimmed, nil
}

// buildSpiralTooth lofts the source-default curved tooth before conical trims.
// The pinion uses the opposite hand from the driving member.
func buildSpiralTooth(ctx context.Context, doc *decad.Document,
	p Params, pinion bool) (*decad.Body, error) {
	if ctx == nil || doc == nil || p.SpiralAngle <= 0 {
		return nil, fmt.Errorf("spiral tooth needs a document, context and positive spiral angle")
	}
	if err := validateSpiralBuild(p); err != nil {
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
	curved, err := decad.LoftFaces(ctx, doc, faces...)
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
