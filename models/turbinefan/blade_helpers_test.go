package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// bladeSizes are the rotor diameters every blade test covers: the compact, default and large rotors.
var bladeSizes = []float64{128, 192, 224}

// bladePitches are the pitches every blade test covers: the least, the default and the greatest.
var bladePitches = []float64{turbinefan.MinBladePitch, turbinefan.DefaultBladePitch, turbinefan.MaxBladePitch}

func mustSize(t *testing.T, d float64) turbinefan.Size {
	t.Helper()
	sz, err := turbinefan.SizeFor(d)
	require.NoError(t, err)
	return sz
}

// requireThickness compares the ring's extent across the chord direction with its extent along it. The chord runs at
// angle from +X toward -Z.
func requireThickness(t *testing.T, want, angle float64, ring []r3.Vec, tol, y float64) {
	t.Helper()
	along := r3.Vec{X: math.Cos(angle), Z: -math.Sin(angle)}
	across := r3.Vec{X: math.Sin(angle), Z: math.Cos(angle)}
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, v := range ring {
		for i, axis := range []r3.Vec{along, across} {
			d := v.X*axis.X + v.Z*axis.Z
			lo[i], hi[i] = math.Min(lo[i], d), math.Max(hi[i], d)
		}
	}
	require.InDelta(t, want, (hi[1]-lo[1])/(hi[0]-lo[0]), tol, "thickness ratio at radius %g", y)
}
