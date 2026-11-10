package handcranksiren_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireSweep turns every vertex of the arm, the pin and the handle in 3 degree steps through a full turn about C and
// requires that none comes within 5 mm of the table. Every crank part also stays at least 0.5 behind the rear plate.
func requireSweep(t *testing.T, parts map[string]handcranksiren.Part) {
	crank := r3.Vec{X: 60, Z: 84}
	for _, name := range []string{"crank-arm", "crank-pin", "crank-handle"} {
		mesh, err := parts[name].Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyNone))
		require.NoError(t, err, name)
		for step := range 120 {
			xf, err := r3.RotationAround(crank, r3.Vec{Y: 1}, units.Radians(float64(step)*3*3.141592653589793/180))
			require.NoError(t, err)
			lowest := 1e9
			for _, v := range mesh.Vertices() {
				lowest = min(lowest, xf.Apply(v).Z)
			}
			require.GreaterOrEqual(t, lowest, 5.0, "%s at step %d stays off the table", name, step)
		}
	}
	for _, name := range []string{"crank-arm", "crank-pin", "crank-handle", "crank-pin-bolt", "crank-pin-nut"} {
		require.GreaterOrEqual(t, bounds(t, parts[name]).Min.Y, 37.6+0.5-1e-6, "%s clears the rear plate", name)
	}
}
