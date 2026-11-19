package spurgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func requireBoreRims(t *testing.T, gear *decad.Body, radius float64) {
	t.Helper()
	for _, cap := range []decad.FeatureRef{decad.CapStart(gear), decad.CapEnd(gear)} {
		faces, err := decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(gear)
		require.NoError(t, err)
		require.Len(t, faces[0].Loops(), 2)
		require.Len(t, faces[0].Loops()[1].Edges(), 1, "the bore rim stays sharp")
		circle, ok := faces[0].Loops()[1].Edges()[0].Curve().(decad.Circle3)
		require.True(t, ok)
		require.InDelta(t, radius, circle.Radius.Base(), 1e-10)
	}
}
