package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestClickerLandGaps reads the gaps the tooth lands leave between the click indexer's mating teeth, from the specs the
// builders use. A tip is cut back by Tip and a root stands Plate above its valley, so each pair of mating teeth keeps a gap
// where tip meets root. Y grows toward the front wall.
func TestClickerLandGaps(t *testing.T) {
	in, out, cr, bt := rotorInnerTeeth(), rotorOuterTeeth(), crownTeeth(), buttonTeeth()
	rotorPeak := func(f faceTeeth) float64 { return rotorPlateY - f.top() }
	rotorLand := func(f faceTeeth) float64 { return rotorPlateY - f.Plate }
	crownTip := housingCrownY + cr.top()
	crownRoot := housingCrownY + cr.Plate
	buttonTip := buttonUndersideY + bt.top()
	buttonRoot := buttonUndersideY + bt.Plate
	require.Equal(t, in.Plate, out.Plate, "the rotor's plate has one root land under both rings")
	const min = 0.02 - 1e-9
	for _, c := range []struct {
		name string
		gap  float64
	}{
		{"rotor outer peak to the crown's notch", rotorPeak(out) - crownRoot},
		{"crown tip to the rotor's outer land", rotorLand(out) - crownTip},
		{"rotor inner peak to the button's tip at rest", rotorPeak(in) - buttonTip},
		{"rotor inner peak to the button's root, pressed", rotorPeak(in) + RotorTravel - (buttonRoot + ButtonStroke)},
		{"button tip to the rotor's inner land, pressed", rotorLand(in) + RotorTravel - (buttonTip + ButtonStroke)},
	} {
		t.Logf("%s: %.3f mm", c.name, c.gap)
		require.GreaterOrEqual(t, c.gap, min, c.name)
	}
}

// TestClickerPressTurn derives the press's turn and the timing margin from the tooth specs: the ramp's 32.5 degrees less
// the turn each land takes, 8.125 degrees per mm of height.
func TestClickerPressTurn(t *testing.T) {
	in, bt := rotorInnerTeeth(), buttonTeeth()
	require.InDelta(t, 36, in.Rise+in.Fall, 1e-12)
	require.InDelta(t, 36, bt.Rise+bt.Fall, 1e-12)
	require.InDelta(t, RampAngle/ToothHeight, 8.125, 1e-12)
	loss := 8.125 * (bt.Tip + math.Max(in.Tip, bt.Plate))
	require.InDelta(t, 0.81, loss, 0.01)
	require.InDelta(t, RampAngle-loss, PressTurn(), 1e-12)
	require.InDelta(t, 31.69, PressTurn(), 0.01)
	require.InDelta(t, 4.31, ReleaseTurn(), 0.01)
	require.InDelta(t, 36, PressTurn()+ReleaseTurn(), 1e-12)
	// The cams finish their ramps by SegmentsDoneBy, which the press must pass with room to spare.
	require.GreaterOrEqual(t, SegmentsDoneBy, RampEnd+TimingAllowance-1.5+0.05)
	require.InDelta(t, 2.14, TimingMargin(), 0.02)
	require.GreaterOrEqual(t, TimingMargin(), 2.0)
}

// TestClickerPoses walks a press and a release: the rotor drops with the button, then turns, and the release ends one step on.
func TestClickerPoses(t *testing.T) {
	drop, turn := PressPose(0)
	require.Zero(t, drop)
	require.Zero(t, turn)
	prevDrop, prevTurn := 0.0, 0.0
	for s := 0.0; s <= ButtonStroke+1e-9; s += 0.1 {
		drop, turn = PressPose(s)
		require.GreaterOrEqual(t, drop, prevDrop-1e-12)
		require.GreaterOrEqual(t, turn, prevTurn-1e-12)
		require.LessOrEqual(t, drop, RotorTravel+1e-12)
		prevDrop, prevTurn = drop, turn
	}
	require.InDelta(t, RotorTravel, drop, 1e-12)
	require.InDelta(t, PressTurn(), turn, 1e-12)
	for s := ButtonStroke; s >= 0; s -= 0.1 {
		d, tn := ReleasePose(s)
		require.GreaterOrEqual(t, tn, prevTurn-1e-12, "the release never turns the shaft back")
		require.GreaterOrEqual(t, d, -1e-12)
		prevTurn = tn
	}
	drop, turn = ReleasePose(0)
	require.InDelta(t, 0, drop, 1e-9)
	require.InDelta(t, ClickerStep, turn, 1e-9)
	// Ten presses turn the shaft once round: 9 to 0 is a step like any other.
	require.InDelta(t, 360, float64(ClickerSteps)*ClickerStep, 1e-12)
}

// TestClickerRadialFits reads the radial fits of the click indexer from its dimensions: the rotor's plate to the tube
// (Regular 0.35), the rotor's skirt on the spring (Tight 0.2), the hub's keys in the skirt's slots (Tight 0.2 a side), the
// button's head to the crown ring and its lugs in the slots (Loose 0.5).
func TestClickerRadialFits(t *testing.T) {
	const springR = 7.62 / 2
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"rotor plate to the tube", housingTubeInner - rotorPlateRadius, RegularFit},
		{"skirt on the spring", rotorSkirtInner - springR, TightFit - 0.05},
		{"hub key in the skirt's slot, a side", (rotorSlotWidth - hubKeyWidth) / 2, 0.2},
		{"head to the crown ring", housingCrownR0 - buttonRadius, 0.55},
		{"lug in the slot, a side", (housingSlotWidth - buttonKeyWidth) / 2, 0.5},
		{"lug to the slot's outer wall", housingSlotOuter - buttonKeyOuter, 0.5},
	} {
		t.Logf("%s: %.3f mm", c.name, c.got)
		require.GreaterOrEqual(t, c.got, c.want-0.06, c.name)
	}
}
