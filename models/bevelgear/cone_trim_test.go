package bevelgear_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

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
			p := spiralDefaults()
			p.Teeth, p.MateTeeth, p.Samples = tc.teeth, tc.mateTeeth, 5
			p.ToothSpacing = tc.spacing

			insideDoc, insideToe := buildToeTrimmedTooth(t, p)
			insideHeel, err := bevelgear.BuildHeelCone(t.Context(), insideDoc, p)
			require.NoError(t, err)
			inside, err := decad.Intersect(insideToe, insideHeel).Do(t.Context())
			require.NoError(t, err)
			requireAnalyticTrim(t, inside)
			insideVolume, err := inside.Volume()
			require.NoError(t, err)

			outsideDoc, outsideToe := buildToeTrimmedTooth(t, p)
			outsideHeel, err := bevelgear.BuildHeelCone(t.Context(), outsideDoc, p)
			require.NoError(t, err)
			outside, err := decad.Cut(outsideToe, outsideHeel).Do(t.Context())
			require.NoError(t, err)
			requireAnalyticTrim(t, outside)
			outsideVolume, err := outside.Volume()
			require.NoError(t, err)

			require.Greater(t, insideVolume.Value.Base()-insideVolume.Bound.Base(),
				outsideVolume.Value.Base()+outsideVolume.Bound.Base())
		})
	}
}
