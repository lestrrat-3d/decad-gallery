package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

type Params struct {
	Scale           float64
	WheelAngle      float64
	DriveTension    float64
	BrakeTravel     float64
	YarnFeedPerTurn float64
	Ratio           int
}

func DefaultParams() Params {
	return Params{Scale: 1, WheelAngle: math.Pi / 3, DriveTension: 6, BrakeTravel: 3,
		YarnFeedPerTurn: 100, Ratio: 7}
}

func (p Params) Validate() error {
	for _, field := range []struct {
		name  string
		value float64
	}{
		{"Scale", p.Scale}, {"WheelAngle", p.WheelAngle}, {"DriveTension", p.DriveTension},
		{"BrakeTravel", p.BrakeTravel}, {"YarnFeedPerTurn", p.YarnFeedPerTurn},
	} {
		if math.IsNaN(field.value) || math.IsInf(field.value, 0) {
			return fmt.Errorf("spinningwheel: %s must be finite", field.name)
		}
	}
	if p.Scale < 0.8 || p.Scale > 1.2 {
		return fmt.Errorf("spinningwheel: Scale must be within [0.8,1.2]")
	}
	if p.DriveTension < 0 || p.DriveTension > 12 {
		return fmt.Errorf("spinningwheel: DriveTension must be within [0,12]")
	}
	if p.BrakeTravel < 0 || p.BrakeTravel > 6 {
		return fmt.Errorf("spinningwheel: BrakeTravel must be within [0,6]")
	}
	if p.YarnFeedPerTurn < 0 || p.YarnFeedPerTurn > 200 {
		return fmt.Errorf("spinningwheel: YarnFeedPerTurn must be within [0,200]")
	}
	switch p.Ratio {
	case 7, 9, 12, 17:
	default:
		return fmt.Errorf("spinningwheel: Ratio must be 7, 9, 12 or 17")
	}
	if math.IsInf(float64(p.Ratio)*p.WheelAngle, 0) {
		return fmt.Errorf("spinningwheel: Ratio times WheelAngle must be finite")
	}
	return nil
}

type State struct {
	Wheel, Flyer, Heel, CrankPin, Toe r3.Vec
	PedalAngle, RodAngle              float64
	FlyerAngle, BobbinAngle           float64
	WheelPitch, FlyerPitch            float64
	BeltLength, BrakeLength           float64
	SpringLength, HeadShift           float64
}

func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	s := p.Scale
	a := math.Remainder(p.WheelAngle, 2*math.Pi)
	cx, cz := 220+25*math.Cos(a), 430+25*math.Sin(a)
	dx, dz := cx-80, cz-60
	d := math.Hypot(dx, dz)
	cosine := (d*d + 200*200 - 370*370) / (2 * d * 200)
	if cosine < -1 || cosine > 1 {
		return State{}, fmt.Errorf("spinningwheel: treadle closure cosine %g is outside [-1,1]", cosine)
	}
	psi := math.Atan2(dz, dx) - math.Acos(cosine)
	state := State{
		Wheel:        r3.Vec{X: 220 * s, Z: 430 * s},
		Flyer:        r3.Vec{X: (-240 - p.DriveTension) * s, Z: 690 * s},
		Heel:         r3.Vec{X: 80 * s, Y: -105 * s, Z: 60 * s},
		CrankPin:     r3.Vec{X: cx * s, Y: -105 * s, Z: cz * s},
		Toe:          r3.Vec{X: (80 + 200*math.Cos(psi)) * s, Y: -105 * s, Z: (60 + 200*math.Sin(psi)) * s},
		PedalAngle:   psi,
		FlyerAngle:   float64(p.Ratio) * p.WheelAngle,
		BobbinAngle:  (float64(p.Ratio) - p.YarnFeedPerTurn/(2*math.Pi*31*s)) * p.WheelAngle,
		WheelPitch:   274*s + math.Sqrt(5),
		SpringLength: 25*s + p.BrakeTravel,
	}
	state.RodAngle = math.Atan2(state.CrankPin.Z-state.Toe.Z, state.CrankPin.X-state.Toe.X)
	state.FlyerPitch = state.WheelPitch / float64(p.Ratio)
	switch p.Ratio {
	case 7:
		state.HeadShift = 24 * s
	case 9:
		state.HeadShift = 16 * s
	case 12:
		state.HeadShift = 8 * s
	}
	distance := math.Hypot(state.Flyer.X-state.Wheel.X, state.Flyer.Z-state.Wheel.Z)
	difference := state.WheelPitch - state.FlyerPitch
	q := difference / distance
	wrap := 2 * math.Acos(q)
	state.BeltLength = state.WheelPitch*(2*math.Pi-wrap) + state.FlyerPitch*wrap +
		2*math.Sqrt(distance*distance-difference*difference)
	state.BrakeLength = math.Pi*(42*s+0.6*math.Sqrt(5)) + 105*s
	return state, nil
}
