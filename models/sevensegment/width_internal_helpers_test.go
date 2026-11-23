package sevensegment

import "math"

// rotorEndPlay is how far a rotor can move along its axis: the bore's end wall stands 0.25 from the rotor's end face.
const rotorEndPlay = 0.25

// rotorSettledZ is the rotor axis height settled in its half-holes (instructions.md "Z stack": 82.1 against 82.6).
const rotorSettledZ = RotorAxisZ - 0.5

// rotorSweptRadius returns an upper bound on the radius an a, g or d rotor sweeps at distance u (u >= 0) from its
// middle along its axis: the rounded square's (RotorHalfWidth-rotorEdgeRound)*√2+rotorEdgeRound, capped by the pyramid's
// (rotorSquareHalf+RotorHalfWidth-u)*√2, for u <= RotorHalfLength, and 0 beyond it: the rotor has no stubs, and its end
// bore (r 0.6) is a void.
func rotorSweptRadius(u float64) float64 {
	u = math.Max(u, 0)
	if u > RotorHalfLength {
		return 0
	}
	square := (RotorHalfWidth-rotorEdgeRound)*math.Sqrt2 + rotorEdgeRound
	return math.Min(square, (rotorSquareHalf+RotorHalfWidth-u)*math.Sqrt2)
}
