package spinningwheel

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// The cord is a fixed length of wire. Moving the take-up screw trades tail length between the
// two ends and rotates the right eye, but it must not stretch or shrink the wire.
//
// Volume: subtracting the centreline length that depends on Scale leaves the two stock eyes and
// their splices, which are congruent in every case, so every case must admit one common residual.
// The held volume bound is about 3.7 mm^3, which is 6.5 mm of wire, so this only catches a gross
// change. Bounds: the rear run is fixed in Y while the right tail end moves back by half the travel, so
// the Y extent beyond that half must stay constant at a much finer resolution.
func TestBrakeCordConservesMaterial(t *testing.T) {
	const wire = 0.6
	area := math.Pi * wire * wire
	lo, hi := math.Inf(-1), math.Inf(1)
	for _, scale := range []float64{.8, 1, 1.2} {
		extentBase := math.NaN()
		for _, travel := range []float64{0, 3, 6} {
			p := DefaultParams()
			p.Scale, p.BrakeTravel = scale, travel
			r := readBrakeCord(t, p)
			centreline := math.Pi*42*scale + 612*scale - 2*brakeShift(scale)
			residual := r.volume - area*centreline
			t.Logf("scale %g travel %g: volume %g +/- %g, residual %g, Y %g..%g +/- %g",
				scale, travel, r.volume, r.volumeBound, residual, r.minY, r.maxY, r.boxBound)
			lo, hi = math.Max(lo, residual-r.volumeBound), math.Min(hi, residual+r.volumeBound)
			extent := r.maxY - r.minY - travel/2
			if math.IsNaN(extentBase) {
				extentBase = extent
				continue
			}
			require.InDelta(t, extentBase, extent, 4*r.boxBound+1e-6,
				"scale %g travel %g: the wire must move with the screw, not stretch", scale, travel)
		}
	}
	require.LessOrEqual(t, lo, hi, "the nine cords must share one residual volume")
}
