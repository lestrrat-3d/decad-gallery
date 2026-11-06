package planetarygearbox

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Params are the gearbox inputs. Lengths are in millimeters and angles in
// radians. Zero fields are not defaults; use DefaultParams for those.
type Params struct {
	// Module is the tooth module of all three gears.
	Module float64
	// PressureAngle is the pressure angle of all three gears.
	PressureAngle float64
	// SunTeeth, PlanetTeeth and RingTeeth are the tooth counts. RingTeeth must be
	// SunTeeth + 2*PlanetTeeth.
	SunTeeth, PlanetTeeth, RingTeeth int
	// PlanetCount is the number of planets in each stage.
	PlanetCount int
	// Stages is the number of identical stages stacked in the one ring.
	Stages int
	// Samples is the number of fit points per tooth flank.
	Samples int
	// GearThickness is the axial thickness of a stage's sun and planets.
	GearThickness float64
	// CarrierThickness is the axial thickness of a carrier plate.
	CarrierThickness float64
	// StageGap is the axial gap between a stage's gears, its carrier, the next
	// stage and the ring ends.
	StageGap float64
	// ShaftDiameter is the diameter of the sun bore, the stub shafts and the
	// output shaft.
	ShaftDiameter float64
	// PinDiameter is the diameter of a planet pin.
	PinDiameter float64
	// PinClearance is the radial clearance between a planet bore and its pin.
	PinClearance float64
	// RingWall is the radial wall of the ring behind the tooth roots.
	RingWall float64
	// Backlash is how much each tooth of the sun and the planets is thinned at the pitch
	// circle. The ring's tooth spaces are widened by the same amount, so a sun-planet pair and
	// a planet-ring pair each have twice Backlash of play.
	Backlash float64
	// OutputShaftLength is the length of the output shaft above the last carrier plate.
	OutputShaftLength float64
	// Placement is the reference point and plane. Nil selects the world XY frame
	// at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the 13/29/71 three-planet gearbox with one stage.
func DefaultParams() Params {
	return Params{
		Module:            1,
		PressureAngle:     20 * math.Pi / 180,
		SunTeeth:          13,
		PlanetTeeth:       29,
		RingTeeth:         71,
		PlanetCount:       3,
		Stages:            1,
		Samples:           15,
		GearThickness:     8,
		CarrierThickness:  3,
		StageGap:          0.5,
		ShaftDiameter:     5,
		PinDiameter:       4,
		PinClearance:      0.1,
		RingWall:          4,
		Backlash:          0.05,
		OutputShaftLength: 10,
	}
}

// Ratio returns the reduction from the first sun to the last carrier with the
// ring held fixed.
func (p Params) Ratio() float64 {
	stage := 1 + float64(p.RingTeeth)/float64(p.SunTeeth)
	return math.Pow(stage, float64(p.Stages))
}

// dims are the derived dimensions in millimeters.
type dims struct {
	orbit, planetRoot, planetTip float64
	ringTip, ringRoot, ringOuter float64
	carrier, planetBore          float64
	stagePitch, ringLength       float64
}

// derive computes the derived dimensions of p without validating p.
func derive(p Params) dims {
	m := p.Module
	d := dims{
		orbit:      m * float64(p.SunTeeth+p.PlanetTeeth) / 2,
		planetRoot: (m*float64(p.PlanetTeeth) - 2.5*m) / 2,
		planetTip:  (m*float64(p.PlanetTeeth) + 2*m) / 2,
		ringTip:    m*float64(p.RingTeeth)/2 - m,
		ringRoot:   m*float64(p.RingTeeth)/2 + 1.25*m,
		planetBore: p.PinDiameter + 2*p.PinClearance,
		stagePitch: p.GearThickness + p.CarrierThickness + 2*p.StageGap,
	}
	d.ringOuter = d.ringRoot + p.RingWall
	d.carrier = d.ringTip - m/2
	d.ringLength = float64(p.Stages)*d.stagePitch + p.StageGap
	return d
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Validate applies the input checks of the construction contract.
func (p Params) Validate() error {
	for name, v := range map[string]float64{
		"Module": p.Module, "PressureAngle": p.PressureAngle, "GearThickness": p.GearThickness,
		"CarrierThickness": p.CarrierThickness, "StageGap": p.StageGap, "ShaftDiameter": p.ShaftDiameter,
		"PinDiameter": p.PinDiameter, "PinClearance": p.PinClearance, "RingWall": p.RingWall, "Backlash": p.Backlash,
		"OutputShaftLength": p.OutputShaftLength,
	} {
		if !finite(v) {
			return fmt.Errorf("%s must be finite, got %v", name, v)
		}
	}
	for _, c := range []struct {
		name string
		v    float64
		ok   bool
	}{
		{"module", p.Module, p.Module > 0},
		{"gear thickness", p.GearThickness, p.GearThickness > 0},
		{"carrier thickness", p.CarrierThickness, p.CarrierThickness > 0},
		{"stage gap", p.StageGap, p.StageGap >= 0},
		{"shaft diameter", p.ShaftDiameter, p.ShaftDiameter > 0},
		{"pin diameter", p.PinDiameter, p.PinDiameter > 0},
		{"pin clearance", p.PinClearance, p.PinClearance >= 0},
		{"ring wall", p.RingWall, p.RingWall > 0},
		{"backlash", p.Backlash, p.Backlash >= 0 && p.Backlash <= p.Module/2},
		{"output shaft length", p.OutputShaftLength, p.OutputShaftLength > 0},
	} {
		if !c.ok {
			return fmt.Errorf("%s %v is out of range", c.name, c.v)
		}
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	}
	if p.Samples < 3 {
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	}
	if p.PlanetCount < 2 {
		return fmt.Errorf("planet count %d must be at least 2", p.PlanetCount)
	}
	if p.Stages < 1 {
		return fmt.Errorf("stages %d must be at least 1", p.Stages)
	}
	for name, teeth := range map[string]int{"sun": p.SunTeeth, "planet": p.PlanetTeeth, "ring": p.RingTeeth} {
		if teeth < 3 {
			return fmt.Errorf("%s teeth %d must be at least 3", name, teeth)
		}
	}
	if p.RingTeeth != p.SunTeeth+2*p.PlanetTeeth {
		return fmt.Errorf("ring teeth %d must equal sun teeth + 2 * planet teeth = %d",
			p.RingTeeth, p.SunTeeth+2*p.PlanetTeeth)
	}
	if (p.SunTeeth+p.RingTeeth)%p.PlanetCount != 0 {
		return fmt.Errorf("sun teeth + ring teeth = %d must divide by planet count %d so the planets mesh at equal angles",
			p.SunTeeth+p.RingTeeth, p.PlanetCount)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("invalid placement frame")
	}
	return p.validateFit(derive(p))
}

// validateFit checks that the parts fit around each other.
func (p Params) validateFit(d dims) error {
	m := p.Module
	if chord := 2 * d.orbit * math.Sin(math.Pi/float64(p.PlanetCount)); chord < 2*d.planetTip+m/4 {
		return fmt.Errorf("neighbouring planets collide: centers are %v apart, tips need %v",
			chord, 2*d.planetTip+m/4)
	}
	sunRoot := (m*float64(p.SunTeeth) - 2.5*m) / 2
	if p.ShaftDiameter/2 > sunRoot-m {
		return fmt.Errorf("shaft diameter %v leaves less than one module of wall under the sun root", p.ShaftDiameter)
	}
	if d.planetBore/2 > d.planetRoot-m {
		return fmt.Errorf("planet bore diameter %v leaves less than one module of wall under the planet root", d.planetBore)
	}
	if p.ShaftDiameter/2+p.PinDiameter/2 >= d.orbit {
		return fmt.Errorf("shaft diameter %v and pin diameter %v do not fit within orbit radius %v",
			p.ShaftDiameter, p.PinDiameter, d.orbit)
	}
	if d.orbit+p.PinDiameter/2 >= d.carrier {
		return fmt.Errorf("carrier radius %v does not cover the pins at orbit radius %v", d.carrier, d.orbit)
	}
	return nil
}
