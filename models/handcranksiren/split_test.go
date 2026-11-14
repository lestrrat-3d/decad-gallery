package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// splitTolerance is the chord tolerance of the split check's mesh.
const splitTolerance = 0.01

// requireSplitShafts checks each shaft against "Split shafts" in docs/model-pipeline.md. The split plane holds the
// shaft's axis and its flat's normal. Built at crank angle 0 the flat faces +Z and the plane's normal is +X; the pose
// turns both about −Y by the shaft's angle, so the normal is (cos a, 0, sin a).
func requireSplitShafts(t *testing.T, a *handcranksiren.Assembly, parts map[string]handcranksiren.Part) {
	t.Helper()
	s := a.State
	for _, shaft := range []struct {
		name   string
		point  r3.Vec
		angle  float64
		height float64 // the larger of the shaft and collar radii
	}{
		{"crank-shaft", s.Crank, s.CrankAngle, 5.5},
		{"intermediate-shaft", s.Intermediate, s.IntermediateAngle, 4.0},
		{"rotor-shaft", s.Rotor, s.RotorAngle, 5.5},
	} {
		t.Run(shaft.name, func(t *testing.T) {
			r, err := splitcheck.Check(t.Context(), splitcheck.Shaft{
				Body:  parts[shaft.name].Body,
				Point: shaft.point,
				Axis:  r3.Vec{Y: 1},
			}, splitTolerance)
			require.NoError(t, err)
			require.Empty(t, r.Violations)
			require.Equal(t, 1, r.Flats, "one flat along the whole shaft")
			want := r3.Vec{X: math.Cos(shaft.angle), Z: math.Sin(shaft.angle)}
			require.InDelta(t, 1, math.Abs(r.Normal.Dot(want)), 1e-9, "split normal %v", r.Normal)
			for i, h := range r.HalfHeights {
				require.InDelta(t, shaft.height, h, splitTolerance, "half %d", i+1)
			}
		})
	}
}
