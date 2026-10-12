package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// twistSpiralSlabs applies the source's heel-face-keyed shaft rotation.
// The tooth is still in its local frame, with the apex at the origin and shaft on X.
func twistSpiralSlabs(ctx context.Context, slabs []*decad.Body,
	frame spiralSliceFrame, trace spiralTrace) ([]*decad.Body, error) {
	if ctx == nil || len(slabs) == 0 || frame.span <= 0 ||
		trace.total <= 0 || math.Abs(trace.handSign) != 1 {
		return nil, fmt.Errorf("spiral twist needs slabs, a positive span and a signed trace")
	}
	twisted := make([]*decad.Body, 0, len(slabs))
	shaft := r3.NewVec(1, 0, 0)
	for i, slab := range slabs {
		if slab == nil {
			return nil, fmt.Errorf("spiral slab %d is nil", i)
		}
		_, heelFace, err := slab.ExtremeFacesByCentroid(ctx, frame.coneVec)
		if err != nil {
			return nil, fmt.Errorf("select spiral slab %d heel face: %w", i, err)
		}
		center, err := heelFace.Centroid()
		if err != nil {
			return nil, fmt.Errorf("measure spiral slab %d heel face: %w", i, err)
		}
		coneDistance := center.Value.Dot(frame.coneVec)
		angle := -trace.handSign * trace.total * (frame.mean - coneDistance) / frame.span
		if math.IsNaN(angle) || math.IsInf(angle, 0) {
			return nil, fmt.Errorf("spiral slab %d twist is not finite", i)
		}
		turn, err := r3.RotationAround(r3.Vec{}, shaft, units.Radians(angle))
		if err != nil {
			return nil, fmt.Errorf("rotate spiral slab %d: %w", i, err)
		}
		placed, err := decad.Transform(ctx, slab, turn)
		if err != nil {
			return nil, fmt.Errorf("place spiral slab %d: %w", i, err)
		}
		twisted = append(twisted, placed)
	}
	return twisted, nil
}
