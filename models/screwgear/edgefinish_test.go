package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSleeveEndFacesAreChamfered checks that both rims of both sleeve end faces have a 45 degree bevel beside them.
// The sleeve is built in the pair frame with no placement, so its end faces lie at z = ±CageRise.
func TestSleeveEndFacesAreChamfered(t *testing.T) {
	p := shortParams()
	m, _ := build(t, p)
	ends := 0
	for _, f := range m.Cage.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(math.Abs(plane.Frame.N().Z)-1) > 1e-9 ||
			math.Abs(math.Abs(plane.Frame.Origin().Z)-p.CageRise) > 1e-6 {
			continue
		}
		area, err := f.Area()
		require.NoError(t, err)
		// The chamfer cuts the 584 mm² face down to about 487 mm², and the face is still large.
		require.GreaterOrEqual(t, area.Value.Base(), edgefinish.LargeFaceArea)
		require.Less(t, area.Value.Base(), 560.0)
		ends++
		require.Len(t, f.Loops(), 2)
		for _, loop := range f.Loops() {
			for _, e := range loop.Edges() {
				bevels := 0
				for _, other := range e.Faces() {
					cone, ok := other.Surface().(decad.Cone)
					if other == f || !ok {
						continue
					}
					half, err := cone.HalfAngle.In(units.Radians(1).Unit())
					require.NoError(t, err)
					if math.Abs(half-math.Pi/4) < 1e-6 || math.Abs(half-3*math.Pi/4) < 1e-6 {
						bevels++
					}
				}
				require.Equal(t, 1, bevels, "an end-face rim edge has no 45 degree bevel")
			}
		}
	}
	require.Equal(t, 2, ends)
}
