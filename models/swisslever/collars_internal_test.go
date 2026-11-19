package swisslever

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOutsideCollar checks the Knob Collar, the one separate collar of the movement, on the built, posed bodies: the
// part of docs/model-pipeline.md Arbor collars that concerns a part pressed on a shaft end outside the frame. The shaft
// half of the rule, the shoulders and body steps, is TestArborCollars.
func TestOutsideCollar(t *testing.T) {
	_, m := sharedDefault(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	// Its length: at least 3.0 mm, measured over its whole body.
	lo, hi := zRange(t, m.KnobCollar)
	require.InDelta(t, collarLow, lo, 2e-3, "rear face")
	require.GreaterOrEqual(t, hi-lo, minCollar-2e-3, "length")

	// Its diameter: the keyed ring's arc reaches 3.7 about the arbor; its bore chord stands at 1.55.
	_, outer, off := seatRadii(t, m.KnobCollar, 35, g.O)
	require.InDelta(t, CollarRadius, outer, 0.01+off, "outer radius")
	require.InDelta(t, EndBoreFlat, boreReach(t, m.KnobCollar, 35, g.O), 0.02+off, "bore chord")
	require.GreaterOrEqual(t, CollarRadius-EndBoreRadius, minPressWall-1e-9, "wall round the bore")
	require.GreaterOrEqual(t, collarFlat-EndBoreFlat, minPressWall, "wall at the flats")

	// Its place: between the front plate's outer face and the knob, stopped by the end of the arbor's flat. It stands
	// 0.25 from the plate (Tight) and the knob's counterbore floor stands 0.05 from it (press stop).
	axial(t, m.KnobCollar, collarLow, m.FrontPlate, frontPlateHigh, 0.25, "ring to the front plate's outer face")
	hasPlane(t, m.EscapeArbor, collarLow, "the arbor's flat starts where the ring does")
	axial(t, m.WindingKnob, keyedCounterLow+keyedCounterLong, m.KnobCollar, collarLow+collarLong, 0.05, "knob floor to ring")

	// Its fit: it enters the counterbore by its whole length, which leaves the knob's floor 1.0 thick; the wall and the
	// counterbore take the press-fit clearance.
	require.GreaterOrEqual(t, knobHigh-(keyedCounterLow+keyedCounterLong), 1.0-1e-9, "knob material under the counterbore")
	radial(t, m.WindingKnob, m.KnobCollar, 35, 0.05, 0, "ring in the counterbore")

	// The carried part stands clear of the plate: the knob's rear face is 1.30 off its outer face (Loose).
	axial(t, m.WindingKnob, knobLow, m.FrontPlate, frontPlateHigh, 1.30, "knob to the front plate")
}
