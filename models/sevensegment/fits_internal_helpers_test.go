package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// fitBound is the allowance for the meshes of the sections a fit is measured on, in mm.
const fitBound = 0.04

// loopsGap returns the least distance between the vertices of a and the segments of b, and the reverse. The grids
// return nearest's distances exactly wherever they can lower the least distance found so far.
func loopsGap(a, b [][]r3.Vec) float64 {
	ga, gb := newSegmentGrid(a), newSegmentGrid(b)
	best := math.Inf(1)
	for _, p := range densify(a, 0.05) {
		best = math.Min(best, gb.nearestWithin(p, best))
	}
	for _, p := range densify(b, 0.05) {
		best = math.Min(best, ga.nearestWithin(p, best))
	}
	return best
}

// loopAt returns the loop of a section whose centroid lies within 0.1 of at and whose extent is within 0.25 of ext, or nil.
func loopAt(loops [][]r3.Vec, at r3.Vec, ext float64) []r3.Vec {
	for _, l := range loops {
		c, e := centroidExtent(l)
		if c.Sub(at).Len() < 0.1 && math.Abs(e-ext) < 0.25 {
			return l
		}
	}
	return nil
}

// nearLoops returns the loops of a section that lie within 6 mm of the horizontal line y.
func nearLoops(loops [][]r3.Vec, y float64) [][]r3.Vec {
	var out [][]r3.Vec
	for _, l := range loops {
		c, _ := centroidExtent(l)
		if math.Abs(c.Y-y) < 6 {
			out = append(out, l)
		}
	}
	return out
}

// nearCap returns the loops of a section that lie within 20 mm of the centre of the loops in cap.
func nearCap(loops, cap [][]r3.Vec) [][]r3.Vec {
	var centre r3.Vec
	n := 0.0
	for _, l := range cap {
		c, _ := centroidExtent(l)
		centre = centre.Add(c)
		n++
	}
	centre = centre.Scale(1 / n)
	var out [][]r3.Vec
	for _, l := range loops {
		c, _ := centroidExtent(l)
		if c.Sub(centre).Len() < 12 {
			out = append(out, l)
		}
	}
	return out
}

// bladePoint reports whether the section point p of segment s's slider lies on its blade (not on the tongue or pin): the
// blade is the thin plate that rises to z 83.55, so at z 81 it is the part away from the drive cone.
func bladePoint(s Segment, yi float64, p r3.Vec) (float64, float64, bool) {
	_, _, horizontal := segmentCentre(s)
	if horizontal {
		return p.X, p.Y, p.Y < yi-6.0
	}
	return p.X, p.Y, math.Abs(p.X) > 8.0
}

// requireDoubleD requires a bore section (points (x, z - ShaftZ) at theta 0) to have both flats, at z +-flat: a single-D
// bore lets its part turn twice as far on the shaft.
func requireDoubleD(t *testing.T, bore [][]r3.Vec, flat float64, name string) {
	t.Helper()
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, l := range bore {
		for _, p := range l {
			lo, hi = math.Min(lo, p.Y), math.Max(hi, p.Y)
		}
	}
	require.InDelta(t, flat, hi, fitBound, "%s: the bore's top flat", name)
	require.InDelta(t, -flat, lo, fitBound, "%s: the bore's bottom flat", name)
}
