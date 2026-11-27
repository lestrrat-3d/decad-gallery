package cubepuzzle

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestCornerExtractionWitness(t *testing.T) {
	ctx, doc := t.Context(), NewDocument()
	p := r3.Vec{X: 13, Y: 13, Z: 7.5}
	q := p.Sub(r3.Vec{X: 3, Y: 3, Z: 3})
	for _, axis := range []r3.Vec{{X: 1}, {Y: 1}, {Z: 1}} {
		body, _, err := makeLayer(ctx, doc, axis, .25, 30)
		require.NoError(t, err)
		if axis.Z == 1 {
			requireWitnessPath(t, "corner Z layer", body, r3.Vec{X: 10, Y: 10, Z: 20}, q, decad.PointInside, 80)
		} else {
			requireWitnessLocation(t, "corner layer", body, q, decad.PointInside)
			requireWitnessLocation(t, "edge layer", body, p, decad.PointInside)
		}
	}
	for _, axis := range []r3.Vec{{Z: 1}, {Z: -1}} {
		cut, _, err := makeLayer(ctx, doc, axis, -.25, 60)
		require.NoError(t, err)
		requireWitnessPath(t, "edge layer cutter", cut, r3.Vec{X: 13, Y: 13}, p, decad.PointOutside, 45)
	}
	sphere, err := makeSphere(ctx, doc, 12)
	require.NoError(t, err)
	requireWitnessLocation(t, "edge sphere cutter", sphere, p, decad.PointOutside)
	requireWitnessLocation(t, "corner sphere cutter", sphere, q, decad.PointOutside)
}

func TestEdgeExtractionWitness(t *testing.T) {
	ctx, doc := t.Context(), NewDocument()
	point := r3.Vec{X: 8.5, Z: 18.3}
	original := r3.Vec{X: 4.35, Y: 14.15}
	center, _, err := makeLayer(ctx, doc, r3.Vec{Z: 1}, .25, 30)
	require.NoError(t, err)
	requireWitnessLocation(t, "center source", center, point, decad.PointInside)
	for _, axis := range []r3.Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		cut, _, err := makeLayer(ctx, doc, axis, -.25, 60)
		require.NoError(t, err)
		if axis.X == 1 {
			requireWitnessPath(t, "center +X cutter", cut, r3.Vec{Z: 18.3}, point, decad.PointOutside, 850)
		} else {
			requireWitnessLocation(t, "center cutter", cut, point, decad.PointOutside)
		}
	}
	for _, cut := range []struct{ z, radius, length float64 }{{-40, 45, 56}, {15, 2.9, 16}, {25.6, 6.4, 5.4}} {
		frame, err := r3.NewFrame(r3.Vec{Z: cut.z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		require.NoError(t, err)
		tool, _, err := fastenertool.Cylinder(ctx, doc, frame, cut.radius, cut.length)
		require.NoError(t, err)
		requireWitnessLocation(t, "center mounting cutter", tool, point, decad.PointOutside)
	}
	for _, axis := range []r3.Vec{{X: 1}, {Y: 1}} {
		body, _, err := makeLayer(ctx, doc, axis, .25, 30)
		require.NoError(t, err)
		if axis.X == 1 {
			requireWitnessPath(t, "edge X layer", body, r3.Vec{X: 20, Y: 14.15}, original, decad.PointInside, 783)
		} else {
			requireWitnessLocation(t, "edge Y layer", body, original, decad.PointInside)
		}
	}
	for _, axis := range []r3.Vec{{Z: 1}, {Z: -1}} {
		cut, _, err := makeLayer(ctx, doc, axis, -.25, 60)
		require.NoError(t, err)
		requireWitnessLocation(t, "edge cutter", cut, original, decad.PointOutside)
	}
	sphere, err := makeSphere(ctx, doc, 12)
	require.NoError(t, err)
	requireWitnessLocation(t, "center sphere cutter", sphere, point, decad.PointOutside)
	requireWitnessLocation(t, "edge sphere cutter", sphere, original, decad.PointOutside)
}
