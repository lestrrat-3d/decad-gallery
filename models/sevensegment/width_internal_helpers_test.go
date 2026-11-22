package sevensegment

import "math"

// rotorEndPlay is how far a rotor can move along its axis: the bore's end wall stands 0.25 from the rotor's end face.
const rotorEndPlay = 0.25

// rotorSettledZ is the rotor axis height settled in its half-holes (instructions.md "Z stack": 82.1 against 82.6).
const rotorSettledZ = RotorAxisZ - 0.5

// rotorSweptRadius returns an upper bound on the radius an a, g or d rotor sweeps at distance u (u >= 0) from its
// middle along its axis: the largest of
//   - the rounded square's (RotorHalfWidth-rotorEdgeRound)*√2+rotorEdgeRound, capped by the pyramid's
//     (rotorSquareHalf+RotorHalfWidth-u)*√2, for u <= RotorHalfLength;
//   - the stub's root disc 1.3 for 11.75 <= u <= 12.15 and its 45° cone 1.3-(u-12.15) for 12.15 < u <= 12.85;
//   - JournalRadius for u <= 14.25;
//
// and 0 beyond u 14.25.
func rotorSweptRadius(u float64) float64 {
	u = math.Max(u, 0)
	best := 0.0
	if u <= RotorHalfLength {
		square := (RotorHalfWidth-rotorEdgeRound)*math.Sqrt2 + rotorEdgeRound
		best = math.Min(square, (rotorSquareHalf+RotorHalfWidth-u)*math.Sqrt2)
	}
	if u >= 11.75 && u <= 12.15 {
		best = math.Max(best, 1.3)
	}
	if u > 12.15 && u <= 12.85 {
		best = math.Max(best, 1.3-(u-12.15))
	}
	if u <= 14.25 {
		best = math.Max(best, JournalRadius)
	}
	return best
}
