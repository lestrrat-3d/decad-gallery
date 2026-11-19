package swisslever

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// minPressWall is the thinnest wall round a press-fit bore: docs/model-pipeline.md Corners and roots, Root width.
const minPressWall = 1.6

// loopWall returns the least distance between two section loops, from the vertices of each to the segments of the
// other.
func loopWall(a, b outline) float64 {
	return polyDist(a, b)
}

// boreWall returns the least wall round the bore of a section of a part about axis: the distance between the loop that
// stays nearest the axis (the bore) and every other loop of the section.
func boreWall(t *testing.T, loops []outline, axis Pt) float64 {
	t.Helper()
	require.Greater(t, len(loops), 1, "a section with a bore has at least two loops")
	bore := 0
	for i := range loops {
		if extent(loops[i], axis) < extent(loops[bore], axis) {
			bore = i
		}
	}
	least := math.Inf(1)
	for i, l := range loops {
		if i != bore {
			least = math.Min(least, loopWall(loops[bore], l))
		}
	}
	return least
}
