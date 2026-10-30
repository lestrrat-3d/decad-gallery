package pedalspindle

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

type Params struct {
	Scale      float64
	CrankAngle float64
}

func DefaultParams() Params { return Params{Scale: 1, CrankAngle: math.Pi / 3} }

func (p Params) Validate() error {
	if math.IsNaN(p.Scale) || math.IsInf(p.Scale, 0) || p.Scale < .8 || p.Scale > 1.2 {
		return fmt.Errorf("pedalspindle: Scale must be finite and within [0.8, 1.2]")
	}
	if math.IsNaN(p.CrankAngle) || math.IsInf(p.CrankAngle, 0) {
		return fmt.Errorf("pedalspindle: CrankAngle must be finite")
	}
	return nil
}

type State struct {
	Input, Intermediate, Output                 r3.Vec
	Heel, CrankPin, Toe                         r3.Vec
	PedalAngle, IntermediateAngle, SpindleAngle float64
	Ratio, CenterDistance                       float64
}

func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	b := math.Pi + math.Pi/17 - 3*a
	c := 3*math.Pi + math.Pi/17 - 3*b
	px, pz := -90+25*math.Cos(a), 380+25*math.Sin(a)
	dx, dz := px+230, pz-60
	d := math.Hypot(dx, dz)
	cosine := (d*d + 200*200 - 320*320) / (2 * d * 200)
	if cosine < -1 || cosine > 1 {
		return State{}, fmt.Errorf("pedalspindle: linkage does not close")
	}
	psi := math.Atan2(dz, dx) - math.Acos(cosine)
	s := p.Scale
	return State{
		Input: r3.NewVec(-90*s, 0, 380*s), Intermediate: r3.NewVec(12.3*s, 0, 380*s),
		Output: r3.NewVec(12.3*s, 0, 482.3*s), Heel: r3.NewVec(-230*s, -125*s, 60*s),
		CrankPin:   r3.NewVec(px*s, -125*s, pz*s),
		Toe:        r3.NewVec((-230+200*math.Cos(psi))*s, -125*s, (60+200*math.Sin(psi))*s),
		PedalAngle: psi, IntermediateAngle: b, SpindleAngle: c, Ratio: 9, CenterDistance: 102.3 * s,
	}, nil
}
