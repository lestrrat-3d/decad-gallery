package sevensegment

// The click indexer's motion. A press has two phases. The button first pushes the rotor down RotorTravel, with no turn. It
// then drives the rotor round on the matching ramps, and the rotor takes the hub and the shaft with it. The release lets the
// spring lift the rotor onto the fixed crown's short ramp, which turns it the rest of the step.

// SegmentsDoneBy is the shaft angle by which every cam has finished moving its segment: the cam ramp end plus the timing
// allowance of the slider and rotor play (instructions.md).
const SegmentsDoneBy = 29.55

// rampRate is the rotor's turn per mm of button travel on the ramps.
const rampRate = RampAngle / ToothHeight

// PressTurn is the shaft's turn at the bottom of a press: the ramp's angle less what the lands take. Each land (the
// tip's, then the root's, or the tip's again when it is wider) costs the ramp's turn over its height.
func PressTurn() float64 {
	in := rotorInnerTeeth()
	bt := buttonTeeth()
	root := max(in.Tip, bt.Plate)
	return RampAngle - rampRate*(bt.Tip+root)
}

// TimingMargin is how far past the last cam's end the press turns the shaft, degrees.
func TimingMargin() float64 { return PressTurn() - SegmentsDoneBy }

// ReleaseTurn is the turn the release adds, degrees.
func ReleaseTurn() float64 { return ClickerStep - PressTurn() }

// releaseReach is the button drop below which the rotor begins to run up the crown's short ramp.
const releaseReach = ButtonStroke - 1.0

// PressPose returns the rotor's drop and turn at button drop s (0 to ButtonStroke) on the way down.
func PressPose(s float64) (drop, turn float64) {
	if s <= RotorTravel {
		return s, 0
	}
	return RotorTravel, PressTurn() * (s - RotorTravel) / (ButtonStroke - RotorTravel)
}

// ReleasePose returns the rotor's drop and turn at button drop s on the way up. The rotor follows the button until its
// outer teeth reach the crown's short ramp, then runs up the ramp as the button clears.
func ReleasePose(s float64) (drop, turn float64) {
	if s >= releaseReach {
		return s - (ButtonStroke - RotorTravel), PressTurn()
	}
	turn = PressTurn() + ReleaseTurn()*(1-s/releaseReach)
	return CrownHeight * (ClickerStep - turn) / (ClickerStep - CrownBack), turn
}
