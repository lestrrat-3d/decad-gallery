package crankhandle

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm.
const (
	// DefaultFlatDepth is the default depth of the shaft's flat, the same as the planetary gearbox shafts' flat.
	DefaultFlatDepth = 0.5
	// PressWall is the least wall round a press-fit bore of a separate printed part (AGENTS.md "Verification").
	PressWall = 1.6
	// BoltLength is the set screw's length, the shortest standard M2 length.
	BoltLength = 6.0
	// CavityDepth is the distance from the head's top face to the nut cavity's near face: 2 mm of material lies
	// between the cavity and the head pocket's floor, and 2 mm between the cavity and the bore's flat.
	CavityDepth = 4.0
	// RimChamfer is the leg of the 45 degree chamfer that fills the concave rims of the head pocket's floor and of
	// the nut cavity, so a section through the set screw turns 45 degrees at a time. Its foot on the pocket floor
	// stays 0.02 mm outside the head's rim, and its face is long enough that internal/cornercheck reads its two
	// turns apart (at most 46 degrees in one window).
	RimChamfer = m2fastener.HeadClearance - 0.02
	// EndLedge is the width of the arm's top face left round the grip's root fillet.
	EndLedge = 1.5
	// GripTipRound is the largest radius of the round on the grip's top edge.
	GripTipRound = 2.0
	// MaxBoreClearance is the largest radial bore clearance Validate accepts, the Loose fit.
	MaxBoreClearance = 0.5
)

// Params specifies a crank handle in mm. The shaft is a purchased bearing's shaft: its diameter equals the
// bearing's bore. A nil Placement selects the local frame (see Build).
type Params struct {
	// ShaftDiameter is the diameter of the D-flat shaft the crank fits, the bore of its bearing.
	ShaftDiameter float64
	// FlatDepth is how deep the shaft's flat cuts below its round.
	FlatDepth float64
	// BoreClearance is the radial clearance of the hub's D bore round the shaft, on the round and on the flat.
	BoreClearance float64
	// HubDiameter is the diameter of the hub round the shaft axis.
	HubDiameter float64
	// Thickness is the crank's thickness along the shaft axis, the length of the bore.
	Thickness float64
	// Throw is the distance from the shaft axis to the grip's axis.
	Throw float64
	// GripDiameter and GripLength size the round grip that stands on the arm's end.
	GripDiameter float64
	GripLength   float64
	Placement    *r3.Frame
}

// Params605ZZ returns a crank for the 5 mm shaft of a 605ZZ bearing (ballbearing.Params605ZZ).
func Params605ZZ() Params {
	return forShaft(ballbearing.Params605ZZ().BoreDiameter, 24)
}

// Params6082RS returns a crank for the 8 mm shaft of a 608-2RS bearing (ballbearing.Params6082RS). A 608ZZ has
// the same bore.
func Params6082RS() Params {
	return forShaft(ballbearing.Params6082RS().BoreDiameter, 28)
}

// DefaultParams returns Params605ZZ.
func DefaultParams() Params { return Params605ZZ() }

func forShaft(shaft, hub float64) Params {
	return Params{
		ShaftDiameter: shaft,
		FlatDepth:     DefaultFlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		HubDiameter:   hub,
		Thickness:     10,
		Throw:         50,
		GripDiameter:  14,
		GripLength:    30,
	}
}

// Dimensions are the values Derive computes from Params, in mm, in the local frame.
type Dimensions struct {
	BoreRadius   float64 // r_b: ShaftDiameter/2 + BoreClearance
	BoreFlat     float64 // f_b: distance from the axis to the bore's flat, on the -Y side
	BoreLoop     float64 // length of the bore's outline: its arc the long way round plus its flat
	HubRadius    float64 // R_h
	TailDistance float64 // t: distance from the axis to the hub's flat tail face, on the -Y side
	TailHalf     float64 // half the width of the tail face
	ScrewHeight  float64 // z of the set screw's axis, half the thickness
	RootRadius   float64 // rho: radius of the grip's root fillet
	EndRadius    float64 // R_e: radius of the arm's end round the grip axis
	TipRound     float64 // radius of the round on the grip's top edge
	// Site is the set screw's bolt and nut: the head's top face is flush with the tail face and the tip ends on
	// the bore's flat.
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
		{"HubDiameter", p.HubDiameter}, {"Thickness", p.Thickness}, {"Throw", p.Throw},
		{"GripDiameter", p.GripDiameter}, {"GripLength", p.GripLength},
	} {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) || f.value <= 0 {
			return Dimensions{}, fmt.Errorf("crankhandle: %s must be finite and positive", f.name)
		}
	}
	if p.FlatDepth > p.ShaftDiameter/4 {
		return Dimensions{}, errors.New("crankhandle: FlatDepth must be at most a quarter of ShaftDiameter")
	}
	if p.BoreClearance < ballbearing.PressFitClearance || p.BoreClearance > MaxBoreClearance {
		return Dimensions{}, fmt.Errorf("crankhandle: BoreClearance must be from %g to %g mm",
			ballbearing.PressFitClearance, MaxBoreClearance)
	}
	var dm Dimensions
	dm.BoreRadius = p.ShaftDiameter/2 + p.BoreClearance
	dm.BoreFlat = p.ShaftDiameter/2 - p.FlatDepth + p.BoreClearance
	// The bolt's hole must open on the bore's flat, not on its round.
	flatHalf := math.Sqrt(dm.BoreRadius*dm.BoreRadius - dm.BoreFlat*dm.BoreFlat)
	if flatHalf < m2fastener.HoleRadius {
		return Dimensions{}, fmt.Errorf("crankhandle: the bore's flat is %.2f mm wide, narrower than the %.1f mm bolt hole",
			2*flatHalf, 2*m2fastener.HoleRadius)
	}
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + 2*flatHalf
	dm.HubRadius = p.HubDiameter / 2
	dm.TailDistance = dm.BoreFlat + m2fastener.HeadPocketDepth + BoltLength
	if dm.HubRadius <= dm.TailDistance {
		return Dimensions{}, fmt.Errorf("crankhandle: HubDiameter must exceed %.2f mm to hold the set screw",
			2*dm.TailDistance)
	}
	dm.TailHalf = math.Sqrt(dm.HubRadius*dm.HubRadius - dm.TailDistance*dm.TailDistance)
	// 1 mm of material round the head pocket on the tail face, and round the nut cavity.
	if need := m2fastener.HeadPocketRadius + 1; dm.TailHalf < need {
		return Dimensions{}, fmt.Errorf("crankhandle: HubDiameter must be at least %.2f mm to hold the set screw's head",
			2*math.Hypot(dm.TailDistance, need))
	}
	if dm.HubRadius-dm.BoreRadius < PressWall {
		return Dimensions{}, fmt.Errorf("crankhandle: HubDiameter leaves less than a %.1f mm wall round the bore", PressWall)
	}
	if need := 2 * (m2fastener.NutCavityRadius + 1); p.Thickness < need {
		return Dimensions{}, fmt.Errorf("crankhandle: Thickness must be at least %.2f mm to enclose the nut", need)
	}
	dm.ScrewHeight = p.Thickness / 2
	dm.RootRadius = math.Min(p.Thickness, p.GripDiameter) / 2
	dm.EndRadius = p.GripDiameter/2 + dm.RootRadius + EndLedge
	dm.TipRound = math.Min(GripTipRound, p.GripDiameter/4)
	if p.GripLength < dm.RootRadius+dm.TipRound+1 {
		return Dimensions{}, fmt.Errorf("crankhandle: GripLength must be at least %.2f mm", dm.RootRadius+dm.TipRound+1)
	}
	if p.Throw < dm.HubRadius+dm.EndRadius {
		return Dimensions{}, fmt.Errorf("crankhandle: Throw must be at least %.2f mm so the arm's end clears the hub",
			dm.HubRadius+dm.EndRadius)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errors.New("crankhandle: Placement must be a valid frame")
	}
	dm.Site = m2fastener.Site{
		Head:   r3.Vec{Y: -dm.TailDistance, Z: dm.ScrewHeight},
		Dir:    r3.Vec{Y: 1},
		Corner: r3.Vec{X: 1},
		Length: BoltLength,
		Cavity: CavityDepth,
	}
	if err := dm.Site.Validate(); err != nil {
		return Dimensions{}, fmt.Errorf("crankhandle: %w", err)
	}
	return dm, nil
}
