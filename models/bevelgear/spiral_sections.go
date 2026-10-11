package bevelgear

import (
	"context"
	"fmt"
	"sort"

	"github.com/lestrrat-3d/decad"
)

// spiralLoftFaces selects the source's nine sections after twist and crown.
// The first face extends the toe trim; every following face is a slab heel.
func spiralLoftFaces(ctx context.Context, slabs []*decad.Body,
	frame spiralSliceFrame) ([]*decad.Face, error) {
	if ctx == nil || len(slabs) == 0 {
		return nil, fmt.Errorf("spiral loft needs crowned slabs and a context")
	}
	type section struct {
		toe, heel *decad.Face
		distance  float64
		bound     float64
	}
	sections := make([]section, len(slabs))
	for i, slab := range slabs {
		if slab == nil {
			return nil, fmt.Errorf("spiral slab %d is nil", i)
		}
		toe, heel, err := slab.ExtremeFacesByCentroid(ctx, frame.coneVec)
		if err != nil {
			return nil, fmt.Errorf("select crowned slab %d end faces: %w", i, err)
		}
		center, err := heel.Centroid()
		if err != nil {
			return nil, fmt.Errorf("measure crowned slab %d heel face: %w", i, err)
		}
		sections[i] = section{toe: toe, heel: heel,
			distance: center.Value.Dot(frame.coneVec), bound: center.Bound.Base()}
	}
	sort.Slice(sections, func(i, j int) bool { return sections[i].distance < sections[j].distance })
	for i := 1; i < len(sections); i++ {
		if sections[i-1].distance+sections[i-1].bound >= sections[i].distance-sections[i].bound {
			return nil, fmt.Errorf("spiral loft section %d is not separated from its predecessor", i)
		}
	}
	faces := make([]*decad.Face, 0, len(sections)+1)
	faces = append(faces, sections[0].toe)
	for _, slab := range sections {
		faces = append(faces, slab.heel)
	}
	return faces, nil
}
