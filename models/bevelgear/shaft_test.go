package bevelgear

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// A pinion shaft that is not along +X needs its section on a grid where every on-axis point lies exactly on the
// shaft vector and the two ends of an end wall share one exact axial reading. Decad refuses the join otherwise.
func TestShaftSectionIsExactOnItsAxis(t *testing.T) {
	section := []axialPoint{
		{station: 3.0371234567, radius: 0},
		{station: 21.9317654321, radius: 0},
		{station: 21.9317654321, radius: 14.8123456789},
		{station: 18.2, radius: 17.1},
		{station: 3.0371234567, radius: 9.7654321},
	}
	for _, degrees := range []float64{30, 45, 70, 90, 110, 150} {
		f := newShaftFrame(degrees * math.Pi / 180)
		xy, err := f.sectionXY(section)
		require.NoError(t, err)

		axial := func(point [2]float64) *big.Rat {
			z := new(big.Rat).Mul(new(big.Rat).SetFloat64(point[0]), new(big.Rat).SetFloat64(f.cos))
			return z.Add(z, new(big.Rat).Mul(new(big.Rat).SetFloat64(point[1]), new(big.Rat).SetFloat64(f.sin)))
		}
		radial := func(point [2]float64) *big.Rat {
			r := new(big.Rat).Mul(new(big.Rat).SetFloat64(point[1]), new(big.Rat).SetFloat64(f.cos))
			return r.Sub(r, new(big.Rat).Mul(new(big.Rat).SetFloat64(point[0]), new(big.Rat).SetFloat64(f.sin)))
		}
		for i, point := range section {
			if point.radius == 0 {
				require.Zero(t, radial(xy[i]).Sign(), "%g degrees: on-axis point %d is off the shaft", degrees, i)
			}
		}
		require.Zero(t, axial(xy[1]).Cmp(axial(xy[2])), "%g degrees: the heel wall is not exactly perpendicular", degrees)
		require.Zero(t, axial(xy[0]).Cmp(axial(xy[4])), "%g degrees: the toe wall is not exactly perpendicular", degrees)
	}
}

func TestLocalShaftKeepsItsSection(t *testing.T) {
	section := []axialPoint{{station: 1.23456789, radius: 0}, {station: 2.3456789, radius: 4.56789123}}
	xy, err := localShaft.sectionXY(section)
	require.NoError(t, err)
	require.Equal(t, [][2]float64{{1.23456789, 0}, {2.3456789, 4.56789123}}, xy)
	require.True(t, newShaftFrame(0).local())
}
