package planetarygearbox

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
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
	// CoverThickness is the axial thickness of each end cover's frame, which the bearing boss stands on.
	CoverThickness float64
	// BoltHoleDiameter is the diameter of the four clearance holes that join the covers and the ring.
	BoltHoleDiameter float64
	// ShaftFlat is the depth of the flat on each shaft, the input shaft and every carrier's stub. A flat turns the
	// shaft into a D, and the sun bore it fills is the same D, so the shaft carries torque into the sun. Zero
	// leaves round shafts and round bores.
	ShaftFlat float64
	// InputShaftLength is how far the input shaft reaches out of the front cover's bearing boss.
	InputShaftLength float64
	// BearingClearance is the radial clearance between a planet's 608ZZ bearing and its pocket.
	BearingClearance float64
	// CarrierWall is the width of carrier material around the shaft and around each planet pin.
	CarrierWall float64
	// RingWall is the radial wall of the ring behind the tooth roots. It carries the bolt holes.
	RingWall float64
	// Backlash is how much each tooth of the sun and the planets is thinned at the pitch
	// circle. The ring's tooth spaces are widened by the same amount, so a sun-planet pair and
	// a planet-ring pair each have twice Backlash of play.
	Backlash float64
	// OutputShaftLength is how far the output shaft reaches out of the rear cover's bearing boss.
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
		CoverThickness:    3,
		BoltHoleDiameter:  3.4,
		ShaftFlat:         1,
		InputShaftLength:  12,
		BearingClearance:  0.1,
		CarrierWall:       3,
		RingWall:          7,
		Backlash:          0.05,
		OutputShaftLength: 12,
	}
}

// Ratio returns the reduction from the first sun to the last carrier with the
// ring held fixed.
func (p Params) Ratio() float64 {
	stage := 1 + float64(p.RingTeeth)/float64(p.SunTeeth)
	return math.Pow(stage, float64(p.Stages))
}

// planetBearing is the purchased bearing each planet turns on.
func planetBearing() ballbearing.Params { return ballbearing.Params608ZZ() }

// lipWidth is the radial width of the lip that holds a bearing in its pocket: the lip bore is this much
// smaller than the bearing's outside radius. lipThickness is the axial thickness of that lip.
const (
	lipWidth     = 2
	lipThickness = 1
	// boltRimMargin is the material kept between a bolt hole and the edge of the ring or cover rim.
	boltRimMargin = 1.5
	// armMargin is how much wider than a pin each carrier arm and cover spoke is on either side, so the pin
	// is not tangent to the arm edges.
	armMargin = 1
	// collarMargin is how far a shaft collar reaches past the shaft on each side. A collar is the printed
	// stand-in for a washer: it spans the gap between a bearing and the part that would rub on it, and it
	// must stay narrower than the bearing's inner race (a 608 inner race is about 11 mm across).
	collarMargin = 1.2
	// coverSpokes is the number of spokes joining a cover's rim to its bearing hub.
	coverSpokes = 3
)

// dims are the derived dimensions in millimeters.
type dims struct {
	orbit, planetRoot, planetTip float64
	ringTip, ringRoot, ringOuter float64
	pin, pocket, lip             float64
	boss, arm, collar            float64
	shaft, coverBoss, coverHub   float64
	boltCircle, rimInner         float64
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
		stagePitch: p.GearThickness + p.CarrierThickness + 2*p.StageGap,
	}
	b := planetBearing()
	d.pin = b.BoreDiameter
	d.pocket = b.OuterDiameter + 2*p.BearingClearance
	d.lip = b.OuterDiameter - 2*lipWidth
	d.shaft = b.BoreDiameter
	d.collar = d.shaft + 2*collarMargin
	d.coverBoss = d.pocket + 2*p.CarrierWall
	d.coverHub = d.coverBoss + 2*p.CarrierWall
	d.boss = d.pin + 2*p.CarrierWall
	d.arm = d.pin + 2*armMargin
	d.ringOuter = d.ringRoot + p.RingWall
	d.boltCircle = (d.ringRoot + d.ringOuter) / 2
	d.rimInner = d.boltCircle - p.BoltHoleDiameter/2 - boltRimMargin
	d.ringLength = float64(p.Stages)*d.stagePitch + p.StageGap
	return d
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Validate applies the input checks of the construction contract.
func (p Params) Validate() error {
	for name, v := range map[string]float64{
		"Module": p.Module, "PressureAngle": p.PressureAngle, "GearThickness": p.GearThickness,
		"CarrierThickness": p.CarrierThickness, "StageGap": p.StageGap, "CoverThickness": p.CoverThickness,
		"BoltHoleDiameter": p.BoltHoleDiameter, "InputShaftLength": p.InputShaftLength, "ShaftFlat": p.ShaftFlat,
		"BearingClearance": p.BearingClearance, "CarrierWall": p.CarrierWall, "RingWall": p.RingWall, "Backlash": p.Backlash,
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
		{"cover thickness", p.CoverThickness, p.CoverThickness > 0},
		{"bolt hole diameter", p.BoltHoleDiameter, p.BoltHoleDiameter > 0},
		{"input shaft length", p.InputShaftLength, p.InputShaftLength > 0},
		{"shaft flat", p.ShaftFlat, p.ShaftFlat >= 0 && p.ShaftFlat <= planetBearing().BoreDiameter/4},
		{"bearing clearance", p.BearingClearance, p.BearingClearance >= 0},
		{"carrier wall", p.CarrierWall, p.CarrierWall > 0},
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
	if d.shaft/2 > sunRoot-m {
		return fmt.Errorf("the %v mm shaft leaves less than one module of wall under the sun root", d.shaft)
	}
	b := planetBearing()
	if p.GearThickness-b.Width < lipThickness {
		return fmt.Errorf("gear thickness %v leaves less than %v for the lip behind a %v wide bearing",
			p.GearThickness, float64(lipThickness), b.Width)
	}
	if d.pocket/2 > d.planetRoot-m {
		return fmt.Errorf("bearing pocket diameter %v leaves less than one module of wall under the planet root", d.pocket)
	}
	if d.shaft/2+d.boss/2 >= d.orbit {
		return fmt.Errorf("shaft diameter %v and planet pin boss %v do not fit within orbit radius %v",
			d.shaft, d.boss, d.orbit)
	}
	if d.boltCircle-p.BoltHoleDiameter/2 < d.ringRoot+boltRimMargin ||
		d.boltCircle+p.BoltHoleDiameter/2 > d.ringOuter-boltRimMargin {
		return fmt.Errorf("bolt holes of diameter %v do not fit the ring wall %v with %v of material either side",
			p.BoltHoleDiameter, p.RingWall, boltRimMargin)
	}
	if d.coverHub/2 >= d.rimInner-d.arm {
		return fmt.Errorf("the cover's bearing hub of diameter %v leaves no room for spokes inside the rim at radius %v",
			d.coverHub, d.rimInner)
	}
	if d.orbit+d.boss/2 >= d.ringTip-m/2 {
		return fmt.Errorf("the carrier reaches radius %v, past the ring's tip radius %v", d.orbit+d.boss/2, d.ringTip)
	}
	return nil
}
