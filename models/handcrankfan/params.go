// Package handcrankfan builds a handheld fan turned by a right-hand crank through a bevel gear pair.
package handcrankfan

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Ratio is the fan turns per crank turn.
const Ratio = 2

// Params selects the crank pose and the rotor. Angles are radians.
type Params struct {
	// CrankAngle is the forward (counterclockwise seen from the right) turn from the arm pointing up.
	CrankAngle float64
	// BladePitch turns each whole blade about its radial line; it is the blade angle at 48 mm and lies in [10°, 40°].
	BladePitch float64
	// Blades is 3 or 4.
	Blades int
}

// DefaultParams returns the arm pointing down and toward the operator, a 25° pitch and four blades.
func DefaultParams() Params {
	return Params{CrankAngle: 3 * math.Pi / 4, BladePitch: 25 * math.Pi / 180, Blades: 4}
}

// Validate rejects a non-finite angle, a pitch outside [10°, 40°] and a blade count other than 3 or 4.
func (p Params) Validate() error {
	if !finite(p.CrankAngle, Ratio*p.CrankAngle) {
		return fmt.Errorf("handcrankfan: CrankAngle must be finite")
	}
	if !finite(p.BladePitch) || p.BladePitch < 10*math.Pi/180 || p.BladePitch > 40*math.Pi/180 {
		return fmt.Errorf("handcrankfan: BladePitch must lie in [10°, 40°] radians")
	}
	if p.Blades != 3 && p.Blades != 4 {
		return fmt.Errorf("handcrankfan: Blades must be 3 or 4")
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

// State is the pose of the mechanism.
type State struct {
	CrankAngle, FanAngle float64
	// Apex is where the drive shaft's axis meets the fan axis.
	Apex r3.Vec
	// Crank is a point on the crank shaft's axis, directly below the apex.
	Crank r3.Vec
	// ArmDirection is the unit vector from the crank axis along the arm.
	ArmDirection r3.Vec
}

// Motion computes the pose without building bodies.
func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	return State{
		CrankAngle:   p.CrankAngle,
		FanAngle:     Ratio * p.CrankAngle,
		Apex:         r3.Vec{Z: axisHeight},
		Crank:        r3.Vec{Z: crankHeight},
		ArmDirection: r3.Vec{Y: -math.Sin(p.CrankAngle), Z: math.Cos(p.CrankAngle)},
	}, nil
}
