// Package handcranksiren builds a hand-cranked siren: a crank turns a two-stage spur step-up train, and the train spins
// a slotted rotor disc in front of a fixed slotted stator disc inside a drum with an outlet horn.
package handcranksiren

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Ratio is the rotor turns per crank turn: (50/16)·(48/15).
const Ratio = 10.0

// Slot counts the model accepts.
const (
	minSlots = 12
	maxSlots = 24
)

// Layout of the three shaft axes, in mm. R, I and C run along Y; the points below are their positions at Y 0.
const (
	rotorHeight = 84.0  // height of R and C above the table
	crankOffset = 60.0  // C.x
	stage1Dist  = 49.5  // I–C: (50 + 16) × 1.5 / 2
	stage2Dist  = 47.25 // R–I: (48 + 15) × 1.5 / 2
)

// Params selects the crank pose and the number of ports.
type Params struct {
	// CrankAngle is the forward turn, clockwise seen from behind (about −Y), from the arm pointing up. Radians.
	CrankAngle float64
	// Slots is the number of ports in the rotor and in the stator, and of vanes on the rotor; it lies in [12, 24].
	Slots int
}

// DefaultParams returns the arm pointing down and toward the operator, with 20 slots.
func DefaultParams() Params { return Params{CrankAngle: 5 * math.Pi / 4, Slots: 20} }

// Validate rejects a non-finite CrankAngle, one whose rotor angle is not finite, and a Slots outside [12, 24].
func (p Params) Validate() error {
	if !finite(p.CrankAngle, Ratio*p.CrankAngle) {
		return fmt.Errorf("handcranksiren: CrankAngle must be finite")
	}
	if p.Slots < minSlots || p.Slots > maxSlots {
		return fmt.Errorf("handcranksiren: Slots must lie in [%d, %d]", minSlots, maxSlots)
	}
	return nil
}

func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// PitchHz is the tone, in Hz, of a siren with slots ports whose crank turns crankRevPerSecond times a second.
func PitchHz(slots int, crankRevPerSecond float64) float64 {
	return float64(slots) * Ratio * crankRevPerSecond
}

// State is the pose of the mechanism.
type State struct {
	CrankAngle, IntermediateAngle, RotorAngle float64
	// Rotor, Intermediate and Crank are points on the three axes at Y 0.
	Rotor, Intermediate, Crank r3.Vec
	// ArmDirection is the unit vector from the crank axis along the arm.
	ArmDirection r3.Vec
}

// Motion computes the pose without building bodies.
func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	ix := (crankOffset*crankOffset + stage2Dist*stage2Dist - stage1Dist*stage1Dist) / (2 * crankOffset)
	iz := rotorHeight - math.Sqrt(stage2Dist*stage2Dist-ix*ix)
	return State{
		CrankAngle:        p.CrankAngle,
		IntermediateAngle: -(50.0 / 16.0) * p.CrankAngle,
		RotorAngle:        Ratio * p.CrankAngle,
		Rotor:             r3.Vec{Z: rotorHeight},
		Intermediate:      r3.Vec{X: ix, Z: iz},
		Crank:             r3.Vec{X: crankOffset, Z: rotorHeight},
		ArmDirection:      r3.Vec{X: -math.Sin(p.CrankAngle), Z: math.Cos(p.CrankAngle)},
	}, nil
}
