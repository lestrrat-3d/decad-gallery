// Package pedalspindle constructs a treadle-driven quill spindle assembly.
package pedalspindle

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Params sets the uniform size and crank position.
type Params struct {
	Scale      float64
	CrankAngle float64
}

// DefaultParams returns full size with the crank at sixty degrees.
func DefaultParams() Params {
	return Params{Scale: 1, CrankAngle: math.Pi / 3}
}

// Validate rejects nonfinite parameters and scales outside [0.8, 1.2].
func (p Params) Validate() error {
	if math.IsNaN(p.Scale) || math.IsInf(p.Scale, 0) || p.Scale < 0.8 || p.Scale > 1.2 {
		return fmt.Errorf("pedalspindle: scale must be finite and in [0.8, 1.2]")
	}
	if math.IsNaN(p.CrankAngle) || math.IsInf(p.CrankAngle, 0) {
		return fmt.Errorf("pedalspindle: crank angle must be finite")
	}
	return nil
}

// State records scaled shaft and linkage coordinates and the drive angles.
type State struct {
	Input, Intermediate, Output                 r3.Vec
	Heel, CrankPin, Toe                         r3.Vec
	PedalAngle, IntermediateAngle, SpindleAngle float64
	Ratio, CenterDistance                       float64
}

// Motion solves the lower four-bar intersection for the requested crank angle.
func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	a := math.Remainder(p.CrankAngle, 2*math.Pi)
	px, pz := -90+25*math.Cos(a), 380+25*math.Sin(a)
	d := math.Hypot(px+230, pz-60)
	arg := (d*d + 200*200 - 320*320) / (2 * d * 200)
	if arg < -1 || arg > 1 || math.IsNaN(arg) {
		return State{}, fmt.Errorf("pedalspindle: four-bar linkage does not close")
	}
	psi := math.Atan2(pz-60, px+230) - math.Acos(arg)
	b := math.Pi + math.Pi/17 - 3*a
	c := 3*math.Pi + math.Pi/17 - 3*b
	point := func(x, y, z float64) r3.Vec { return r3.NewVec(x*p.Scale, y*p.Scale, z*p.Scale) }
	return State{
		Input: point(-90, 0, 380), Intermediate: point(12.3, 0, 380), Output: point(12.3, 0, 482.3),
		Heel: point(-230, -101, 60), CrankPin: point(px, -101, pz),
		Toe:        point(-230+200*math.Cos(psi), -101, 60+200*math.Sin(psi)),
		PedalAngle: psi, IntermediateAngle: b, SpindleAngle: c,
		Ratio: 9, CenterDistance: 102.3 * p.Scale,
	}, nil
}
