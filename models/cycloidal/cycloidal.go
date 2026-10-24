// Package cycloidal builds a parametric cycloidal drive: one or two rotor discs, a pinless housing,
// an eccentric cam and an output plate with its pins.
package cycloidal

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// Params describes one cycloidal drive. Lengths are millimeters. A diameter of zero where the field says
// "auto" selects the mean of the size's theoretical bounds.
type Params struct {
	// PinCount is N, the number of ring-pin positions. The rotor has N-1 lobes.
	PinCount          int
	PinCircleDiameter float64
	// PinDiameter is the ring-pin diameter; zero derives it.
	PinDiameter  float64
	Eccentricity float64
	// DiskClearance is the backlash cut into the rotor profile.
	DiskClearance float64
	// DiscCount is 1, or 2 for two discs 180° apart; two discs need even PinCount and OutputPinCount.
	DiscCount     int
	DiscThickness float64
	// DiscGap separates disc 1 from disc 2.
	DiscGap float64
	// CenterBearingDiameter is both the disc center bore and the cam outer diameter.
	CenterBearingDiameter float64
	// InputShaftDiameter is the bore through the cam; zero leaves the cam solid.
	InputShaftDiameter float64
	// BearingClearance enlarges the disc center bore over the cam, diametrally.
	BearingClearance        float64
	OutputPinCircleDiameter float64
	OutputPinCount          int
	// OutputPinDiameter is the output-pin diameter; zero derives it. The output hole is this plus twice
	// the eccentricity.
	OutputPinDiameter float64
	// HousingWall is the minimum radial wall of the pinless casing.
	HousingWall          float64
	BaseThickness        float64
	OutputPlateThickness float64
	// ChamferSize is a 45° chamfer on the outer rims of the rotor discs, the housing and the output plate and on
	// the pin ends; zero leaves them sharp.
	ChamferSize float64
	// Placement is the reference point and plane; nil selects the world XY frame at the origin. The first
	// disc's lower face lies on the plane and the stack grows along its normal.
	Placement *r3.Frame
}

// DefaultParams returns the add-in's default drive: 16 pins, a 90 mm pin circle and one disc.
func DefaultParams() Params {
	return Params{
		PinCount:                16,
		PinCircleDiameter:       90,
		Eccentricity:            1.5,
		DiskClearance:           0.3,
		DiscCount:               1,
		DiscThickness:           8,
		DiscGap:                 0.5,
		CenterBearingDiameter:   30,
		InputShaftDiameter:      8,
		BearingClearance:        0.2,
		OutputPinCircleDiameter: 50,
		OutputPinCount:          6,
		HousingWall:             3,
		BaseThickness:           5,
		OutputPlateThickness:    5,
		ChamferSize:             0.5,
	}
}

// Drive holds the finished bodies. Their names follow the Fusion add-in: "Cycloidal Disk 1" (and 2),
// "Housing", "Eccentric Cam", "Output Plate" and "Output Pin 1" to "Output Pin M".
type Drive struct {
	Discs       []*decad.Body
	Housing     *decad.Body
	Cam         *decad.Body
	OutputPlate *decad.Body
	OutputPins  []*decad.Body
}
