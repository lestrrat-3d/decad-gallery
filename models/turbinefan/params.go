package turbinefan

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). Values that another model owns are referenced, not
// redefined.
const (
	BladeCount = 3 // the requirement: three replaceable blades

	HubRadius = 14.0 // the blade's root and stud fix it
	HubLength = 20.0
	HubMid    = HubLength / 2 // the sockets' and blade bolts' plane

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

// Fixed dimensions of the clutch mount in mm and radians. Values that another model owns are referenced.
const (
	ClutchTurn          = math.Pi / 3                                                                  // the fan frame's turn about Z from the clutch frame
	FlangeThickness     = m2fastener.HeadPocketDepth + 1.0                                             // 3.0: the head pocket plus 1 mm under it
	FlangeRadius        = 28.0                                                                         // covers the ear tips (onewayclutch.EarReach 27.1)
	FlangeRound         = 1.5                                                                          // the hub's root round on the flange: half its thickness
	ClutchBoreClearance = onewayclutch.LooseFit                                                        // 0.55
	ClutchSeam          = onewayclutch.Seam                                                            // 0.25
	ClampThickness      = 1 + m2fastener.NutCavityHeight + 1                                           // 4.0
	ClampInnerRadius    = 19.0                                                                         // 0.5 off the clutch's 37 mm circle
	ClampOuterRadius    = FlangeRadius                                                                 // 28
	ClampBack           = -(FlangeThickness + 2*ClutchSeam + onewayclutch.RingHeight + ClampThickness) // -28.5
	EarBoltLength       = 25.0                                                                         // M2 x 25, the longest standard length
	EarBoltCavity       = -(ClampBack + 1 + m2fastener.NutCavityHeight)                                // 25.5
	MinWall             = 0.8                                                                          // least wall of a part that does not bend
	MountZ              = onewayclutch.RingTo + ClutchSeam + FlangeThickness                           // 23.0: the fan frame's origin in the clutch frame
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

// EarBoltAngle returns the angle about +Z from +X, in the fan frame, of ear bolt k, k = 1, 2, 3: 30, 150 and 270 degrees.
func EarBoltAngle(k int) float64 { return onewayclutch.EarAngles[k-1]*math.Pi/180 - ClutchTurn }

// Freewheel returns the clutch direction that drives a fan of hand h the way it blows air forward and lets it coast:
// Clockwise for RightHand, CounterClockwise for LeftHand, 0 for any other Hand.
func (h Hand) Freewheel() onewayclutch.Direction {
	switch h {
	case RightHand:
		return onewayclutch.Clockwise
	case LeftHand:
		return onewayclutch.CounterClockwise
	}
	return 0
}

// Mount is how the hub is held.
type Mount int

const (
	// ShaftMount is the zero value: the hub's D bore grips the shaft and a set screw holds it.
	ShaftMount Mount = iota
	// ClutchMount bolts the hub's flange to the ring ears of a onewayclutch on the shaft; a round bore clears the shaft.
	ClutchMount
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
	// BladePitch is the blade angle at 48 mm radius, in radians, from MinBladePitch to MaxBladePitch.
	BladePitch float64
	// Hand selects the blades' hand; the zero value is RightHand.
	Hand Hand
	// Mount selects how the hub is held; the zero value is ShaftMount.
	Mount     Mount
	Placement *r3.Frame
}

// Params605ZZ returns a fan for the 5 mm shaft of a 605ZZ bearing, with the planetary gearbox shafts' 0.5 mm flat.
func Params605ZZ() Params {
	return Params{
		ShaftDiameter: ballbearing.Params605ZZ().BoreDiameter,
		FlatDepth:     crankhandle.Params605ZZ().FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		BladePitch:    DefaultBladePitch,
	}
}

// Params608ZZ returns a fan for the 8 mm shaft of a 608ZZ bearing, with a 0.75 mm flat. A 608-2RS has the same bore.
func Params608ZZ() Params {
	return Params{
		ShaftDiameter: ballbearing.Params608ZZ().BoreDiameter,
		FlatDepth:     crankhandle.Params6082RS().FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		BladePitch:    DefaultBladePitch,
	}
}

// MountFrame returns the Placement that seats a ClutchMount fan on a clutch built with Placement clutch (nil selects
// the clutch's local frame).
func MountFrame(clutch *r3.Frame) (r3.Frame, error) {
	sin, cos := math.Sincos(ClutchTurn)
	local, err := r3.NewFrame(r3.Vec{Z: MountZ}, r3.Vec{X: cos, Y: sin}, r3.Vec{X: -sin, Y: cos})
	if err != nil {
		return r3.Frame{}, fmt.Errorf("turbinefan: mount frame: %w", err)
	}
	if clutch == nil {
		return local, nil
	}
	if !clutch.IsValid() {
		return r3.Frame{}, errors.New("turbinefan: clutch Placement must be a valid frame")
	}
	xf, err := r3.FromFrame(*clutch)
	if err != nil {
		return r3.Frame{}, fmt.Errorf("turbinefan: clutch placement: %w", err)
	}
	f, err := r3.NewFrame(xf.Apply(local.Origin()), xf.ApplyDir(local.U()), xf.ApplyDir(local.V()))
	if err != nil {
		return r3.Frame{}, fmt.Errorf("turbinefan: clutch placement: %w", err)
	}
	return f, nil
}

// ParamsForClutch returns a ClutchMount fan matched to the clutch c: c's shaft diameter and flat, the hand that c's
// Freewheel drives forward, the default pitch, ballbearing.PressFitClearance, and Placement MountFrame(c.Placement).
func ParamsForClutch(c onewayclutch.Params) (Params, error) {
	dm, err := c.Derive()
	if err != nil {
		return Params{}, fmt.Errorf("turbinefan: clutch: %w", err)
	}
	f, err := MountFrame(c.Placement)
	if err != nil {
		return Params{}, err
	}
	hand := RightHand
	if c.Freewheel == onewayclutch.CounterClockwise {
		hand = LeftHand
	}
	return Params{
		ShaftDiameter: dm.Shaft.Diameter,
		FlatDepth:     dm.Shaft.FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		BladePitch:    DefaultBladePitch,
		Hand:          hand,
		Mount:         ClutchMount,
		Placement:     &f,
	}, nil
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
	// ClutchBoreRadius is r_c: ShaftDiameter/2 + ClutchBoreClearance in ClutchMount, else 0.
	ClutchBoreRadius float64
	// EarSites are the three ear bolts in ClutchMount, local frame, each head's top face flush with the flange's front
	// face; zero in ShaftMount.
	EarSites [onewayclutch.Ears]m2fastener.Site
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
	if p.Mount != ShaftMount && p.Mount != ClutchMount {
		return Dimensions{}, errors.New("turbinefan: Mount must be ShaftMount or ClutchMount")
	}
	clutch := p.Mount == ClutchMount
	dm := Dimensions{
		BoreRadius: p.ShaftDiameter/2 + p.BoreClearance,
		BoreFlat:   p.ShaftDiameter/2 - p.FlatDepth + p.BoreClearance,
	}
	dm.BoreFlatWidth = 2 * math.Sqrt(dm.BoreRadius*dm.BoreRadius-dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + dm.BoreFlatWidth
	dm.ShaftFlatWidth = 2 * math.Sqrt(p.FlatDepth*(p.ShaftDiameter-p.FlatDepth))
	if clutch {
		dm.ClutchBoreRadius = p.ShaftDiameter/2 + ClutchBoreClearance
		if BladeSocketStart-dm.ClutchBoreRadius < MinWall-1e-9 {
			return Dimensions{}, fmt.Errorf("turbinefan: ShaftDiameter leaves less than a %.1f mm wall between the clutch "+
				"mount's bore and the blade sockets", MinWall)
		}
	} else {
		if BladeSocketStart-dm.BoreRadius < PressWall-1e-9 {
			return Dimensions{}, fmt.Errorf("turbinefan: ShaftDiameter leaves less than a %.1f mm wall to the blade sockets",
				PressWall)
		}
		if dm.ShaftFlatWidth < m2bolt.ShankDiameter-1e-9 {
			return Dimensions{}, errors.New("turbinefan: FlatDepth must leave the shaft's flat at least 2 mm wide for the set screw's tip")
		}
		if dm.BoreFlatWidth < 2*(m2fastener.HoleRadius+0.1)-1e-9 {
			return Dimensions{}, errors.New("turbinefan: FlatDepth must leave the bore's flat at least 2.8 mm wide for the set screw's hole")
		}
	}
	if p.BladePitch < MinBladePitch || p.BladePitch > MaxBladePitch {
		return Dimensions{}, errors.New("turbinefan: BladePitch must lie in [10°, 40°] radians")
	}
	if p.Hand != RightHand && p.Hand != LeftHand {
		return Dimensions{}, errors.New("turbinefan: Hand must be RightHand or LeftHand")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errors.New("turbinefan: Placement must be a valid frame")
	}
	if !clutch {
		if err := dm.deriveSetScrew(p); err != nil {
			return Dimensions{}, err
		}
	}
	for i := range dm.BladeSites {
		u := r3.Vec{X: math.Cos(BladeAngle(i + 1)), Y: math.Sin(BladeAngle(i + 1))}
		dm.BladeSites[i] = m2fastener.Site{
			Head:   u.Scale(BladeBoltAt).Add(r3.Vec{Z: HubLength}),
			Dir:    r3.Vec{Z: -1},
			Corner: u,
			Length: BladeBoltLength,
			Cavity: BladeBoltCavity,
		}
		if err := dm.BladeSites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("turbinefan: blade bolt %d: %w", i+1, err)
		}
	}
	if !clutch {
		return dm, nil
	}
	for i := range dm.EarSites {
		a := EarBoltAngle(i + 1)
		u := r3.Vec{X: math.Cos(a), Y: math.Sin(a)}
		dm.EarSites[i] = m2fastener.Site{
			Head:   u.Scale(onewayclutch.EarCircle),
			Dir:    r3.Vec{Z: -1},
			Corner: u,
			Length: EarBoltLength,
			Cavity: EarBoltCavity,
		}
		if err := dm.EarSites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("turbinefan: ear bolt %d: %w", i+1, err)
		}
	}
	return dm, nil
}

// deriveSetScrew fills the set screw's values and checks them: a ShaftMount fan only.
func (dm *Dimensions) deriveSetScrew(p Params) error {
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
		return fmt.Errorf("turbinefan: set screw: %w", err)
	}
	if dm.SetScrewHead > math.Sqrt(HubRadius*HubRadius-m2bolt.HeadDiameter*m2bolt.HeadDiameter/4) {
		return errors.New("turbinefan: set screw: its head would stand out of the round face")
	}
	if dm.SetScrewSite.Cavity-m2fastener.HeadPocketDepth-SetScrewTravel < 1-1e-9 {
		return errors.New("turbinefan: set screw: less than 1 mm between its counterbore and its nut slot")
	}
	if SetScrewZ-m2fastener.HeadPocketRadius < 1 {
		return errors.New("turbinefan: set screw: less than 1 mm between its counterbore and the back face")
	}
	return nil
}
