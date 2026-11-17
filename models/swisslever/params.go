// Package swisslever builds a desk-scale Swiss lever escapement as a parametric decad model: a 15-tooth club-tooth
// escape wheel, a lever with two pressed-in pallet stones, a balance with a roller table, an impulse pin, a safety
// roller and a printed hairspring, in a two-plate frame on six 605ZZ bearings. A knob on the escape arbor drives it.
package swisslever

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

const deg = math.Pi / 180

// Params selects the movement. Every other dimension is a fixed constant of instructions.md.
type Params struct {
	// Amplitude is the balance amplitude in radians, 45° to 270°.
	Amplitude float64
	// Phase is the pose as a fraction of one balance period, 0 up to but not including 1.
	Phase float64
	// Placement maps the model frame into the world. Nil is the identity.
	Placement *r3.Frame
}

// DefaultParams returns the default movement: a 180° balance swing, posed at Phase 0.25 where the balance is at rest.
func DefaultParams() Params {
	return Params{Amplitude: 180 * deg, Phase: 0.25}
}

// Validate rejects a nonfinite value, a value outside its range and an invalid Placement. The error names the field.
func (p Params) Validate() error {
	for _, f := range []struct {
		name     string
		v        float64
		lo, hi   float64
		openHigh bool
	}{
		{"Amplitude", p.Amplitude, 45 * deg, 270 * deg, false},
		{"Phase", p.Phase, 0, 1, true},
	} {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) {
			return fmt.Errorf("swisslever: %s must be finite, got %v", f.name, f.v)
		}
		// A small slack keeps the exact range ends valid despite rounding.
		if f.v < f.lo-1e-12 || f.v > f.hi+1e-12 || (f.openHigh && f.v >= f.hi) {
			return fmt.Errorf("swisslever: %s %v is outside its range [%v, %v]", f.name, f.v, f.lo, f.hi)
		}
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("swisslever: Placement must be a valid frame")
	}
	return nil
}
