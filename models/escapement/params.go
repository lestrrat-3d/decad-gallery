// Package escapement builds a Graham deadbeat escapement clock movement as a parametric decad model: a weight
// drive, an escape wheel, an anchor, a crutch and a pendulum in a two-plate frame on seven 605ZZ bearings.
package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

const deg = math.Pi / 180

// Params selects the movement. Every other dimension is a fixed constant of instructions.md.
type Params struct {
	// PendulumLength is the distance in mm from the suspension to the bob centre, 120 to 250.
	PendulumLength float64
	// Amplitude is the anchor's swing in radians from its centre, (LiftAngle+LockAngle)/2 + 1° to 6°.
	Amplitude float64
	// LockAngle is the lock angle κ in radians, 1.25° to 1.5°.
	LockAngle float64
	// LiftAngle is the lift (impulse) angle λ in radians, 2° to 3°.
	LiftAngle float64
	// Phase is the pose as a fraction of one pendulum period, 0 up to but not including 1.
	Phase float64
	// Backlash is the gear tooth thinning in mm, 0 to 0.3.
	Backlash float64
	// Placement maps the model frame into the world. Nil is the identity.
	Placement *r3.Frame
}

// DefaultParams returns the default movement: a 180 mm pendulum, a 5° anchor swing, 1.5° of lock and 2.5° of lift.
func DefaultParams() Params {
	return Params{
		PendulumLength: 180,
		Amplitude:      5 * deg,
		LockAngle:      1.5 * deg,
		LiftAngle:      2.5 * deg,
		Backlash:       0.15,
	}
}

// Validate rejects a nonfinite value, a value outside its range and an invalid Placement. The error names the field.
func (p Params) Validate() error {
	for _, f := range []struct {
		name  string
		v     float64
		lo    float64
		hi    float64
		openH bool
	}{
		{"PendulumLength", p.PendulumLength, 120, 250, false},
		{"LockAngle", p.LockAngle, 1.25 * deg, 1.5 * deg, false},
		{"LiftAngle", p.LiftAngle, 2 * deg, 3 * deg, false},
		{"Phase", p.Phase, 0, 1, true},
		{"Backlash", p.Backlash, 0, 0.3, false},
	} {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) {
			return fmt.Errorf("escapement: %s must be finite, got %v", f.name, f.v)
		}
		if f.v < f.lo || f.v > f.hi || (f.openH && f.v >= f.hi) {
			return fmt.Errorf("escapement: %s %v is outside its range [%v, %v]", f.name, f.v, f.lo, f.hi)
		}
	}
	if math.IsNaN(p.Amplitude) || math.IsInf(p.Amplitude, 0) {
		return fmt.Errorf("escapement: Amplitude must be finite, got %v", p.Amplitude)
	}
	lo := (p.LiftAngle+p.LockAngle)/2 + deg
	// A small slack keeps the exact range ends valid despite rounding.
	if p.Amplitude < lo-1e-12 || p.Amplitude > 6*deg+1e-12 {
		return fmt.Errorf("escapement: Amplitude %v is outside its range [%v, %v]", p.Amplitude, lo, 6*deg)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("escapement: Placement must be a valid frame")
	}
	return nil
}
