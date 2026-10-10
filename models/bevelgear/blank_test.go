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
