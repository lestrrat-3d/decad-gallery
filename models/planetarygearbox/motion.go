package planetarygearbox

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// DriveMode says which shaft of the gearbox turns by the input angle. The ring is fixed in both modes.
type DriveMode int

const (
	// SunDriven turns the input shaft, and with it the first sun, by the input angle. Each stage reduces the speed
	// by Zs/(Zs+Zr), so the last carrier, which carries the output shaft, turns the slowest.
	SunDriven DriveMode = iota
	// CarrierDriven turns the last carrier and its output shaft by the input angle, as a crank on the carrier does.
	// Each stage steps the speed up by 1 + Zr/Zs, so the first sun and the input shaft, which then carry the
	// rotor, turn the fastest.
	CarrierDriven
)

// Motion is the kinematics of a gearbox: how far each moving body turns when the input turns. The ring, the covers,
// their bearings, the bolts and the nuts are fixed. A Motion holds only its parameters, so it is safe to share
// between goroutines.
type Motion struct {
	mode  DriveMode
	p     Params
	frame r3.Transform
	// angle maps the input angle to the angle of each moving body about the gearbox axis, by body name.
	angle map[string]func(input float64) float64
	// spin lists the planet bodies by name with the center, in the local frame, of the planet's pin.
	spin map[string]r3.Vec
}

// NewMotion returns the motion of a gearbox built with p, driven in mode. It returns an error for invalid p or an
// unknown mode.
func NewMotion(p Params, mode DriveMode) (*Motion, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("planetarygearbox: motion: %w", err)
	}
	if mode != SunDriven && mode != CarrierDriven {
		return nil, errors.New("planetarygearbox: motion: unknown drive mode")
	}
	frame, err := frameTransform(p)
	if err != nil {
		return nil, fmt.Errorf("planetarygearbox: motion: %w", err)
	}
	m := &Motion{
		mode: mode, p: p, frame: frame,
		angle: make(map[string]func(float64) float64),
		spin:  make(map[string]r3.Vec),
	}
	ratio := float64(p.SunTeeth) / float64(p.SunTeeth+p.RingTeeth)
	step := 1 / ratio
	orbit := p.Module * float64(p.SunTeeth+p.PlanetTeeth) / 2
	var sun, carrier func(stage int) func(float64) float64
	if mode == SunDriven {
		sun = func(k int) func(float64) float64 {
			return func(input float64) float64 { return input * math.Pow(ratio, float64(k)) }
		}
		carrier = func(k int) func(float64) float64 {
			return func(input float64) float64 { return input * math.Pow(ratio, float64(k)) * ratio }
		}
	} else {
		// The last carrier turns by the input. A stage's sun turns 1 + Zr/Zs times as fast as its carrier, and a
		// link shaft joins that sun to the carrier of the stage before it.
		sun = func(k int) func(float64) float64 {
			return func(input float64) float64 { return input * math.Pow(step, float64(p.Stages-k)) }
		}
		carrier = func(k int) func(float64) float64 {
			return func(input float64) float64 { return input * math.Pow(step, float64(p.Stages-1-k)) }
		}
	}
	m.angle[InputShaftName] = sun(0)
	for k := range p.Stages {
		m.angle[SunName(k)] = sun(k)
		m.angle[CarrierName(k)] = carrier(k)
		m.angle[ShaftName(k, p.Stages)] = carrier(k)
		for i := range p.PlanetCount {
			phi := PlanetOrbitAngle(p, i)
			m.angle[PinName(k, i)] = carrier(k)
			m.angle[PlanetName(k, i)] = carrier(k)
			m.angle[BearingName(k, i)] = carrier(k)
			center := r3.NewVec(orbit*math.Cos(phi), orbit*math.Sin(phi), 0)
			m.spin[PlanetName(k, i)] = center
			m.spin[BearingName(k, i)] = center
		}
	}
	return m, nil
}

// Mode returns the drive mode.
func (m *Motion) Mode() DriveMode { return m.mode }

// Ratio returns how many turns the output makes per input turn. Driven by the sun it is the reduction
// (Zs/(Zs+Zr))^stages, below 1: the last carrier is the output. Driven by the carrier it is the step-up
// (1 + Zr/Zs)^stages, above 1: the first sun and the input shaft are the output.
func (m *Motion) Ratio() float64 { return m.OutputAngle(1) }

// OutputAngle returns the angle of the output shaft for an input angle: the last carrier's when SunDriven, the
// input shaft's when CarrierDriven. The input shaft, not a planet, is where a rotor or a crank mounts.
func (m *Motion) OutputAngle(input float64) float64 {
	ratio := float64(m.p.SunTeeth) / float64(m.p.SunTeeth+m.p.RingTeeth)
	if m.mode == SunDriven {
		return input * math.Pow(ratio, float64(m.p.Stages))
	}
	return m.angle[InputShaftName](input)
}

// Transform returns the rigid motion of the named body, in its built position, for an input angle in radians. The
// names are the ones Gearbox.Parts gives. A planet and its bearing are carried around by their carrier and turn on
// their pin by -Zr/Zp of the carrier's angle, because they roll inside the fixed ring. The boolean is false for a
// fixed body, whose transform is the identity. A motion of a gearbox built with a placement is in the placed frame.
func (m *Motion) Transform(name string, input float64) (r3.Transform, bool, error) {
	angle, ok := m.angle[name]
	if !ok {
		return r3.Identity(), false, nil
	}
	axis := r3.NewVec(0, 0, 1)
	about := func(a float64) (r3.Transform, error) { return r3.Rotation(axis, units.Radians(a)) }
	var t r3.Transform
	var err error
	if center, planet := m.spin[name]; planet {
		carried := angle(input)
		spin, err := r3.RotationAround(center, axis,
			units.Radians(-float64(m.p.RingTeeth)/float64(m.p.PlanetTeeth)*carried))
		if err != nil {
			return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
		}
		around, err := about(carried)
		if err != nil {
			return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
		}
		if t, err = spin.Then(around); err != nil {
			return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
		}
	} else if t, err = about(angle(input)); err != nil {
		return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
	}
	if m.p.Placement == nil {
		return t, true, nil
	}
	// The axis and centers above are in the local frame: turn there, then move onto the placement. The built body
	// is already placed, so conjugate: undo the frame, turn, redo it.
	inverse, err := m.frame.Inverse()
	if err != nil {
		return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
	}
	placed, err := inverse.Then(t)
	if err != nil {
		return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
	}
	if placed, err = placed.Then(m.frame); err != nil {
		return r3.Transform{}, false, fmt.Errorf("planetarygearbox: motion of %s: %w", name, err)
	}
	return placed, true, nil
}
