// Package cycloidal builds a parametric cycloidal drive: one or two rotor discs, a pinless housing,
// an eccentric cam and an output plate with its pins.
package cycloidal

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

const (
	// regularFit is the Regular fit of docs/model-pipeline.md, the least gap between moving custom parts.
	regularFit = 0.35
	// pressFit is the radial clearance of a press-fit part and its socket, from Part clearances in
	// docs/model-pipeline.md. The 0.2 mm Tight minimum does not apply to it.
	pressFit = 0.05
	// regularMeasurementMargin covers held chamfer-mesh bounds and section measurement uncertainty.
	regularMeasurementMargin = 0.2
)

// MinClearance is the smallest nominal gap, in millimeters, that Problems accepts between moving custom parts:
// the Regular fit plus the margin that lets a measurement prove the Regular fit. It applies to the disc and casing
// (DiskClearance), the disc bore and cam (BearingClearance, diametral, so twice this), the output pins and disc
// holes (OutputPinClearance), and the two discs (DiscGap).
const MinClearance = regularFit + regularMeasurementMargin

// MinChamferSize is the minimum outer-edge chamfer required by Surface edge finishing.
const MinChamferSize = 0.5

// socketClearance is the radial gap, in millimeters, between an output pin and its socket in the output plate:
// the press fit. The pin stays in the plate once pushed in.
const socketClearance = pressFit

// Params describes one cycloidal drive. Lengths are millimeters. A diameter of zero where the field says
// "auto" selects the mean of the size's theoretical bounds.
type Params struct {
	// PinCount is N, the number of ring-pin positions. The rotor has N-1 lobes.
	PinCount          int
	PinCircleDiameter float64
	// PinDiameter is the ring-pin diameter; zero derives it.
	PinDiameter  float64
	Eccentricity float64
	// DiskClearance is the backlash cut into the rotor profile, and the gap between the rotor and the casing.
	// At least MinClearance.
	DiskClearance float64
	// DiscCount is 1, or 2 for two discs 180° apart; two discs need even PinCount and OutputPinCount.
	DiscCount     int
	DiscThickness float64
	// DiscGap separates disc 1 from disc 2 along the axis. A two-disc drive needs at least MinClearance;
	// a one-disc drive ignores it.
	DiscGap float64
	// CenterBearingDiameter is both the disc center bore and the cam outer diameter.
	CenterBearingDiameter float64
	// InputShaftDiameter is the bore through the cam; zero leaves the cam solid.
	InputShaftDiameter float64
	// BearingClearance enlarges the disc center bore over the cam, diametrally. At least 2 * MinClearance.
	BearingClearance        float64
	OutputPinCircleDiameter float64
	OutputPinCount          int
	// OutputPinDiameter is the output-pin diameter; zero derives it. The output hole is this plus twice
	// the eccentricity plus twice OutputPinClearance.
	OutputPinDiameter float64
	// OutputPinClearance is the radial gap between an output pin and the wall of a disc hole at the pin's
	// farthest offset. At least MinClearance.
	OutputPinClearance float64
	// HousingWall is the minimum radial wall of the pinless casing.
	HousingWall          float64
	BaseThickness        float64
	OutputPlateThickness float64
	// ChamferSize is a 45° chamfer on the outer rims of the rotor discs, housing, cam and output plate, and on
	// the pin ends. It must be at least MinChamferSize.
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
		DiskClearance:           MinClearance,
		DiscCount:               1,
		DiscThickness:           8,
		DiscGap:                 MinClearance,
		CenterBearingDiameter:   30,
		InputShaftDiameter:      8,
		BearingClearance:        2 * MinClearance,
		OutputPinCircleDiameter: 50,
		OutputPinCount:          6,
		OutputPinClearance:      MinClearance,
		HousingWall:             3,
		BaseThickness:           5,
		OutputPlateThickness:    5,
		ChamferSize:             MinChamferSize,
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
