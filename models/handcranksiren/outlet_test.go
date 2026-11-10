package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireOutlet checks the air path of the collector and the horn with sections at fitTolerance. The collector is a
// rounded prism like the plates, whose mesh bound (0.01615) the finer sectionTolerance would refuse.
func requireOutlet(t *testing.T, parts map[string]handcranksiren.Part) {
	// solid reports whether the part's section on the plane through origin (spanned by u and v) holds the point at
	// radius and angle from centre in that plane. The angle runs from u toward v.
	section := func(name string, origin, u, v r3.Vec) func(centre [2]float64, radius, angle float64) bool {
		plane := frameAt(t, origin, u, v)
		outline, err := parts[name].Body.SectionOutline(t.Context(), plane, units.Millimeters(fitTolerance))
		require.NoError(t, err, name)
		s := sectionAt{outline: outline, origin: origin, u: u, v: v}
		return func(centre [2]float64, radius, angle float64) bool {
			return s.contains(u.Scale(centre[0] + radius*math.Cos(angle)).Add(v.Scale(centre[1] + radius*math.Sin(angle))))
		}
	}
	x, y, z := r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}
	around := func(f func(radius, angle float64) bool, radius float64, want bool, what string) {
		for k := range 8 {
			require.Equal(t, want, f(radius, float64(k)*math.Pi/4), "%s at %d degrees", what, k*45)
		}
	}

	// The rear wall is open inside the bore (radius 44 about R) and solid outside it.
	rear := section("collector", r3.Vec{Y: -35.90}, x, z)
	rearAt := func(radius, angle float64) bool { return rear([2]float64{0, 84}, radius, angle) }
	around(rearAt, 36, false, "collector bore")
	around(rearAt, 47, true, "collector rear wall")

	// The -X wall is open in the throat (radius 22 about the throat axis) and solid beyond it.
	wall := section("collector", r3.Vec{X: -48.5}, y, z)
	throat := func(radius, angle float64) bool { return wall([2]float64{-65.90, 84}, radius, angle) }
	require.False(t, throat(0, 0), "the throat centre is open")
	around(throat, 20, false, "throat")
	for k := range 4 {
		require.True(t, throat(23, float64(k)*math.Pi/2), "the wall is solid beside the throat at %d degrees", k*90)
	}

	// The horn's wall at X -100 is 32.92 to 35.42 from its axis.
	cone := section("horn", r3.Vec{X: -100}, y, z)
	horn := func(radius, angle float64) bool { return cone([2]float64{-65.90, 84}, radius, angle) }
	around(horn, 30, false, "horn bore")
	around(horn, 34.17, true, "horn wall")
}
