package sevensegment_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// radii returns the least distance of the outline of loops from the shaft axis (to its segments, so that a flat counts
// at its middle) and the greatest distance of its vertices.
func radii(loops [][]r3.Vec) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, l := range loops {
		for i, a := range l {
			b := l[(i+1)%len(l)]
			ab := b.Sub(a)
			k := 0.0
			if d := ab.Dot(ab); d > 0 {
				k = math.Min(math.Max(-a.Dot(ab)/d, 0), 1)
			}
			lo = math.Min(lo, a.Add(ab.Scale(k)).Len())
			hi = math.Max(hi, a.Len())
		}
	}
	return lo, hi
}

// shaftEdge returns the y in [lo, hi] where pred of the shaft section's least and greatest radius changes value, found
// by bisection on sections of the built shaft. The least radius is the flat's distance wherever the section has a flat
// and the round's radius elsewhere, so the result does not depend on how far the camshaft is turned.
func shaftEdge(t *testing.T, shaft *decad.Body, lo, hi float64, pred func(minR, maxR float64) bool) float64 {
	t.Helper()
	at := func(y float64) bool {
		minR, maxR := radii(sevensegment.SectionAt(t, shaft, y))
		return pred(minR, maxR)
	}
	first := at(lo)
	require.NotEqual(t, first, at(hi), "the shaft's section changes between y %.2f and %.2f", lo, hi)
	for hi-lo > 0.004 {
		m := (lo + hi) / 2
		if at(m) == first {
			lo = m
		} else {
			hi = m
		}
	}
	return (lo + hi) / 2
}

// faceCoord returns the coordinate along axis of the face of body that separates a solid point from a void one, found by
// bisection with the mesh's crossing grid. from is a point in the material and to a point in the void beyond the face; the
// coordinate moves along the axis from one to the other.
func faceCoord(t *testing.T, g *crossingGrid, from, to r3.Vec) r3.Vec {
	t.Helper()
	require.True(t, g.pointIn(from), "%v lies in the material", from)
	require.False(t, g.pointIn(to), "%v lies in the void", to)
	for to.Sub(from).Len() > 0.004 {
		m := from.Add(to).Scale(0.5)
		if g.pointIn(m) {
			from = m
		} else {
			to = m
		}
	}
	return from.Add(to).Scale(0.5)
}
