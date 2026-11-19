package screwgear_test

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// loopSpan is the largest distance between two points of the loop in the plane square to axis.
func loopSpan(loop []r3.Vec, axis r3.Vec) float64 {
	var span float64
	for i := range loop {
		for j := i + 1; j < len(loop); j++ {
			d := loop[i].Sub(loop[j])
			d = d.Sub(axis.Scale(d.Dot(axis)))
			span = math.Max(span, d.Len())
		}
	}
	return span
}
