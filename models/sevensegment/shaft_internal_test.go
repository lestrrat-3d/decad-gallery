package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSevenSegmentShaft(t *testing.T) {
	_, b := buildShaft(t)
	box, err := b.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -72.2, box.Min.Y, 1e-6)
	require.InDelta(t, 48.0, box.Max.Y, 1e-6)
	// The cam section is a double D: round 3.2 at the sides, flats 2.7 above and below, until the flats' square end at 39.2.
	loops := sectionAt(t, b, 39.0)
	_, maxZ, maxR := extent(loops)
	require.InDelta(t, shaftCamFlat, maxZ, 0.02)
	require.InDelta(t, 3.2, maxR, 0.02)
	minZ := math.Inf(1)
	for _, l := range loops {
		for _, p := range l {
			minZ = math.Min(minZ, p.Y)
		}
	}
	require.InDelta(t, -shaftCamFlat, minZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, 39.4))
	require.InDelta(t, 3.2, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, -58.1))
	require.InDelta(t, shaftStubFlat, maxZ, 0.02)
	_, maxZ, _ = extent(sectionAt(t, b, -57.6))
	require.InDelta(t, 2.5, maxZ, 0.02)
	// Journals are round 2.5.
	_, maxZ, maxR = extent(sectionAt(t, b, -52))
	require.InDelta(t, 2.5, maxZ, 0.02)
	require.InDelta(t, 2.5, maxR, 0.02)
}

func TestSevenSegmentCollars(t *testing.T) {
	_, b := buildShaft(t)
	// Front step: diameter 6.4 just behind -49.25.
	maxX, _, _ := extent(sectionAt(t, b, -49.0))
	require.InDelta(t, 3.2, maxX, 0.02)
	_, _, maxR := extent(sectionAt(t, b, -49.5))
	require.InDelta(t, 2.5, maxR, 0.02)
	// Rear collar: cone 39.7 to 40.739, cylinder of diameter 7.6 to 42.75: 3.05 long.
	_, _, maxR = extent(sectionAt(t, b, 39.65))
	require.InDelta(t, 3.2, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, 41.5))
	require.InDelta(t, 7.6/2, maxR, 0.02)
	_, _, maxR = extent(sectionAt(t, b, 42.8))
	require.InDelta(t, 2.5, maxR, 0.02)
	// The collar's two ends, found on the built body: the cone starts where the round section grows past 3.2, and the
	// cylinder ends where it falls back below 3.0 (Arbor collars: at least 3.0 mm apart, 3.05 as drawn).
	start := bisectY(t, b, 39.4, 40.5, func(r float64) bool { return r > 3.21 })
	end := bisectY(t, b, 42.0, 43.5, func(r float64) bool { return r < 3.0 })
	// The 0.01 of radius the probe needs on the cone's 30 degree slope puts the found start about 0.02 past the apex.
	require.InDelta(t, 39.7, start, 0.04, "the collar's cone starts")
	require.InDelta(t, 42.75, end, 0.01, "the collar's cylinder ends")
	require.GreaterOrEqual(t, end-start, 3.0, "rear collar length")
	require.InDelta(t, 3.05, end-start, 0.05, "rear collar length")
	// Keyed hub: base 3.5 long on the stub, cup to the rotor's stop, diameter 21.
	_, hub := sharedBody(t, "hub", (*run).hub)
	box, err := hub.Bounds()
	require.NoError(t, err)
	require.InDelta(t, hubRimY, box.Min.Y, 1e-6)
	require.InDelta(t, hubBaseY, box.Max.Y, 1e-6)
	require.InDelta(t, 2*hubRadius, box.Max.X-box.Min.X, 0.01)
}
