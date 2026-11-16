package turbinefan

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). Values that another model owns are referenced, not
// redefined.
const (
	BladeCount = 3 // the requirement: three replaceable blades

	HubRadius = handcrankfan.HubRadius // 14, the hand-crank fan's hub: the blade's root and stud fix it
	HubLength = handcrankfan.HubLength // 20
	HubMid    = HubLength / 2          // the sockets', the blade bolts' line and the clamp bolt's plane

	FirstBladeAngle = math.Pi / 2 // blade 1 lies along +Y; the slit and the clamp tab lie along -Y, 270 degrees
	TabHalfWidth    = 9.0         // the clamp tab is 18 wide: head pocket 2 plus an M2 x 16 bolt
	TabRootR        = 1.0         // the tab's two root rounds: a larger round reaches the sockets of blades 2 and 3
	ClampBolt       = 16.0        // M2 x 16: HeadPocketDepth + 16 = 2 TabHalfWidth

	BoreFloor      = crankhandle.BoreFloor
	ChamferSize    = crankhandle.ChamferSize
	PressWall      = crankhandle.PressWall
	SlitWidth      = crankhandle.SlitWidth
	SlitInBore     = crankhandle.SlitInBore
	ClampSideRun   = crankhandle.ClampSideRun
	ClampCornerR   = crankhandle.ClampCornerR
	ClampEndR      = crankhandle.ClampEndR
	NutPocketRound = crankhandle.NutPocketRound
	RegularFit     = crankhandle.RegularFit // 0.40: the stud in its socket and the blades' clearance to the tab
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
	// ShaftDiameter is the diameter of the D-flat shaft the hub pinches; it equals the bearing bore.
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

// Dimensions are the values Derive computes from Params, in mm, in the local frame.
type Dimensions struct {
	BoreRadius float64 // r_b: ShaftDiameter/2 + BoreClearance
	BoreFlat   float64 // f_b: distance from the axis to the bore's flat, on the +Y side
	BoreLoop   float64 // length of the bore's outline in the back face: its arc the long way round plus its flat
	// RootCenter is the centre of the right root round, tangent to the disc and to the tab's right side. RootDisc is
	// where that round meets the disc. The left ones are the mirrors in x = 0. RootSide is the y where the round meets
	// the straight side.
	RootCenter, RootDisc r3.Vec
	RootSide             float64
	// ClampY is the clamp bolt's axis, ClampSideRun below the roots. SideEnd is where the tab's sides stop being
	// straight. Reach is how far the tail's end arc runs behind the shaft axis.
	ClampY, SideEnd, Reach float64
	// TailCorner and TailEnd are the centres of the right corner arc and of the end arc.
	TailCorner, TailEnd r3.Vec
	// ClampSite is the clamp bolt: its head's top face is flush with the +X side face of the tab at (TabHalfWidth,
	// ClampY, HubMid), and it runs along -X. Its Cavity is the nut pocket's floor, NutCavityHeight inside the -X side
	// face, so CavityFar and the bolt's tip both lie on that face.
	ClampSite m2fastener.Site
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
	half := math.Sqrt(dm.BoreRadius*dm.BoreRadius - dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + 2*half
	if handcrankfan.BladeSocketStart-dm.BoreRadius < PressWall-1e-9 {
		return Dimensions{}, fmt.Errorf("turbinefan: ShaftDiameter leaves less than a %.1f mm wall to the blade sockets",
			PressWall)
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
	w, rho := float64(TabHalfWidth), float64(TabRootR)
	dm.RootCenter = r3.Vec{X: w + rho, Y: -math.Sqrt((HubRadius+rho)*(HubRadius+rho) - (w+rho)*(w+rho))}
	dm.RootDisc = dm.RootCenter.Scale(HubRadius / (HubRadius + rho))
	dm.RootSide = dm.RootCenter.Y
	dm.ClampY = dm.RootSide - ClampSideRun
	dm.SideEnd = dm.ClampY - ClampSideRun
	dm.TailCorner, dm.TailEnd = crankhandle.TailCenters(w, dm.SideEnd)
	dm.Reach = crankhandle.TailReach(w, dm.SideEnd)
	dm.ClampSite = m2fastener.Site{
		Head:   r3.Vec{X: w, Y: dm.ClampY, Z: HubMid},
		Dir:    r3.Vec{X: -1},
		Corner: r3.Vec{Z: 1},
		Length: ClampBolt,
		Cavity: 2*w - m2fastener.NutCavityHeight,
	}
	if err := dm.ClampSite.Validate(); err != nil {
		return Dimensions{}, fmt.Errorf("turbinefan: clamp bolt: %w", err)
	}
	if dm.ClampSite.Tip() != dm.ClampSite.CavityFar() {
		return Dimensions{}, errors.New("turbinefan: clamp bolt: its tip must end flush with the -X side face")
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
