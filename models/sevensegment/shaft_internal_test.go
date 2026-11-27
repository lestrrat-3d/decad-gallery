package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSevenSegmentShaft(t *testing.T) {
	t.Parallel()
	_, b := buildShaft(t)
	box, err := b.Bounds()
	require.NoError(t, err)
	require.InDelta(t, shaftFrontEnd, box.Min.Y, 1e-6)
	require.InDelta(t, rearInnerY+bearingWidth, box.Max.Y, 1e-6)
	// The cam section is a double D: round 4.8 at the sides, flats 4.05 above and below, until the flats' square end.
	loops := sectionAt(t, b, shaftCamFlatTo-0.2)
	_, maxZ, maxR := extent(loops)
	require.InDelta(t, shaftCamFlat, maxZ, 0.02)
	require.InDelta(t, shaftCamRadius, maxR, 0.02)
	minZ := math.Inf(1)
	for _, l := range loops {
		for _, p := range l {
			minZ = math.Min(minZ, p.Y)
		}
	}
	require.InDelta(t, -shaftCamFlat, minZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, shaftCamFlatTo+0.2))
	require.InDelta(t, shaftCamRadius, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, shaftStubFlatTo-0.25))
	require.InDelta(t, shaftStubFlat, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, shaftStubFlatTo+0.25))
	require.InDelta(t, shaftStubRadius, maxZ, 0.02)
	// Journals are round, the bearings' bore: radius 4.0 in the front bearing, and the same at the rear.
	for _, y := range []float64{frontInnerY - bearingWidth/2, rearInnerY + bearingWidth/2} {
		_, maxZ, maxR = extent(sectionAt(t, b, y))
		require.InDelta(t, shaftJournal, maxZ, 0.02)
		require.InDelta(t, shaftJournal, maxR, 0.02)
	}
	// The knob stub steps down inside the front wall's lip hole by a cone of 30 degrees or less (1.5 of radius over 2.6).
	_, _, maxR = extent(sectionAt(t, b, shaftStubStepFrom+0.1))
	require.InDelta(t, shaftJournal, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, (shaftStubStepFrom+shaftStubStepTo)/2))
	require.InDelta(t, (shaftJournal+shaftStubRadius)/2, maxR, 0.03)
	_, _, maxR = extent(sectionAt(t, b, shaftStubStepTo-0.1))
	require.InDelta(t, shaftStubRadius, maxR, 0.02)
	require.LessOrEqual(t, math.Atan((shaftJournal-shaftStubRadius)/(shaftStubStepFrom-shaftStubStepTo))*180/math.Pi, 30.0)
}

func TestSevenSegmentCollars(t *testing.T) {
	t.Parallel()
	_, b := buildShaft(t)
	// Front step: the cam section's diameter just behind the step, the journal's just in front of it.
	maxX, _, _ := extent(sectionAt(t, b, shaftCamFrom+0.25))
	require.InDelta(t, shaftCamRadius, maxX, 0.02)
	_, _, maxR := extent(sectionAt(t, b, shaftCamFrom-0.25))
	require.InDelta(t, shaftJournal, maxR, 0.02)
	// Rear collar: a cone of 30 degrees from the end of the cam section, then a cylinder of RearCollarRadius to the rear
	// bearing's inner face less the Tight gap.
	_, _, maxR = extent(sectionAt(t, b, shaftCamEnd-0.05))
	require.InDelta(t, shaftCamRadius, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, shaftRearFrom-1.25))
	require.InDelta(t, RearCollarRadius, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, shaftRearFrom+0.05))
	require.InDelta(t, shaftJournal, maxR, 0.02)
	// The collar's two ends, found on the built body: the cone starts where the round section grows past the cam section's
	// radius, and the cylinder ends where it falls back below the middle of the collar and the journal (Arbor collars: at
	// least 3.0 mm apart, measured from where the cone meets the smaller diameter).
	start := bisectY(t, b, shaftCamEnd-0.3, shaftCamEnd+0.8, func(r float64) bool { return r > shaftCamRadius+0.01 })
	end := bisectY(t, b, shaftRearFrom-0.75, shaftRearFrom+0.75, func(r float64) bool { return r < (RearCollarRadius+shaftJournal)/2 })
	// The 0.01 of radius the probe needs on the cone's 30 degree slope puts the found start about 0.02 past the apex.
	require.InDelta(t, shaftCamEnd, start, 0.04, "the collar's cone starts")
	require.InDelta(t, shaftRearFrom, end, 0.01, "the collar's cylinder ends")
	require.GreaterOrEqual(t, end-start, 3.0, "rear collar length")
	require.InDelta(t, shaftRearFrom-shaftCamEnd, end-start, 0.05, "rear collar length")
	// Keyed hub: base 3.5 long on the stub, cup to the rotor's stop, diameter 21.
	_, hub := sharedBody(t, "hub", (*run).hub)
	box, err := hub.Bounds()
	require.NoError(t, err)
	require.InDelta(t, hubRimY, box.Min.Y, 1e-6)
	require.InDelta(t, hubBaseY, box.Max.Y, 1e-6)
	require.InDelta(t, 2*hubRadius, box.Max.X-box.Min.X, 0.01)
}
