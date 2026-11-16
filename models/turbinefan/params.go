package turbinefan

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). Values that another model owns are referenced, not
// redefined.
const (
	BladeCount = 3 // the requirement: three replaceable blades

	HubRadius = handcrankfan.HubRadius // 14, the hand-crank fan's hub: the blade's root and stud fix it
	HubLength = handcrankfan.HubLength // 20
	HubMid    = HubLength / 2          // the sockets' and blade bolts' plane

	FirstBladeAngle = math.Pi / 2 // blade 1 along +Y; the set screw lies along -Y, midway between blades 2 and 3

	SetScrewBolt     = 6.0 // M2 x 6: the longest standard length whose head stays inside the round face at both presets
	SetScrewZ        = 4.0 // the set screw's line: the counterbore keeps 1.8 to the back face, and a shaft need reach only 6 into the hub
	SetScrewTravel   = 0.4 // gap under the head, so the screw grips before its head reaches the counterbore floor
	CounterboreRound = 0.5 // the counterbore's floor rim round ("Corners and roots")
	NutSlotRound     = 0.5 // the rounds of the nut slots' inside corners ("Corners and roots")

	ChamferSize = crankhandle.ChamferSize // 0.5, the hub's face chamfer
	PressWall   = crankhandle.PressWall   // 1.6, the least wall round a press-fit bore
	RegularFit  = crankhandle.RegularFit  // 0.40: the stud in its socket
)

// BladeAngle returns the angle, about +Z from +X, of blade k's radial line, k = 1, 2, 3: 90, 210 and 330 degrees.
func BladeAngle(k int) float64 { return FirstBladeAngle + 2*math.Pi*float64(k-1)/BladeCount }

// Hand is the way the fan turns to blow air forward. Seen from the front, a right-hand fan turns clockwise and a
// left-hand fan counter-clockwise. A left-hand fan's blades are the right-hand blades reflected in the plane x = 0.
type Hand int

const (
	// RightHand is the zero value: clockwise seen from the front.
	RightHand Hand = iota
	// LeftHand turns counter-clockwise seen from the front.
	LeftHand
)

// Params specifies a fan in mm and radians. The shaft runs in a purchased bearing, so its diameter equals the bearing's
// bore. A nil Placement selects the local frame.
type Params struct {
	// ShaftDiameter is the diameter of the D-flat shaft the hub grips; it equals the bearing bore.
	ShaftDiameter float64
	// FlatDepth is how deep the shaft's flat cuts below its round.
	FlatDepth float64
	// BoreClearance is the radial clearance of the hub's D bore round the shaft, on the round and on the flat.
	BoreClearance float64
	// BladePitch is the blade angle at 48 mm radius, in radians, from handcrankfan.MinBladePitch to MaxBladePitch.
	BladePitch float64
	// Hand selects the blades' hand; the zero value is RightHand.
	Hand      Hand
	Placement *r3.Frame
}

// Params605ZZ returns a fan for the 5 mm shaft of a 605ZZ bearing, with the planetary gearbox shafts' 0.5 mm flat.
func Params605ZZ() Params {
	return Params{
		ShaftDiameter: ballbearing.Params605ZZ().BoreDiameter,
		FlatDepth:     crankhandle.Params605ZZ().FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		BladePitch:    handcrankfan.DefaultBladePitch,
	}
}

// Params608ZZ returns a fan for the 8 mm shaft of a 608ZZ bearing, with a 0.75 mm flat. A 608-2RS has the same bore.
func Params608ZZ() Params {
	return Params{
		ShaftDiameter: ballbearing.Params608ZZ().BoreDiameter,
		FlatDepth:     crankhandle.Params6082RS().FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		BladePitch:    handcrankfan.DefaultBladePitch,
	}
}

// DefaultParams returns Params605ZZ.
func DefaultParams() Params { return Params605ZZ() }

// Dimensions are the values Derive computes from Params, in mm, in the local frame. rho is a distance from the shaft
// axis along -Y.
type Dimensions struct {
	BoreRadius       float64 // r_b: ShaftDiameter/2 + BoreClearance
	BoreFlat         float64 // f_b: D/2 - d + c; the bore's flat lies at y = -f_b
	BoreFlatWidth    float64 // w_b: 2 sqrt(r_b^2 - f_b^2)
	BoreLoop         float64 // length of the bore's outline: 2 r_b (pi - acos(f_b / r_b)) + w_b
	ShaftFlatWidth   float64 // w_s: 2 sqrt(d (D - d)), the width of the shaft's flat
	SetScrewTip      float64 // rho_t: D/2 - d - c, the flat of the shaft pushed c toward +Y
	SetScrewHead     float64 // rho_h: rho_t + HeadPocketDepth + SetScrewBolt, the head's top face
	CounterboreFloor float64 // rho_f: rho_h - HeadPocketDepth - SetScrewTravel
	// SetScrewSite is the set screw: its head's top face at (0, -rho_h, SetScrewZ), running along +Y. Its Cavity puts the
	// nut slot's inner face PressWall from the bore's flat.
	SetScrewSite m2fastener.Site
	// BladeSites are the three blade bolts: each head's top face is flush with the front face (z = HubLength) at
	// BladeBoltAt from the axis along blade k's radial line, and the bolt runs along -Z.
	BladeSites [BladeCount]m2fastener.Site
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
		{"BladePitch", p.BladePitch},
	} {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) || f.value <= 0 {
			return Dimensions{}, fmt.Errorf("turbinefan: %s must be finite and positive", f.name)
		}
	}
	if p.FlatDepth > p.ShaftDiameter/4 {
		return Dimensions{}, errors.New("turbinefan: FlatDepth must be at most a quarter of ShaftDiameter")
	}
	if p.BoreClearance < ballbearing.PressFitClearance || p.BoreClearance > 0.5 {
		return Dimensions{}, fmt.Errorf("turbinefan: BoreClearance must be from %g to 0.5 mm", ballbearing.PressFitClearance)
	}
	dm := Dimensions{
		BoreRadius: p.ShaftDiameter/2 + p.BoreClearance,
		BoreFlat:   p.ShaftDiameter/2 - p.FlatDepth + p.BoreClearance,
	}
	dm.BoreFlatWidth = 2 * math.Sqrt(dm.BoreRadius*dm.BoreRadius-dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + dm.BoreFlatWidth
	dm.ShaftFlatWidth = 2 * math.Sqrt(p.FlatDepth*(p.ShaftDiameter-p.FlatDepth))
	if handcrankfan.BladeSocketStart-dm.BoreRadius < PressWall-1e-9 {
		return Dimensions{}, fmt.Errorf("turbinefan: ShaftDiameter leaves less than a %.1f mm wall to the blade sockets",
			PressWall)
	}
	if dm.ShaftFlatWidth < m2bolt.ShankDiameter-1e-9 {
		return Dimensions{}, errors.New("turbinefan: FlatDepth must leave the shaft's flat at least 2 mm wide for the set screw's tip")
	}
	if dm.BoreFlatWidth < 2*(m2fastener.HoleRadius+0.1)-1e-9 {
		return Dimensions{}, errors.New("turbinefan: FlatDepth must leave the bore's flat at least 2.8 mm wide for the set screw's hole")
	}
	if p.BladePitch < handcrankfan.MinBladePitch || p.BladePitch > handcrankfan.MaxBladePitch {
		return Dimensions{}, errors.New("turbinefan: BladePitch must lie in [10°, 40°] radians")
	}
	if p.Hand != RightHand && p.Hand != LeftHand {
		return Dimensions{}, errors.New("turbinefan: Hand must be RightHand or LeftHand")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errors.New("turbinefan: Placement must be a valid frame")
	}
	dm.SetScrewTip = p.ShaftDiameter/2 - p.FlatDepth - p.BoreClearance
	dm.SetScrewHead = dm.SetScrewTip + m2fastener.HeadPocketDepth + SetScrewBolt
	dm.CounterboreFloor = dm.SetScrewHead - m2fastener.HeadPocketDepth - SetScrewTravel
	dm.SetScrewSite = m2fastener.Site{
		Head:   r3.Vec{Y: -dm.SetScrewHead, Z: SetScrewZ},
		Dir:    r3.Vec{Y: 1},
		Corner: r3.Vec{Z: 1},
		Length: SetScrewBolt,
		Cavity: dm.SetScrewHead - dm.BoreFlat - PressWall - m2fastener.NutCavityHeight,
	}
	if err := dm.SetScrewSite.Validate(); err != nil {
		return Dimensions{}, fmt.Errorf("turbinefan: set screw: %w", err)
	}
	if dm.SetScrewHead > math.Sqrt(HubRadius*HubRadius-m2bolt.HeadDiameter*m2bolt.HeadDiameter/4) {
		return Dimensions{}, errors.New("turbinefan: set screw: its head would stand out of the round face")
	}
	if dm.SetScrewSite.Cavity-m2fastener.HeadPocketDepth-SetScrewTravel < 1-1e-9 {
		return Dimensions{}, errors.New("turbinefan: set screw: less than 1 mm between its counterbore and its nut slot")
	}
	if SetScrewZ-m2fastener.HeadPocketRadius < 1 {
		return Dimensions{}, errors.New("turbinefan: set screw: less than 1 mm between its counterbore and the back face")
	}
	for i := range dm.BladeSites {
		u := r3.Vec{X: math.Cos(BladeAngle(i + 1)), Y: math.Sin(BladeAngle(i + 1))}
		dm.BladeSites[i] = m2fastener.Site{
			Head:   u.Scale(handcrankfan.BladeBoltAt).Add(r3.Vec{Z: HubLength}),
			Dir:    r3.Vec{Z: -1},
			Corner: u,
			Length: handcrankfan.BladeBoltLength,
			Cavity: handcrankfan.BladeBoltCavity,
		}
		if err := dm.BladeSites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("turbinefan: blade bolt %d: %w", i+1, err)
		}
	}
	return dm, nil
}
