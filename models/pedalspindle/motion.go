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
	if math.IsNaN(p.Scale) || math.IsInf(p.Scale, 0) || p.Scale < 0.8 || p.Scale > 1.2 {
		return fmt.Errorf("scale must be finite and between 0.8 and 1.2")
	}
	if math.IsNaN(p.CrankAngle) || math.IsInf(p.CrankAngle, 0) {
		return fmt.Errorf("crank angle must be finite")
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
	px, pz := -90+25*math.Cos(a), 380+25*math.Sin(a)
	dx, dz := px+230, pz-60
	d := math.Hypot(dx, dz)
	cosine := (d*d + 200*200 - 320*320) / (2 * d * 200)
	if d == 0 || cosine < -1 || cosine > 1 {
		return State{}, fmt.Errorf("four-bar linkage does not close")
	}
	psi := math.Atan2(dz, dx) - math.Acos(cosine)
	b := math.Pi + math.Pi/17 - 3*a
	c := 3*math.Pi + math.Pi/17 - 3*b
	s := p.Scale
	point := func(x, y, z float64) r3.Vec { return r3.NewVec(x*s, y*s, z*s) }
	return State{
		Input: point(-90, 0, 380), Intermediate: point(12.3, 0, 380), Output: point(12.3, 0, 482.3),
		Heel: point(-230, -125, 60), CrankPin: point(px, -125, pz),
		Toe:        point(-230+200*math.Cos(psi), -125, 60+200*math.Sin(psi)),
		PedalAngle: psi, IntermediateAngle: b, SpindleAngle: c, Ratio: 9, CenterDistance: 102.3 * s,
	}, nil
}
