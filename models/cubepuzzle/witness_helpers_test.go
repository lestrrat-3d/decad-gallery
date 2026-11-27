package cubepuzzle

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireWitnessLocation(t *testing.T, name string, body *decad.Body, point r3.Vec, want decad.PointLocation) {
	t.Helper()
	got, err := body.LocatePoint(t.Context(), point, units.Millimeters(.01))
	require.NoError(t, err, "%s", name)
	require.Equal(t, want, got, "%s", name)
}

// requireWitnessPath proves the path stays off the boundary between samples:
// each sample clears the boundary by more than half the sample spacing.
func requireWitnessPath(t *testing.T, name string, body *decad.Body, start, end r3.Vec, want decad.PointLocation, steps int) {
	t.Helper()
	requireWitnessLocation(t, name, body, start, want)
	minimum := math.Inf(1)
	for i := 0; i <= steps; i++ {
		p := start.Add(end.Sub(start).Scale(float64(i) / float64(steps)))
		for _, face := range body.Faces() {
			d, err := face.DistanceToPoint(t.Context(), p, units.Millimeters(.01))
			require.NoError(t, err, "%s boundary sample %d", name, i)
			minimum = min(minimum, d.Value.Base()-d.Bound.Base())
		}
	}
	require.Greater(t, minimum, start.Sub(end).Len()/float64(2*steps)+.001, "%s boundary clearance", name)
}
