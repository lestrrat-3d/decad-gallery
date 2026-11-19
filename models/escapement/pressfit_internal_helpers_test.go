package escapement

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// boreWall returns the least wall round the bore of a section of a part about axis: the distance between the loop that
// stays nearest the axis (the bore) and every other loop of the section, measured from the vertices of each to the
// segments of the other.
func boreWall(t *testing.T, loops [][]Pt, axis Pt) float64 {
	t.Helper()
	reach := func(l []Pt) float64 {
		m := 0.0
		for _, p := range l {
			m = math.Max(m, p.Sub(axis).Len())
		}
		return m
	}
	bore := 0
	for i := range loops {
		if reach(loops[i]) < reach(loops[bore]) {
			bore = i
		}
	}
	require.Greater(t, len(loops), 1, "a section with a bore has at least two loops")
	least := math.Inf(1)
	for i, l := range loops {
		if i == bore {
			continue
		}
		for _, p := range loops[bore] {
			for j := range l {
				least = math.Min(least, distPointSeg(p, l[j], l[(j+1)%len(l)]))
			}
		}
		for _, p := range l {
			for j := range loops[bore] {
				least = math.Min(least, distPointSeg(p, loops[bore][j], loops[bore][(j+1)%len(loops[bore])]))
			}
		}
	}
	return least
}
