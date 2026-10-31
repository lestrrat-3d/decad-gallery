package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

type Params struct {
	Scale, WheelAngle, DriveTension, BrakeTravel, YarnFeedPerTurn float64
	Ratio                                                         int
}

type State struct {
	Wheel, Flyer, Heel, CrankPin, Toe, RodHead, HeadPivot                    r3.Vec
	PedalAngle, RodAngle, FlyerAngle, BobbinAngle, HeadTilt                  float64
	WheelPitch, FlyerPitch, BeltLength, BrakeLength, SpringLength, HeadShift float64
	PedalFrame, RodFrame, GimbalFrame                                        r3.Frame
}

func DefaultParams() Params {
	return Params{Scale: 1, WheelAngle: math.Pi / 3, DriveTension: 6, BrakeTravel: 3,
		YarnFeedPerTurn: 100, Ratio: 7}
}

func motionFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func (p Params) Validate() error {
	for _, limit := range []struct {
		name             string
		value, low, high float64
	}{
		{"Scale", p.Scale, .8, 1.2},
		{"DriveTension", p.DriveTension, 0, 12},
		{"BrakeTravel", p.BrakeTravel, 0, 6},
		{"YarnFeedPerTurn", p.YarnFeedPerTurn, 0, 200},
	} {
		if !motionFinite(limit.value) || limit.value < limit.low || limit.value > limit.high {
			return fmt.Errorf("invalid %s: require a finite value in [%g,%g]", limit.name, limit.low, limit.high)
		}
	}
	if !motionFinite(p.WheelAngle) {
		return fmt.Errorf("invalid WheelAngle: require finite radians")
	}
	switch p.Ratio {
	case 7, 9, 12, 17:
	default:
		return fmt.Errorf("invalid Ratio: require 7, 9, 12 or 17")
	}
	if !motionFinite(float64(p.Ratio) * p.WheelAngle) {
		return fmt.Errorf("invalid flyer phase: Ratio times WheelAngle must be finite")
	}
	return nil
}

func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, err
	}
	s := p.Scale
	a := math.Remainder(p.WheelAngle, 2*math.Pi)
	state := State{
		Wheel:        r3.Vec{X: 220 * s, Z: 580 * s},
		Heel:         r3.Vec{X: 60 * s, Y: -190 * s, Z: 60 * s},
		CrankPin:     r3.Vec{X: (220 + 25*math.Cos(a)) * s, Y: 105 * s, Z: (580 + 25*math.Sin(a)) * s},
		HeadPivot:    r3.Vec{X: -172 * s, Z: 555 * s},
		HeadTilt:     math.Asin((p.DriveTension - 6) / 122),
		FlyerAngle:   float64(p.Ratio) * p.WheelAngle,
		BobbinAngle:  (float64(p.Ratio) - p.YarnFeedPerTurn/(2*math.Pi*31*s)) * p.WheelAngle,
		WheelPitch:   274*s + math.Sqrt(5),
		BrakeLength:  math.Pi*(42*s+.6*math.Sqrt(5)) + 612*s,
		SpringLength: 25*s + p.BrakeTravel/2,
	}
	for i, ratio := range []int{7, 9, 12, 17} {
		if p.Ratio == ratio {
			state.HeadShift = float64(278-8*i) * s
			break
		}
	}
	tilt, err := r3.RotationAround(state.HeadPivot, r3.Vec{Y: -1}, units.Radians(state.HeadTilt))
	if err != nil {
		return State{}, fmt.Errorf("head tilt: %w", err)
	}
	state.Flyer = tilt.Apply(r3.Vec{X: -140 * s, Z: 690 * s})
	state.FlyerPitch = state.WheelPitch / float64(p.Ratio)
	distance := state.Flyer.Sub(state.Wheel).Len()
	difference := state.WheelPitch - state.FlyerPitch
	q := difference / distance
	if !motionFinite(q) || q <= -1 || q >= 1 {
		return State{}, fmt.Errorf("drive pitch circles have no external tangent")
	}
	alpha := math.Acos(q)
	state.BeltLength = state.WheelPitch*(2*math.Pi-2*alpha) + state.FlyerPitch*2*alpha +
		2*math.Sqrt(distance*distance-difference*difference)

	closure := func(psi float64) (float64, r3.Frame, r3.Vec, r3.Vec, r3.Vec, error) {
		pedal, err := r3.NewFrame(r3.Vec{Y: -190 * s, Z: 60 * s}, r3.Vec{X: 1},
			r3.Vec{Y: math.Cos(psi), Z: math.Sin(psi)})
		if err != nil {
			return 0, r3.Frame{}, r3.Vec{}, r3.Vec{}, r3.Vec{}, err
		}
		toe := pedal.ToWorld(r3.Vec{X: 220 * s, Y: 450 * s, Z: 20 * s})
		w, ok := (r3.Vec{X: toe.X - state.CrankPin.X, Z: toe.Z - state.CrankPin.Z}).Normalize()
		if !ok {
			return 0, r3.Frame{}, r3.Vec{}, r3.Vec{}, r3.Vec{}, fmt.Errorf("degenerate projected footman direction")
		}
		head := state.CrankPin.Add(w.Scale(32 * s))
		return toe.Sub(head).Len() - 486*s, pedal, toe, head, w, nil
	}
	lo, hi := -.12, .20
	left, _, _, _, _, err := closure(lo)
	if err != nil {
		return State{}, fmt.Errorf("lower linkage endpoint: %w", err)
	}
	right, _, _, _, _, err := closure(hi)
	if err != nil {
		return State{}, fmt.Errorf("upper linkage endpoint: %w", err)
	}
	if !motionFinite(left) || !motionFinite(right) || left*right >= 0 {
		return State{}, fmt.Errorf("linkage endpoints do not bracket a continuous solution")
	}
	for range 80 {
		mid := (lo + hi) / 2
		value, _, _, _, _, err := closure(mid)
		if err != nil {
			return State{}, fmt.Errorf("linkage bisection: %w", err)
		}
		if !motionFinite(value) {
			return State{}, fmt.Errorf("non-finite linkage residual")
		}
		if left*value > 0 {
			lo, left = mid, value
		} else {
			hi = mid
		}
	}
	state.PedalAngle = (lo + hi) / 2
	residual, pedal, toe, head, w, err := closure(state.PedalAngle)
	if err != nil {
		return State{}, fmt.Errorf("final linkage pose: %w", err)
	}
	if !motionFinite(residual) || math.Abs(residual) >= 1e-8*s {
		return State{}, fmt.Errorf("linkage closure residual %g exceeds tolerance", residual)
	}
	state.PedalFrame, state.Toe, state.RodHead = pedal, toe, head
	v := r3.Vec{X: -w.Z, Z: w.X}
	state.GimbalFrame, err = r3.NewFrame(state.CrankPin, w, v)
	if err != nil {
		return State{}, fmt.Errorf("gimbal frame: %w", err)
	}
	u, ok := head.Sub(toe).Normalize()
	if !ok {
		return State{}, fmt.Errorf("degenerate footman direction")
	}
	state.RodFrame, err = r3.NewFrame(toe, u, v)
	if err != nil {
		return State{}, fmt.Errorf("footman frame: %w", err)
	}
	state.RodAngle = math.Atan2(head.Z-toe.Z, head.X-toe.X)
	return state, nil
}

func PoseTransform(base, state State, motion string, wheelDelta float64) (r3.Transform, error) {
	var from, to r3.Frame
	switch motion {
	case "fixed":
		return r3.Identity(), nil
	case "wheel":
		return r3.RotationAround(base.Wheel, r3.Vec{Y: -1}, units.Radians(math.Remainder(wheelDelta, 2*math.Pi)))
	case "flyer":
		return r3.RotationAround(base.Flyer, r3.Vec{Y: -1}, units.Radians(math.Remainder(state.FlyerAngle-base.FlyerAngle, 2*math.Pi)))
	case "bobbin":
		return r3.RotationAround(base.Flyer, r3.Vec{Y: -1}, units.Radians(math.Remainder(state.BobbinAngle-base.BobbinAngle, 2*math.Pi)))
	case "treadle":
		from, to = base.PedalFrame, state.PedalFrame
	case "gimbal":
		from, to = base.GimbalFrame, state.GimbalFrame
	case "footman":
		from, to = base.RodFrame, state.RodFrame
	default:
		return r3.Transform{}, fmt.Errorf("unknown motion group %q", motion)
	}
	origin, err := r3.FromFrame(from)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("base %s frame: %w", motion, err)
	}
	inverse, err := origin.Inverse()
	if err != nil {
		return r3.Transform{}, fmt.Errorf("invert base %s frame: %w", motion, err)
	}
	target, err := r3.FromFrame(to)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("target %s frame: %w", motion, err)
	}
	return inverse.Then(target)
}
