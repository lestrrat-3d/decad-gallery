package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestRealEightToothBlankOneToothJoin(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	doc := decad.New()
	blank, err := bevelgear.BuildBlank(t.Context(), doc, p)
	require.NoError(t, err)
	tooth, err := bevelgear.BuildPointTooth(t.Context(), doc, p)
	require.NoError(t, err)
	toeCone, err := bevelgear.BuildToeCone(t.Context(), doc, p)
	require.NoError(t, err)
	toeKeeper, err := decad.Cut(t.Context(), tooth, toeCone)
	require.NoError(t, err)
	heelCone, err := bevelgear.BuildHeelCone(t.Context(), doc, p)
	require.NoError(t, err)
	finalTooth, err := decad.Intersect(t.Context(), toeKeeper, heelCone)
	require.NoError(t, err)
	requireAnalyticTrim(t, finalTooth)

	// Confirm that the trimmed tooth seats inside the piecewise revolved blank.
	toothMesh, err := finalTooth.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	dedStation := 4 + 1.25/math.Sqrt2
	dedRadius := 4 - 1.25/math.Sqrt2
	rootSlope := dedRadius / dedStation
	maxRadialMargin := math.Inf(-1)
	interiorTriangles := 0
	vertices := toothMesh.Vertices()
	for _, triangle := range toothMesh.Triangles() {
		centroid := vertices[triangle[0]].Add(vertices[triangle[1]]).Add(vertices[triangle[2]]).Scale(1.0 / 3)
		if centroid.X <= 4.05 || centroid.X >= 4.95 {
			continue
		}
		outerRadius := rootSlope * centroid.X
		if centroid.X > dedStation {
			outerRadius = dedRadius + (3-dedRadius)*(centroid.X-dedStation)/(5-dedStation)
		}
		margin := outerRadius - math.Hypot(centroid.Y, centroid.Z)
		if margin > 0.01 {
			interiorTriangles++
		}
		maxRadialMargin = math.Max(maxRadialMargin, margin)
	}
	require.Positive(t, interiorTriangles)
	require.Greater(t, maxRadialMargin, 0.05)

	blankVolume, err := blank.Volume()
	require.NoError(t, err)
	joined, err := decad.Union(t.Context(), blank, finalTooth)
	require.NoError(t, err)
	require.True(t, joined.IsSolid())
	joinedVolume, err := joined.Volume()
	require.NoError(t, err)
	require.Greater(t, joinedVolume.Value.Base()-joinedVolume.Bound.Base(),
		blankVolume.Value.Base()+blankVolume.Bound.Base())
	mesh, err := joined.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}
