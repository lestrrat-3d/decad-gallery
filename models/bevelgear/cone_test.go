package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestEightToothToeCuttingCone(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	doc := decad.New()
	cone, err := bevelgear.BuildToeCone(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, cone.IsSolid())
	require.Len(t, doc.Bodies(), 1)

	coneDistance := 4 * math.Sqrt2
	toeFactor := 1 - math.Hypot(8, 8)/(6*coneDistance)
	dedStation := 4 + 1.25/math.Sqrt2
	dedRadius := 4 - 1.25/math.Sqrt2
	wantApex := toeFactor * (dedStation + dedRadius)
	wantFar := -0.1 * coneDistance
	wantFarRadius := wantApex - wantFar
	bounds, err := cone.Bounds()
	require.NoError(t, err)
	require.InDelta(t, wantFar, bounds.Min.X, bounds.Bound.Base()+1e-8)
	require.InDelta(t, wantApex, bounds.Max.X, bounds.Bound.Base()+1e-8)

	var coneCount, capCount int
	for _, face := range cone.Faces() {
		switch surface := face.Surface().(type) {
		case decad.Cone:
			coneCount++
			require.InDelta(t, math.Pi/4, surface.HalfAngle.Base(), 1e-8)
		case decad.Plane:
			capCount++
			area, err := face.Area()
			require.NoError(t, err)
			require.InDelta(t, math.Pi*wantFarRadius*wantFarRadius,
				area.Value.Base(), area.Bound.Base()+1e-7)
		default:
			t.Fatalf("unexpected cutting-cone surface %T", surface)
		}
	}
	require.Equal(t, 1, coneCount)
	require.Equal(t, 1, capCount)
	volume, err := cone.Volume()
	require.NoError(t, err)
	wantVolume := math.Pi * wantFarRadius * wantFarRadius * (wantApex - wantFar) / 3
	require.InDelta(t, wantVolume, volume.Value.Base(), volume.Bound.Base()+1e-6)
	mesh, err := cone.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestToeCuttingConeFollowsExtension(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth = 8, 8
	p.ToeRadius, p.MateToeRadius = 1, 1
	plain, err := bevelgear.BuildToeCone(t.Context(), decad.New(), p)
	require.NoError(t, err)
	plainBounds, err := plain.Bounds()
	require.NoError(t, err)
	p.ToeExtension = 50
	extended, err := bevelgear.BuildToeCone(t.Context(), decad.New(), p)
	require.NoError(t, err)
	extendedBounds, err := extended.Bounds()
	require.NoError(t, err)
	require.Less(t, extendedBounds.Max.X+extendedBounds.Bound.Base(),
		plainBounds.Max.X-plainBounds.Bound.Base())
	require.InDelta(t, plainBounds.Min.X, extendedBounds.Min.X,
		plainBounds.Bound.Base()+extendedBounds.Bound.Base()+1e-8)
	mesh, err := extended.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestEightToothHeelCuttingCone(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	doc := decad.New()
	cone, err := bevelgear.BuildHeelCone(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, cone.IsSolid())
	require.Len(t, doc.Bodies(), 1)

	coneDistance := 4 * math.Sqrt2
	wantApex := coneDistance / math.Cos(math.Pi/4)
	wantFar := -0.1 * coneDistance
	wantFarRadius := wantApex - wantFar
	bounds, err := cone.Bounds()
	require.NoError(t, err)
	require.InDelta(t, wantFar, bounds.Min.X, bounds.Bound.Base()+1e-8)
	require.InDelta(t, wantApex, bounds.Max.X, bounds.Bound.Base()+1e-8)
	var coneCount, capCount int
	for _, face := range cone.Faces() {
		switch surface := face.Surface().(type) {
		case decad.Cone:
			coneCount++
			require.InDelta(t, math.Pi/4, surface.HalfAngle.Base(), 1e-8)
		case decad.Plane:
			capCount++
			area, err := face.Area()
			require.NoError(t, err)
			require.InDelta(t, math.Pi*wantFarRadius*wantFarRadius,
				area.Value.Base(), area.Bound.Base()+1e-7)
		default:
			t.Fatalf("unexpected cutting-cone surface %T", surface)
		}
	}
	require.Equal(t, 1, coneCount)
	require.Equal(t, 1, capCount)
	mesh, err := cone.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestSpacedSixteenTwelveHeelConeCrossesToeTooth(t *testing.T) {
	p := spiralDefaults()
	p.Teeth, p.MateTeeth, p.Samples = 16, 12, 5
	p.ToothSpacing = 0.5
	doc := decad.New()
	tooth, err := bevelgear.BuildPointTooth(t.Context(), doc, p)
	require.NoError(t, err)
	toeCone, err := bevelgear.BuildToeCone(t.Context(), doc, p)
	require.NoError(t, err)
	toeKeeper, err := decad.Cut(t.Context(), tooth, toeCone)
	require.NoError(t, err)
	heelCone, err := bevelgear.BuildHeelCone(t.Context(), doc, p)
	require.NoError(t, err)
	heelBounds, err := heelCone.Bounds()
	require.NoError(t, err)
	var halfAngle float64
	for _, face := range heelCone.Faces() {
		if surface, ok := face.Surface().(decad.Cone); ok {
			halfAngle = surface.HalfAngle.Base()
		}
	}
	require.InDelta(t, 0.75, math.Tan(halfAngle), 1e-8)
	mesh, err := toeKeeper.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	minScore, maxScore := math.Inf(1), math.Inf(-1)
	for _, vertex := range mesh.Vertices() {
		score := vertex.X + math.Hypot(vertex.Y, vertex.Z)/math.Tan(halfAngle)
		minScore = math.Min(minScore, score)
		maxScore = math.Max(maxScore, score)
	}
	require.Less(t, minScore, heelBounds.Max.X-1)
	require.Greater(t, maxScore, heelBounds.Max.X+0.02)
}
