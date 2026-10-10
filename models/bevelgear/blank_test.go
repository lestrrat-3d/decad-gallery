package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestEightToothReferenceBlank(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	doc := decad.New()
	body, err := bevelgear.BuildBlank(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	require.Len(t, body.Faces(), 5)

	coneDist := 4 * math.Sqrt2
	width := math.Hypot(8, 8) / 6
	wantToeRadius := 4 - width/math.Sin(math.Pi/4)
	wantHeelRadius := 4 - 1/math.Tan(math.Pi/4)
	var planeAreas, coneAngles []float64
	for _, face := range body.Faces() {
		switch surface := face.Surface().(type) {
		case decad.Plane:
			area, err := face.Area()
			require.NoError(t, err)
			planeAreas = append(planeAreas, area.Value.Base())
		case decad.Cone:
			coneAngles = append(coneAngles, surface.HalfAngle.Base())
		default:
			t.Fatalf("unexpected blank surface %T", surface)
		}
	}
	require.Len(t, planeAreas, 2)
	require.Len(t, coneAngles, 3)
	require.InDelta(t, math.Pi*wantToeRadius*wantToeRadius,
		math.Min(planeAreas[0], planeAreas[1]), 1e-7)
	require.InDelta(t, math.Pi*wantHeelRadius*wantHeelRadius,
		math.Max(planeAreas[0], planeAreas[1]), 1e-7)
	wantRootAngle := math.Pi/4 - math.Atan2(1.25, coneDist)
	require.InDelta(t, wantRootAngle, math.Min(coneAngles[0], math.Min(coneAngles[1], coneAngles[2])), 1e-6)
	require.InDelta(t, math.Pi/4, math.Max(coneAngles[0], math.Max(coneAngles[1], coneAngles[2])), 1e-6)

	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestReferenceBlankBaseHeight(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth = 8, 8
	p.BaseHeight = 1.5
	body, err := bevelgear.BuildBlank(t.Context(), decad.New(), p)
	require.NoError(t, err)

	wantHeelArea := math.Pi * math.Pow(4-p.BaseHeight, 2)
	foundHeel := false
	for _, face := range body.Faces() {
		if _, ok := face.Surface().(decad.Plane); !ok {
			continue
		}
		area, err := face.Area()
		require.NoError(t, err)
		if math.Abs(area.Value.Base()-wantHeelArea) < area.Bound.Base()+1e-7 {
			foundHeel = true
		}
	}
	require.True(t, foundHeel, "the explicit base height must set the heel radius")

	for _, height := range []float64{0.5, 4} {
		p.BaseHeight = height
		_, err := bevelgear.BuildBlank(t.Context(), decad.New(), p)
		require.ErrorContains(t, err, "base height")
	}
}

func TestReferenceBlankToeControls(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth = 8, 8
	p.ToeRadius, p.MateToeRadius = 1, 1
	plain, err := bevelgear.BuildBlank(t.Context(), decad.New(), p)
	require.NoError(t, err)
	plainBounds, err := plain.Bounds()
	require.NoError(t, err)
	plainVolume, err := plain.Volume()
	require.NoError(t, err)

	p.ToeExtension = 50
	extended, err := bevelgear.BuildBlank(t.Context(), decad.New(), p)
	require.NoError(t, err)
	extendedBounds, err := extended.Bounds()
	require.NoError(t, err)
	extendedVolume, err := extended.Volume()
	require.NoError(t, err)
	require.Less(t, extendedBounds.Min.X+extendedBounds.Bound.Base(),
		plainBounds.Min.X-plainBounds.Bound.Base())
	require.Greater(t, extendedVolume.Value.Base()-plainVolume.Value.Base(),
		extendedVolume.Bound.Base()+plainVolume.Bound.Base())
	mesh, err := extended.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())

	wantToeArea := math.Pi
	foundToe := false
	for _, face := range extended.Faces() {
		if _, ok := face.Surface().(decad.Plane); !ok {
			continue
		}
		area, err := face.Area()
		require.NoError(t, err)
		if math.Abs(area.Value.Base()-wantToeArea) < area.Bound.Base()+1e-7 {
			foundToe = true
		}
	}
	require.True(t, foundToe, "the explicit toe radius must set the toe end area")

	p.ToeExtension = 101
	_, err = bevelgear.BuildBlank(t.Context(), decad.New(), p)
	require.ErrorContains(t, err, "toe extension")
	p.ToeExtension = 50
	p.MateToeRadius = 3
	_, err = bevelgear.BuildBlank(t.Context(), decad.New(), p)
	require.ErrorContains(t, err, "toe-radius ceilings")
}

func TestUnequalReferenceBlankPair(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth = 16, 12
	p.BaseHeight = 3
	doc := decad.New()
	pair, err := bevelgear.BuildBlankPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	require.True(t, pair.Driving.IsSolid())
	require.True(t, pair.Pinion.IsSolid())

	// The shared apex is the origin. R=10 mm, and the pinion's fallback
	// height scales the explicit 3 mm driving height to 2.25 mm.
	for _, tc := range []struct {
		body         *decad.Body
		axisX, axisY float64
		wantHeel     float64
	}{
		{pair.Driving, 1, 0, 9},
		{pair.Pinion, 0, 1, 10.25},
	} {
		var heel float64
		planes := 0
		for _, face := range tc.body.Faces() {
			plane, ok := face.Surface().(decad.Plane)
			if !ok {
				continue
			}
			planes++
			normal := plane.Frame.N()
			require.InDelta(t, 1, math.Abs(normal.X*tc.axisX+normal.Y*tc.axisY), 1e-8)
			origin := plane.Frame.Origin()
			station := origin.X*tc.axisX + origin.Y*tc.axisY
			heel = math.Max(heel, station)
		}
		require.Equal(t, 2, planes)
		require.InDelta(t, tc.wantHeel, heel, 1e-7)
		mesh, err := tc.body.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		require.True(t, mesh.BoundaryVerified())
		require.True(t, mesh.VolumeVerified())
	}

	p.MateBaseHeight = 2
	explicit, err := bevelgear.BuildBlankPair(t.Context(), decad.New(), p)
	require.NoError(t, err)
	var explicitHeel float64
	for _, face := range explicit.Pinion.Faces() {
		plane, ok := face.Surface().(decad.Plane)
		if ok {
			explicitHeel = math.Max(explicitHeel, plane.Frame.Origin().Y)
		}
	}
	require.InDelta(t, 10, explicitHeel, 1e-7)

	p.MateBaseHeight = 5
	invalidDoc := decad.New()
	_, err = bevelgear.BuildBlankPair(t.Context(), invalidDoc, p)
	require.ErrorContains(t, err, "pinion: base height")
	require.Empty(t, invalidDoc.Bodies())
}

func TestObliqueReferenceBlankPair(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth = 16, 12
	p.ShaftAngle = 70 * math.Pi / 180
	doc := decad.New()
	pair, err := bevelgear.BuildBlankPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)

	pitchRadius := p.Module * float64(p.Teeth) / 2
	mateRadius := p.Module * float64(p.MateTeeth) / 2
	gamma := math.Atan2(math.Sin(p.ShaftAngle)*pitchRadius,
		mateRadius+math.Cos(p.ShaftAngle)*pitchRadius)
	coneDistance := pitchRadius / math.Sin(gamma)
	for _, tc := range []struct {
		body                            *decad.Body
		axisX, axisY, coneAngle, height float64
	}{
		{pair.Driving, 1, 0, gamma, p.Module * float64(p.Teeth) / 8},
		{pair.Pinion, math.Cos(p.ShaftAngle), math.Sin(p.ShaftAngle),
			p.ShaftAngle - gamma, p.Module * float64(p.MateTeeth) / 8},
	} {
		var heel float64
		for _, face := range tc.body.Faces() {
			plane, ok := face.Surface().(decad.Plane)
			if !ok {
				continue
			}
			normal := plane.Frame.N()
			require.InDelta(t, 1, math.Abs(normal.X*tc.axisX+normal.Y*tc.axisY), 1e-8)
			origin := plane.Frame.Origin()
			heel = math.Max(heel, origin.X*tc.axisX+origin.Y*tc.axisY)
		}
		require.InDelta(t, coneDistance*math.Cos(tc.coneAngle)+tc.height, heel, 1e-7)
		mesh, err := tc.body.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		require.True(t, mesh.BoundaryVerified())
		require.True(t, mesh.VolumeVerified())
	}
}
