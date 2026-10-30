// Package pedalspindle constructs a treadle-powered quill spindle assembly.
package pedalspindle

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Params selects frame scale and crank pose in radians.
type Params struct {
	Scale      float64
	CrankAngle float64
}

// DefaultParams returns nominal size at a pi/3 crank angle.
func DefaultParams() Params { return Params{Scale: 1, CrankAngle: math.Pi / 3} }

// Validate rejects nonfinite inputs and scales outside [0.8,1.2].
func (p Params) Validate() error {
	if math.IsNaN(p.Scale) || math.IsInf(p.Scale, 0) || p.Scale < 0.8 || p.Scale > 1.2 {
		return fmt.Errorf("pedalspindle: scale %g must be finite and in [0.8,1.2]", p.Scale)
	}
	if math.IsNaN(p.CrankAngle) || math.IsInf(p.CrankAngle, 0) {
		return fmt.Errorf("pedalspindle: crank angle %g must be finite", p.CrankAngle)
	}
	return nil
}

// State contains scaled shaft/linkage centers and the solved angular pose.
type State struct {
	Input, Intermediate, Output                 r3.Vec
	Heel, CrankPin, Toe                         r3.Vec
	PedalAngle, IntermediateAngle, SpindleAngle float64
	Ratio, CenterDistance                       float64
}

// Motion solves the lower four-bar intersection without adding geometry.
func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	px, pz := -90+25*math.Cos(a), 380+25*math.Sin(a)
	dx, dz := px+230, pz-60
	d := math.Hypot(dx, dz)
	cosine := (d*d + 200*200 - 320*320) / (2 * d * 200)
	if d == 0 || cosine < -1 || cosine > 1 || math.IsNaN(cosine) {
		return State{}, fmt.Errorf("pedalspindle: linkage does not close at crank angle %g", a)
	}
	psi := math.Atan2(dz, dx) - math.Acos(cosine)
	b := math.Pi + math.Pi/17 - 3*a
	c := 3*math.Pi + math.Pi/17 - 3*b
	s := p.Scale
	return State{
		Input:        r3.NewVec(-90*s, 0, 380*s),
		Intermediate: r3.NewVec(12.3*s, 0, 380*s),
		Output:       r3.NewVec(12.3*s, 0, 482.3*s),
		Heel:         r3.NewVec(-230*s, -125*s, 60*s),
		CrankPin:     r3.NewVec(px*s, -125*s, pz*s),
		Toe:          r3.NewVec((-230+200*math.Cos(psi))*s, -125*s, (60+200*math.Sin(psi))*s),
		PedalAngle:   psi, IntermediateAngle: b, SpindleAngle: c,
		Ratio: 9, CenterDistance: 102.3 * s,
	}, nil
}
