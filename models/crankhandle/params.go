package crankhandle

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). They are the hand-crank siren's crank, which imports this
// package. Each fit gap is its class minimum from docs/model-pipeline.md "Part clearances" plus FitMargin.
const (
	FitMargin  = 0.05
	TightFit   = 0.2 + FitMargin  // 0.25: the pin's stub in the arm's hexagonal pocket
	RegularFit = 0.35 + FitMargin // 0.40: the handle on the pin's shank, and its gap to the arm

	ArmRadius    = 10.0 // the arm is a stadium 20 wide
	ArmThickness = 8.0
	BoreFloor    = 1.0 // the D bore is blind: 1 mm of arm closes it at the outer face
	PinHoleR     = 1.3 // the pin bolt's clearance hole through the arm
	PocketFlats  = StubFlats + 2*TightFit
	PocketFrom   = 3.25 // the hexagonal pocket runs from here through the outer face
	PocketLength = 5.75
	StubFlats    = 7.03 // keeps 1 mm of stub round the nut cavity (radius 2.51)
	StubFrom     = 3.5
	StubLength   = 5.05
	ShankR       = 4.5
	ShankFrom    = 8.25
	ShankTo      = 55.4
	HeadR        = 5.8
	HeadFrom     = 55.0
	HeadLength   = 3.0
	HandleFrom   = 7.4 // the handle sketch's origin, 1 behind the handle's inner end
	PinBolt      = 10.0
	PinCavity    = 4.75 // from the bolt head's top face, in the arm's back face, to the nut cavity's near face
	PressWall    = 1.6  // the least wall round a press-fit bore (AGENTS.md "Verification")
	ChamferSize  = 0.5  // the arm's face chamfer
)

// HandleProfile is the handle's meridian: (distance along the pin from the sketch origin at HandleFrom, radius). A small
// barrel with a waist and a flared outer end, with a bore for the pin's shank and a counterbore for its head. Every
// corner is cut by a small chamfer, so the handle is made of cones and cylinders. The origin is 1 behind the inner end.
var HandleProfile = [...][2]float64{
	{1.5, 4.9}, {47.2, 4.9}, {47.2, 6.4}, {50.5, 6.4}, {51, 6.9}, {51, 8.3}, {49.8, 9.3},
	{44, 8}, {37, 7.25}, {23, 8}, {9, 7.8}, {2.2, 7.2}, {1, 6}, {1, 5.4},
}

// Params specifies a crank in mm. The shaft runs in a purchased bearing, so its diameter equals the bearing's bore. A
// nil Placement selects the local frame.
type Params struct {
	// ShaftDiameter is the diameter of the D-flat shaft the arm presses onto.
	ShaftDiameter float64
	// FlatDepth is how deep the shaft's flat cuts below its round.
	FlatDepth float64
	// BoreClearance is the radial clearance of the arm's D bore round the shaft, on the round and on the flat.
	BoreClearance float64
	// Throw is the distance from the shaft axis to the pin axis.
	Throw     float64
	Placement *r3.Frame
}

// Params605ZZ returns a crank for the 5 mm shaft of a 605ZZ bearing (ballbearing.Params605ZZ), with the planetary
// gearbox shafts' 0.5 mm flat.
func Params605ZZ() Params {
	return Params{ShaftDiameter: ballbearing.Params605ZZ().BoreDiameter, FlatDepth: 0.5,
		BoreClearance: ballbearing.PressFitClearance, Throw: 55}
}

// Params6082RS returns a crank for the 8 mm shaft of a 608-2RS bearing (ballbearing.Params6082RS), with a 0.75 mm
// flat: the hand-crank siren's crank. A 608ZZ has the same bore.
func Params6082RS() Params {
	return Params{ShaftDiameter: ballbearing.Params6082RS().BoreDiameter, FlatDepth: 0.75,
		BoreClearance: ballbearing.PressFitClearance, Throw: 55}
}

// DefaultParams returns Params605ZZ.
func DefaultParams() Params { return Params605ZZ() }

// Dimensions are the values Derive computes from Params, in mm, in the local frame.
type Dimensions struct {
	BoreRadius float64 // r_b: ShaftDiameter/2 + BoreClearance
	BoreFlat   float64 // f_b: distance from the axis to the bore's flat, on the +Y side
	BoreLoop   float64 // length of the bore's outline in the back face: its arc the long way round plus its flat
	// Site is the pin bolt: its head's top face is flush with the arm's back face at the pin axis, and it runs along
	// +Z into the nut cavity in the pin's stub.
	Site m2fastener.Site
}

// Validate reports the first problem Derive finds.
func (p Params) Validate() error {
	_, err := p.Derive()
	return err
}

// Derive validates p and computes its dimensions.
func (p Params) Derive() (Dimensions, error) {
	for _, f := range []struct {
		name  string
		value float64
	}{
		{"ShaftDiameter", p.ShaftDiameter}, {"FlatDepth", p.FlatDepth}, {"BoreClearance", p.BoreClearance},
		{"Throw", p.Throw},
	} {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) || f.value <= 0 {
			return Dimensions{}, fmt.Errorf("crankhandle: %s must be finite and positive", f.name)
		}
	}
	if p.FlatDepth > p.ShaftDiameter/4 {
		return Dimensions{}, errors.New("crankhandle: FlatDepth must be at most a quarter of ShaftDiameter")
	}
	if p.BoreClearance < ballbearing.PressFitClearance || p.BoreClearance > 0.5 {
		return Dimensions{}, fmt.Errorf("crankhandle: BoreClearance must be from %g to 0.5 mm",
			ballbearing.PressFitClearance)
	}
	dm := Dimensions{
		BoreRadius: p.ShaftDiameter/2 + p.BoreClearance,
		BoreFlat:   p.ShaftDiameter/2 - p.FlatDepth + p.BoreClearance,
	}
	half := math.Sqrt(dm.BoreRadius*dm.BoreRadius - dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + 2*half
	if ArmRadius-dm.BoreRadius < PressWall {
		return Dimensions{}, fmt.Errorf("crankhandle: ShaftDiameter leaves less than a %.1f mm wall in the %g mm arm",
			PressWall, 2*ArmRadius)
	}
	if p.Throw < 2*ArmRadius {
		return Dimensions{}, fmt.Errorf("crankhandle: Throw must be at least %g mm", 2*ArmRadius)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errors.New("crankhandle: Placement must be a valid frame")
	}
	dm.Site = m2fastener.Site{
		Head:   r3.Vec{Y: p.Throw},
		Dir:    r3.Vec{Z: 1},
		Corner: r3.Vec{X: 1},
		Length: PinBolt,
		Cavity: PinCavity,
	}
	if err := dm.Site.Validate(); err != nil {
		return Dimensions{}, fmt.Errorf("crankhandle: %w", err)
	}
	return dm, nil
}
