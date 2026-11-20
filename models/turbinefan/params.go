package turbinefan

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). Values that another model owns are referenced, not
// redefined.
const (
	BladeCount = 3 // the requirement: three replaceable blades

	FirstBladeAngle = math.Pi / 2 // blade 1 along +Y; the set screw lies along -Y, midway between blades 2 and 3

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
	ClutchTurn = math.Pi                             // the fan frame's turn about Z from the clutch frame: the bore's flat (-Y) meets the shaft's (+Y)
	ClutchSeam = onewayclutch.Seam                   // 0.25
	MountZ     = onewayclutch.AdapterTo + ClutchSeam // 26.25: the fan frame's origin in the clutch frame

	// ClutchAnchorZ is the clutch cover's outer face in the fan frame (-7.75): the placement point of a ClutchMount fan.
	ClutchAnchorZ = onewayclutch.CoverTo - MountZ
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

// EarBoltAngle returns the angle about +Z from +X, in the fan frame, of ear bolt k, k = 1, 2, 3: 270, 30 and 150 degrees.
// The hub's two adapter holes lie at ear bolts 2 and 3 (30 and 150 degrees); ear bolt 1 (270) has no hole.
func EarBoltAngle(k int) float64 {
	return math.Mod(onewayclutch.EarAngles[k-1]*math.Pi/180-ClutchTurn+2*math.Pi, 2*math.Pi)
}

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
	// ClutchMount bolts the hub's back face to the adapter ring of a onewayclutch; the hub holds no shaft and has no set
	// screw.
	ClutchMount
)

// Params specifies a fan in mm and radians. The shaft runs in a purchased bearing, so its diameter equals the bearing's
// bore.
//
// Placement puts the fan's placement point on a frame. The placement point is the hub's front face (fan z = Size.HubLength)
// in ShaftMount, with the hub extending Size.HubLength back along the frame's axis, and the clutch cover's outer face
// (fan z = ClutchAnchorZ) in ClutchMount, with the clutch extending back and the adapter and fan in front. The point is
// on the shaft axis. A nil Placement leaves every part in the local frame, whose origin is the hub's back face.
type Params struct {
	// ShaftDiameter is the diameter of the D-flat shaft the hub grips; it equals the bearing bore.
	ShaftDiameter float64
	// FlatDepth is how deep the shaft's flat cuts below its round.
	FlatDepth float64
	// BoreClearance is the radial clearance of the hub's D bore round the shaft, on the round and on the flat.
	BoreClearance float64
	// RotorDiameter is the rotor's diameter across the blade tips, from MinRotorDiameter to MaxRotorDiameter. It sets
	// every size in Size.
	RotorDiameter float64
	// BladePitch is the blade angle at 75 % of the tip radius (48 mm at 128, 72 mm at 192), in radians, from
	// MinBladePitch to MaxBladePitch.
	BladePitch float64
	// Hand selects the blades' hand; the zero value is RightHand.
	Hand Hand
	// Mount selects how the hub is held; the zero value is ShaftMount.
	Mount     Mount
	Placement *r3.Frame
}

// anchorZ returns the fan-frame z of the placement point of m on a fan of size sz.
func (m Mount) anchorZ(sz Size) float64 {
	if m == ClutchMount {
		return ClutchAnchorZ
	}
	return sz.HubLength
}

// buildFrame returns the frame that moves the local frame to its place: Placement carried back from the placement
// point to the local origin. It is nil when Placement is nil. Placement must be valid.
func (p Params) buildFrame(sz Size) (*r3.Frame, error) {
	if p.Placement == nil {
		return nil, nil
	}
	xf, err := r3.FromFrame(*p.Placement)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: placement: %w", err)
	}
	f, err := r3.NewFrame(xf.Apply(r3.Vec{Z: -p.Mount.anchorZ(sz)}), xf.ApplyDir(r3.Vec{X: 1}), xf.ApplyDir(r3.Vec{Y: 1}))
	if err != nil {
		return nil, fmt.Errorf("turbinefan: placement: %w", err)
	}
	return &f, nil
}

// Params605ZZ returns a fan for the 5 mm shaft of a 605ZZ bearing, with the planetary gearbox shafts' 0.5 mm flat, at
// DefaultRotorDiameter.
func Params605ZZ() Params {
	return Params{
		ShaftDiameter: ballbearing.Params605ZZ().BoreDiameter,
		FlatDepth:     crankhandle.Params605ZZ().FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		RotorDiameter: DefaultRotorDiameter,
		BladePitch:    DefaultBladePitch,
	}
}

// Params608ZZ returns a fan for the 8 mm shaft of a 608ZZ bearing, with a 0.75 mm flat, at DefaultRotorDiameter. A
// 608-2RS has the same bore.
func Params608ZZ() Params {
	return Params{
		ShaftDiameter: ballbearing.Params608ZZ().BoreDiameter,
		FlatDepth:     crankhandle.Params6082RS().FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		RotorDiameter: DefaultRotorDiameter,
		BladePitch:    DefaultBladePitch,
	}
}

// MountFrame returns the Placement that seats a ClutchMount fan on a clutch built with Placement clutch (nil selects
// the clutch's local frame). Its origin is the clutch cover's outer face, on the axis.
func MountFrame(clutch *r3.Frame) (r3.Frame, error) {
	sin, cos := math.Sincos(ClutchTurn)
	local, err := r3.NewFrame(r3.Vec{Z: onewayclutch.CoverTo}, r3.Vec{X: cos, Y: sin}, r3.Vec{X: -sin, Y: cos})
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
// Freewheel drives forward, DefaultRotorDiameter (WithRotorDiameter sets another), the default pitch (WithBladePitch
// sets another), ballbearing.PressFitClearance, and Placement MountFrame(c.Placement).
func ParamsForClutch(c onewayclutch.Params, opts ...ClutchOption) (Params, error) {
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
	out := Params{
		ShaftDiameter: dm.Shaft.Diameter,
		FlatDepth:     dm.Shaft.FlatDepth,
		BoreClearance: ballbearing.PressFitClearance,
		RotorDiameter: DefaultRotorDiameter,
		BladePitch:    DefaultBladePitch,
		Hand:          hand,
		Mount:         ClutchMount,
		Placement:     &f,
	}
	for _, o := range opts {
		o.applyClutch(&out)
	}
	if err := out.Validate(); err != nil {
		return Params{}, err
	}
	return out, nil
}

// ClutchOption changes what ParamsForClutch returns.
type ClutchOption interface {
	applyClutch(*Params)
}

type bladePitchOption float64

func (o bladePitchOption) applyClutch(p *Params) { p.BladePitch = float64(o) }

// WithBladePitch sets the fan's BladePitch, in radians, in place of DefaultBladePitch. ParamsForClutch returns an error
// when the pitch lies outside [MinBladePitch, MaxBladePitch].
func WithBladePitch(pitch float64) ClutchOption { return bladePitchOption(pitch) }

type rotorDiameterOption float64

func (o rotorDiameterOption) applyClutch(p *Params) { p.RotorDiameter = float64(o) }

// WithRotorDiameter sets the fan's RotorDiameter, in mm, in place of DefaultRotorDiameter. ParamsForClutch returns an
// error when the diameter lies outside [MinRotorDiameter, MaxRotorDiameter].
func WithRotorDiameter(d float64) ClutchOption { return rotorDiameterOption(d) }

// DefaultParams returns Params605ZZ.
func DefaultParams() Params { return Params605ZZ() }

// Dimensions are the values Derive computes from Params, in mm, in the local frame. rho is a distance from the shaft
// axis along -Y.
type Dimensions struct {
	// Size holds the hub, blade and blade joint dimensions that follow the rotor diameter.
	Size Size
	// SetScrewBolt is the set screw's M2 length: the longest standard length whose head stays inside the round face.
	SetScrewBolt     float64
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
	// AdapterSites are the two adapter bolts, local frame: each head's top face lies on the adapter's rear face, one
	// adapter thickness and one seam behind the hub's back face, and the bolt runs along +Z into a nut slot open to
	// the hub's round face. The hub carries their holes and slots in both modes, at 30 and 150 degrees.
	AdapterSites [2]m2fastener.Site
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
		{"BladePitch", p.BladePitch}, {"RotorDiameter", p.RotorDiameter},
	} {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) || f.value <= 0 {
			return Dimensions{}, fmt.Errorf("turbinefan: %s must be finite and positive", f.name)
		}
	}
	sz, err := SizeFor(p.RotorDiameter)
	if err != nil {
		return Dimensions{}, err
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
	dm := Dimensions{
		Size:       sz,
		BoreRadius: p.ShaftDiameter/2 + p.BoreClearance,
		BoreFlat:   p.ShaftDiameter/2 - p.FlatDepth + p.BoreClearance,
	}
	dm.BoreFlatWidth = 2 * math.Sqrt(dm.BoreRadius*dm.BoreRadius-dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + dm.BoreFlatWidth
	dm.ShaftFlatWidth = 2 * math.Sqrt(p.FlatDepth*(p.ShaftDiameter-p.FlatDepth))
	if sz.BladeSocketStart-dm.BoreRadius < PressWall-1e-9 {
		return Dimensions{}, fmt.Errorf("turbinefan: ShaftDiameter leaves less than a %.1f mm wall to the blade sockets",
			PressWall)
	}
	if dm.ShaftFlatWidth < m2fastener.BoltSize.ShankDiameter-1e-9 {
		return Dimensions{}, errors.New("turbinefan: FlatDepth must leave the shaft's flat at least 2 mm wide for the set screw's tip")
	}
	if dm.BoreFlatWidth < 2*(m2fastener.HoleRadius+0.1)-1e-9 {
		return Dimensions{}, errors.New("turbinefan: FlatDepth must leave the bore's flat at least 2.8 mm wide for the set screw's hole")
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
	if err := dm.deriveSetScrew(p); err != nil {
		return Dimensions{}, err
	}
	for i := range dm.AdapterSites {
		a := EarBoltAngle(i + 2)
		u := r3.Vec{X: math.Cos(a), Y: math.Sin(a)}
		dm.AdapterSites[i] = m2fastener.Site{
			Head:   u.Scale(onewayclutch.FaceCircle).Add(r3.Vec{Z: -(onewayclutch.AdapterThickness + ClutchSeam)}),
			Dir:    r3.Vec{Z: 1},
			Corner: u,
			Length: onewayclutch.FaceLength,
			Cavity: onewayclutch.AdapterThickness + ClutchSeam + onewayclutch.FaceNutDepth,
		}
		if err := dm.AdapterSites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("turbinefan: adapter bolt %d: %w", i+1, err)
		}
	}
	for i := range dm.BladeSites {
		u := r3.Vec{X: math.Cos(BladeAngle(i + 1)), Y: math.Sin(BladeAngle(i + 1))}
		dm.BladeSites[i] = m2fastener.Site{
			Head:   u.Scale(sz.BladeBoltAt).Add(r3.Vec{Z: sz.HubLength}),
			Dir:    r3.Vec{Z: -1},
			Corner: u,
			Length: sz.BladeBoltLength,
			Cavity: sz.BladeBoltCavity,
		}
		if err := dm.BladeSites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("turbinefan: blade bolt %d: %w", i+1, err)
		}
	}
	return dm, nil
}

// deriveSetScrew fills the set screw's values and checks them. The hub carries the set screw's cuts in both modes.
func (dm *Dimensions) deriveSetScrew(p Params) error {
	dm.SetScrewTip = p.ShaftDiameter/2 - p.FlatDepth - p.BoreClearance
	// The head must stay inside the round face: rho_h <= sqrt(HubRadius^2 - (HeadDiameter/2)^2).
	reach := math.Sqrt(dm.Size.HubRadius*dm.Size.HubRadius - m2fastener.BoltSize.SocketCap.HeadDiameter*m2fastener.BoltSize.SocketCap.HeadDiameter/4)
	for _, l := range m2fastener.BoltSize.StandardLengths {
		if dm.SetScrewTip+m2fastener.HeadPocketDepth+l <= reach {
			dm.SetScrewBolt = l
		}
	}
	if dm.SetScrewBolt == 0 {
		return errors.New("turbinefan: set screw: no standard length keeps its head inside the round face")
	}
	dm.SetScrewHead = dm.SetScrewTip + m2fastener.HeadPocketDepth + dm.SetScrewBolt
	dm.CounterboreFloor = dm.SetScrewHead - m2fastener.HeadPocketDepth - SetScrewTravel
	dm.SetScrewSite = m2fastener.Site{
		Head:   r3.Vec{Y: -dm.SetScrewHead, Z: SetScrewZ},
		Dir:    r3.Vec{Y: 1},
		Corner: r3.Vec{Z: 1},
		Length: dm.SetScrewBolt,
		Cavity: dm.SetScrewHead - dm.BoreFlat - PressWall - m2fastener.NutCavityHeight,
	}
	if err := dm.SetScrewSite.Validate(); err != nil {
		return fmt.Errorf("turbinefan: set screw: %w", err)
	}
	if dm.SetScrewSite.Cavity-m2fastener.HeadPocketDepth-SetScrewTravel < 1-1e-9 {
		return errors.New("turbinefan: set screw: less than 1 mm between its counterbore and its nut slot")
	}
	if SetScrewZ-m2fastener.HeadPocketRadius < 1 {
		return errors.New("turbinefan: set screw: less than 1 mm between its counterbore and the back face")
	}
	return nil
}
