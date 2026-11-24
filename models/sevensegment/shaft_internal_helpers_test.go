package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// extent returns the largest |x| and the largest z (local) of the section loops, and the largest radius.
func extent(loops [][]r3.Vec) (maxX, maxZ, maxR float64) {
	for _, l := range loops {
		for _, p := range l {
			maxX = math.Max(maxX, math.Abs(p.X))
			maxZ = math.Max(maxZ, p.Y)
			maxR = math.Max(maxR, p.Len())
		}
	}
	return
}

func buildShaft(t *testing.T) (*run, *decad.Body) {
	t.Helper()
	r, b := sharedBody(t, "shaft", (*run).shaft)
	require.True(t, b.IsSolid())
	return r, b
}

// bisectY returns the y in [lo, hi] where pred of the section's largest radius changes, found by bisection on sections of
// body: pred is false at lo and true at hi, or true at lo and false at hi.
func bisectY(t *testing.T, body *decad.Body, lo, hi float64, pred func(float64) bool) float64 {
	t.Helper()
	radius := func(y float64) float64 {
		_, _, r := extent(sectionAt(t, body, y))
		return r
	}
	at := pred(radius(lo))
	require.NotEqual(t, at, pred(radius(hi)), "the predicate changes between %.2f and %.2f", lo, hi)
	for hi-lo > 0.004 {
		m := (lo + hi) / 2
		if pred(radius(m)) == at {
			lo = m
		} else {
			hi = m
		}
	}
	return (lo + hi) / 2
}
