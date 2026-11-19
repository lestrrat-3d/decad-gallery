package escapement

import (
	"testing"

	"github.com/lestrrat-3d/decad"
)

// perimeter returns the length of the closed polygon.
func perimeter(p []Pt) float64 {
	sum := 0.0
	for i := range p {
		sum += p[i].Sub(p[(i+1)%len(p)]).Len()
	}
	return sum
}

// bevel is the volume a 0.5 mm 45° chamfer removes per millimetre of straight edge.
const bevel = 0.5 * 0.5 / 2

func centroidLen(l []Pt) float64 { return centroid(l).Len() }

// plateRims is the length of the outer loop and the two window loops of a plate at height z.
func plateRims(t *testing.T, body *decad.Body, z float64) float64 {
	t.Helper()
	sum := 0.0
	for _, l := range loopsBy(sectionAt(t, body, z)) {
		// The outer loop and the windows are the loops longer than 100 mm; holes, pockets and sockets are shorter.
		if perimeter(l) > 100 {
			sum += perimeter(l)
		}
	}
	return sum
}
