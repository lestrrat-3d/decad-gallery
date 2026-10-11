package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func buildToeTrimmedTooth(t *testing.T, p bevelgear.Params) (*decad.Document, *decad.Body) {
	t.Helper()
	doc := decad.New()
	tooth, err := bevelgear.BuildPointTooth(t.Context(), doc, p)
	require.NoError(t, err)
	cone, err := bevelgear.BuildToeCone(t.Context(), doc, p)
	require.NoError(t, err)
	trimmed, err := decad.Cut(t.Context(), tooth, cone)
	require.NoError(t, err)
	require.True(t, trimmed.IsSolid())
	return doc, trimmed
}

func requireAnalyticTrim(t *testing.T, body *decad.Body) {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	var coneFaces, freeformFaces int
	for _, face := range body.Faces() {
		switch face.Surface().(type) {
		case decad.Cone:
			coneFaces++
		case decad.NURBSSurface:
			freeformFaces++
		}
	}
	require.Positive(t, coneFaces)
	require.Positive(t, freeformFaces)
}

func TestRealToeAndHeelTrimKeepLargerAnalyticTooth(t *testing.T) {
	for _, tc := range []struct {
		name             string
		teeth, mateTeeth int
		spacing          float64
	}{
		{name: "eight equal", teeth: 8, mateTeeth: 8},
		{name: "sixteen twelve spaced", teeth: 16, mateTeeth: 12, spacing: 0.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := bevelgear.DefaultParams()
			p.Teeth, p.MateTeeth, p.Samples = tc.teeth, tc.mateTeeth, 5
			p.ToothSpacing = tc.spacing

			insideDoc, insideToe := buildToeTrimmedTooth(t, p)
			insideHeel, err := bevelgear.BuildHeelCone(t.Context(), insideDoc, p)
			require.NoError(t, err)
			inside, err := decad.Intersect(t.Context(), insideToe, insideHeel)
			require.NoError(t, err)
			requireAnalyticTrim(t, inside)
			insideVolume, err := inside.Volume()
			require.NoError(t, err)

			outsideDoc, outsideToe := buildToeTrimmedTooth(t, p)
			outsideHeel, err := bevelgear.BuildHeelCone(t.Context(), outsideDoc, p)
			require.NoError(t, err)
			outside, err := decad.Cut(t.Context(), outsideToe, outsideHeel)
			require.NoError(t, err)
			requireAnalyticTrim(t, outside)
			outsideVolume, err := outside.Volume()
			require.NoError(t, err)

			require.Greater(t, insideVolume.Value.Base()-insideVolume.Bound.Base(),
				outsideVolume.Value.Base()+outsideVolume.Bound.Base())
		})
	}
}
