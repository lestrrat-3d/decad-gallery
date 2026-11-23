package sevensegment

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// pivotSection cuts body with the plane normal to segment seg's axis at u from its middle and returns its loops as
// (s, z - RotorAxisZ): the horizontal coordinate across the rotor and the height above the modeled axis.
func pivotSection(t *testing.T, body *decad.Body, seg Segment, u float64) [][]r3.Vec {
	t.Helper()
	f := rotorFrame(seg)
	origin := f.Origin().Add(f.N().Scale(u))
	return planeSection(t, body, origin, f.U(), r3.Vec{Z: 1})
}

// scanEdge returns the coordinate along one axis of a section at which the material of loops starts or ends, found by
// bisection between a point in the void (void) and a point in the material (solid), each given as the coordinate along
// the axis; at returns the section point for a coordinate.
func scanEdge(loops [][]r3.Vec, at func(c float64) r3.Vec, void, solid float64) float64 {
	lo, hi := void, solid
	for i := 0; i < 40; i++ {
		m := (lo + hi) / 2
		if insideLoops(loops, at(m)) == insideLoops(loops, at(solid)) {
			hi = m
		} else {
			lo = m
		}
	}
	return (lo + hi) / 2
}

// placedPivotBlock returns segment seg's block at the +u (end 1) or -u (end 0) end, in the world at turn 0.
func placedPivotBlock(t *testing.T, r *run, seg Segment, end int) *decad.Body {
	t.Helper()
	proto, err := r.pivotBlock()
	require.NoError(t, err)
	sgn := float64(2*end - 1)
	b, err := r.toSegment(seg, sgn, proto)
	require.NoError(t, err)
	return b
}
