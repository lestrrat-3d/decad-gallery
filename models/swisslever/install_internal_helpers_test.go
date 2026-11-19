package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
)

// boreReach returns the least distance from axis to a segment of the section of body at z whose two ends lie within
// 4 mm of axis: the bore's nearest wall.
func boreReach(t *testing.T, body *decad.Body, z float64, axis Pt) float64 {
	t.Helper()
	loops, _ := sectionLoops(t, body, z)
	least := math.Inf(1)
	for _, l := range loops {
		for i, p := range l {
			q := l[(i+1)%len(l)]
			if p.Sub(axis).Len() > 4 || q.Sub(axis).Len() > 4 {
				continue
			}
			least = math.Min(least, segDist(axis, p, q))
		}
	}
	return least
}
