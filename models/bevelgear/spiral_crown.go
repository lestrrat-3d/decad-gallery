package bevelgear

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

const spiralCrownPerRad = 0.5

type spiralHeelSection struct {
	face     *decad.Face
	distance float64
	bound    float64
}

// crownSpiralSlabs scales every twisted slab except the heel-most one.
// Each scale is anchored to the root edge of that slab's post-twist heel face.
func crownSpiralSlabs(ctx context.Context, slabs []*decad.Body,
	frame spiralSliceFrame, trace spiralTrace) ([]*decad.Body, error) {
	if ctx == nil || len(slabs) == 0 || frame.span <= 0 || trace.total <= 0 {
		return nil, fmt.Errorf("spiral crown needs twisted slabs and a positive trace")
	}
	heels := make([]spiralHeelSection, len(slabs))
	heelMost := -1
	for i, slab := range slabs {
		if slab == nil {
			return nil, fmt.Errorf("spiral slab %d is nil", i)
		}
		_, heel, err := slab.ExtremeFacesByCentroid(ctx, frame.coneVec)
		if err != nil {
			return nil, fmt.Errorf("select post-twist heel face of slab %d: %w", i, err)
		}
		center, err := heel.Centroid()
		if err != nil {
			return nil, fmt.Errorf("measure post-twist heel face of slab %d: %w", i, err)
		}
		heels[i] = spiralHeelSection{
			face: heel, distance: center.Value.Dot(frame.coneVec), bound: center.Bound.Base(),
		}
		if heelMost < 0 || heels[i].distance > heels[heelMost].distance {
			heelMost = i
		}
	}
	for i, heel := range heels {
		if i != heelMost && heels[heelMost].distance-heels[heelMost].bound <= heel.distance+heel.bound {
			return nil, fmt.Errorf("spiral heel-most slab is not separated from slab %d by centroid bounds", i)
		}
	}
	crowned := make([]*decad.Body, len(slabs))
	for i, slab := range slabs {
		if i == heelMost {
			crowned[i] = slab
			continue
		}
		u := (frame.heel - heels[i].distance) / frame.span
		factor := 1 - spiralCrownPerRad*(math.Abs(trace.total)/2)*u
		if math.IsNaN(factor) || math.IsInf(factor, 0) || factor <= 0 {
			return nil, fmt.Errorf("spiral slab %d crown factor %.9g is invalid at u=%.9g", i, factor, u)
		}
		anchor, err := spiralHeelRootAnchor(heels[i].face)
		if err != nil {
			return nil, fmt.Errorf("root anchor of spiral slab %d: %w", i, err)
		}
		scaled, err := slab.ScaledAt(ctx, anchor, units.Scalar(factor))
		if err != nil {
			return nil, fmt.Errorf("crown spiral slab %d at u=%.9g, factor=%.9g: %w", i, u, factor, err)
		}
		crowned[i] = scaled
	}
	return crowned, nil
}

func spiralHeelRootAnchor(heel *decad.Face) (r3.Vec, error) {
	if heel == nil {
		return r3.Vec{}, fmt.Errorf("heel face is nil")
	}
	type radialVertex struct {
		point  r3.Vec
		radius float64
		bound  float64
	}
	seen := make(map[*decad.Vertex]bool)
	vertices := make([]radialVertex, 0)
	for _, edge := range heel.Edges() {
		for _, vertex := range []*decad.Vertex{edge.Start(), edge.End()} {
			if vertex == nil || seen[vertex] {
				continue
			}
			seen[vertex] = true
			reading := vertex.Position()
			radius := math.Hypot(reading.Value.Y, reading.Value.Z)
			bound := reading.Bound.Base()
			if math.IsNaN(radius) || math.IsInf(radius, 0) ||
				math.IsNaN(bound) || math.IsInf(bound, 0) || bound < 0 {
				return r3.Vec{}, fmt.Errorf("heel vertex has no finite shaft-axis distance bound")
			}
			vertices = append(vertices, radialVertex{point: reading.Value, radius: radius, bound: bound})
		}
	}
	if len(vertices) < 3 {
		return r3.Vec{}, fmt.Errorf("heel face has %d vertices; need three to distinguish its root edge", len(vertices))
	}
	sort.Slice(vertices, func(i, j int) bool { return vertices[i].radius < vertices[j].radius })
	rootUpper := math.Max(vertices[0].radius+vertices[0].bound,
		vertices[1].radius+vertices[1].bound)
	otherLower := math.Inf(1)
	for _, vertex := range vertices[2:] {
		otherLower = math.Min(otherLower, vertex.radius-vertex.bound)
	}
	if rootUpper >= otherLower {
		return r3.Vec{}, fmt.Errorf("the two root corners overlap another heel vertex within position bounds")
	}
	return vertices[0].point.Add(vertices[1].point).Scale(0.5), nil
}
