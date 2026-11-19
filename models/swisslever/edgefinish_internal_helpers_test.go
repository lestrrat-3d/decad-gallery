package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// bevel is the volume a 0.5 mm 45° chamfer removes per millimetre of straight edge.
const bevel = 0.5 * 0.5 / 2

// perimeter returns the length of the closed polygon.
func perimeter(p outline) float64 {
	sum := 0.0
	for i := range p {
		sum += p[i].Sub(p[(i+1)%len(p)]).Len()
	}
	return sum
}

// extent returns the greatest distance of a vertex of p from c.
func extent(p outline, c Pt) float64 {
	m := 0.0
	for _, v := range p {
		m = math.Max(m, v.Sub(c).Len())
	}
	return m
}

// outerAndInner splits the loops of body at z into the outer loop (the one that reaches farthest from c) and the others.
func outerAndInner(t *testing.T, body *decad.Body, z float64, c Pt) (outline, []outline) {
	t.Helper()
	loops, _ := sectionLoops(t, body, z)
	best := 0
	for i := range loops {
		if extent(loops[i], c) > extent(loops[best], c) {
			best = i
		}
	}
	var inner []outline
	for i, l := range loops {
		if i != best {
			inner = append(inner, l)
		}
	}
	return loops[best], inner
}

// volumeOf returns the body's volume in mm³.
func volumeOf(t *testing.T, body *decad.Body) float64 {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}
