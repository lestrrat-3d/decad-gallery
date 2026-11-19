package ballbearing_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func requireBore(t *testing.T, body *decad.Body, radius float64, axis r3.Vec) {
	t.Helper()
	caps := 0
	for _, face := range body.Faces() {
		if face.Surface().Kind() != decad.KindPlane {
			continue
		}
		require.Equal(t, 2, len(face.Loops()))
		require.Equal(t, 1, len(face.Loops()[1].Edges()))
		circle, ok := face.Loops()[1].Edges()[0].Curve().(decad.Circle3)
		require.True(t, ok)
		require.InDelta(t, radius, circle.Radius.Base(), 1e-8)
		require.InDelta(t, 1, math.Abs(circle.Axis.Dot(axis)), 1e-8)
		caps++
	}
	require.Equal(t, 2, caps)
}
