// Package screwgear builds a pair of crossed, twisted-rack screw gears and the sleeve that holds them.
package screwgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

// Params is the whole mechanism: the 19 numeric dialog inputs of the spec's Variables table
// plus the placement that stands in for its plane and point selections.
// Lengths are millimeters and angles are radians. The field names follow the dialog
// labels; the comment on each gives the spec's input id, which the error messages use.
type Params struct {
	// Ribbon group.
	Width       float64 // W, across the plate (ribbonWidth)
	ToothCount  int     // N (toothCount)
	TwistLead   float64 // advance per full turn (twistLead)
	Thickness   float64 // T, through the plate (ribbonThickness)
	ToothPitch  float64 // P, along the axis (toothPitch)
	ToothHeight float64 // H, crest to root (toothHeight)
	// ToothSlant leans the tooth ridges across the thickness (toothSlant).
	// Zero is the straight ridge; the sign is the pair frame's.
	ToothSlant float64
	// ToothBow lowers the toothed edge by ToothBow*v^2, in mm per mm^2 (toothBow).
	ToothBow float64

	// Frame group.
	CageRadius float64 // where each bore sits on its gear's axis (cageRadius)
	CageRise   float64 // half the sleeve's height, to its flat end faces (cageRise)
	Clearance  float64 // added all round every bore (clearance)
	// RoofAllowance is added to the clearance on the roof face of every bore,
	// which the printer bridges (roofAllowance).
	RoofAllowance float64
	CollarHalf    float64 // half the sleeve's wall thickness (collarHalf)
	CollarWall    float64 // the least material left round a bore or window (collarWall)

	// Mesh group, from the meshing search.
	CrossAngle  float64 // Sigma, the angle between the two axes (crossAngle)
	Engagement  float64 // how deep the crests overlap (engagement)
	MountAngleA float64 // Phi for gear A (mountAngleA)
	MountAngleB float64 // Phi for gear B (mountAngleB)
	// AssemblyPhase is the tooth phase gear B is built at with gear A at zero (assemblyPhase).
	AssemblyPhase float64

	// Placement gives the mechanism's centre C (its origin), the common perpendicular
	// n̂ (its normal) and the reference direction ê (its U axis). Nil selects the
	// world XY frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the spec's default table.
func DefaultParams() Params {
	return Params{
		Width:       15,
		ToothCount:  68,
		TwistLead:   49.5,
		Thickness:   3.75,
		ToothPitch:  2.625,
		ToothHeight: 2.625,
		ToothSlant:  25.8 * math.Pi / 180,
		ToothBow:    0.048,

		CageRadius:    15.5,
		CageRise:      18.75,
		Clearance:     0.20,
		RoofAllowance: 0.60,
		CollarHalf:    3,
		CollarWall:    3,

		CrossAngle:    80 * math.Pi / 180,
		Engagement:    1.15,
		MountAngleA:   0,
		MountAngleB:   0,
		AssemblyPhase: -1.31,
	}
}

// Lambda is the screw parameter: millimeters of advance per radian of turn.
func (p Params) Lambda() float64 { return p.TwistLead / (2 * math.Pi) }

// Beta is the helix angle of the toothed edge, which sits at radius Width/2.
func (p Params) Beta() float64 { return math.Atan(math.Pi * p.Width / p.TwistLead) }

// AxisOffset is A, the distance between the two axes.
func (p Params) AxisOffset() float64 { return p.Width - p.Engagement }

// Length is the finished ribbon's length.
func (p Params) Length() float64 { return float64(p.ToothCount) * p.ToothPitch }

// Travel is how far the mechanism runs. A gear has to keep both its bores full,
// so a ribbon runs Length minus twice the bore's far face through its bores.
func (p Params) Travel() float64 { return p.Length() - 2*(p.CageRadius+p.CollarHalf) }

// AxialWindow is how far along its own axis a gear can still reach the other,
// with half again as slack. A point of gear A at station s stands
// sqrt(A^2 + (s*sin Sigma)^2) from gear B's axis, and nothing further than Width from
// that axis can touch a ribbon of that width.
func (p Params) AxialWindow() float64 {
	a := p.AxisOffset()
	if a >= p.Width {
		return p.Width
	}
	return 1.5 * math.Sqrt(p.Width*p.Width-a*a) / math.Sin(p.CrossAngle)
}

// SleeveInner is Ri, the sleeve's inner radius.
func (p Params) SleeveInner() float64 { return p.CageRadius - p.CollarHalf }

// SleeveOuter is Ro, the sleeve's outer radius.
func (p Params) SleeveOuter() float64 { return p.CageRadius + p.CollarHalf }

// BoreHalfWidth is hw: the crest rectangle's half width plus the clearance.
func (p Params) BoreHalfWidth() float64 { return p.Width/2 + p.Clearance }

// BoreHalfThickness is ht: the crest rectangle's half thickness plus the clearance.
// The roof allowance is added on top of it on one face of each bore (see boreOpening).
func (p Params) BoreHalfThickness() float64 { return p.Thickness/2 + p.Clearance }

// BoreCorner is c: how far the furthest corner of any bore stands from its axis.
// The roof face stands RoofAllowance further across the thickness than the other,
// so its two corners are the furthest.
func (p Params) BoreCorner() float64 {
	return math.Hypot(p.BoreHalfWidth(), p.BoreHalfThickness()+p.RoofAllowance)
}

// sleeveCutMargin is how far each bore's cut runs past the material at both ends,
// so the cut starts and finishes in air.
const sleeveCutMargin = 1

// SleeveCut returns sIn and sOut. The +R bore's cut spans [sIn, sOut] of its gear's axis
// and the -R bore's spans [-sOut, -sIn]. The inner face is a cylinder, so the channel's
// corner reaches into the wall before its centre line does: sqrt(Ri^2 - c^2) is the
// station at which the corner first touches it, and a millimeter before that the whole
// section is in the hollow. A millimeter past Ro the whole section is outside.
func (p Params) SleeveCut() (float64, float64) {
	ri, c := p.SleeveInner(), p.BoreCorner()
	return math.Sqrt(ri*ri-c*c) - sleeveCutMargin, p.SleeveOuter() + sleeveCutMargin
}

// meshFootprintRadius bounds how far from the frame's axis the mesh zone reaches:
// a point within AxialWindow of the crossing is at most that far along the axis, which
// is radial in projection, and at most half the crest rectangle's diagonal across it.
func (p Params) meshFootprintRadius() float64 {
	return math.Hypot(p.AxialWindow(), math.Hypot(p.Width/2, p.Thickness/2))
}

// placementFrame returns the placement, or the world XY frame when it is nil.
func (p Params) placementFrame() (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// finite is the numeric fields as named values, for the finiteness check.
func (p Params) finite() []struct {
	name  string
	value float64
} {
	return []struct {
		name  string
		value float64
	}{
		{"ribbonWidth", p.Width}, {"ribbonThickness", p.Thickness}, {"toothPitch", p.ToothPitch},
		{"twistLead", p.TwistLead}, {"collarHalf", p.CollarHalf}, {"collarWall", p.CollarWall},
		{"clearance", p.Clearance}, {"roofAllowance", p.RoofAllowance}, {"toothHeight", p.ToothHeight},
		{"toothSlant", p.ToothSlant}, {"toothBow", p.ToothBow}, {"engagement", p.Engagement},
		{"crossAngle", p.CrossAngle}, {"assemblyPhase", p.AssemblyPhase}, {"cageRadius", p.CageRadius},
		{"cageRise", p.CageRise}, {"mountAngleA", p.MountAngleA}, {"mountAngleB", p.MountAngleB},
	}
}

// Validate applies the spec's range checks in the spec's order: the ribbon's own sizes,
// the tooth, the mesh, the bore range, then the sleeve's four checks. It returns the first
// failure, naming the offending input and its bound.
func (p Params) Validate() error {
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("screwgear: invalid placement frame")
	}
	for _, f := range p.finite() {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) {
			return fmt.Errorf("screwgear: %s %g must be finite", f.name, f.value)
		}
	}
	if err := p.validateRibbon(); err != nil {
		return err
	}
	if err := p.validateMesh(); err != nil {
		return err
	}
	return p.validateSleeve()
}

// validateRibbon checks the ribbon's sizes and its tooth.
func (p Params) validateRibbon() error {
	for _, f := range []struct {
		name  string
		value float64
	}{
		{"ribbonWidth", p.Width}, {"ribbonThickness", p.Thickness}, {"toothPitch", p.ToothPitch},
		{"twistLead", p.TwistLead}, {"collarHalf", p.CollarHalf}, {"collarWall", p.CollarWall},
		{"clearance", p.Clearance},
	} {
		if f.value <= 0 {
			return fmt.Errorf("screwgear: %s %g must be greater than 0", f.name, f.value)
		}
	}
	if p.RoofAllowance < 0 {
		return fmt.Errorf("screwgear: roofAllowance %g must not be negative", p.RoofAllowance)
	}
	if p.ToothCount < 4 {
		return fmt.Errorf("screwgear: toothCount %d must be a whole number of at least 4", p.ToothCount)
	}
	if p.ToothHeight <= 0 || p.ToothHeight >= p.Width/2 {
		return fmt.Errorf("screwgear: toothHeight %g must be greater than 0 and less than ribbonWidth/2 = %g",
			p.ToothHeight, p.Width/2)
	}
	if p.ToothSlant <= -math.Pi/2 || p.ToothSlant >= math.Pi/2 {
		return fmt.Errorf("screwgear: toothSlant %g rad must lie strictly between -90 and 90 degrees", p.ToothSlant)
	}
	if p.ToothBow < 0 {
		return fmt.Errorf("screwgear: toothBow %g must not be negative", p.ToothBow)
	}
	if root := p.ToothHeight + p.ToothBow*(p.Thickness/2)*(p.Thickness/2); root >= p.Width/2 {
		return fmt.Errorf("screwgear: toothBow %g puts the root at the faces %g mm in, "+
			"which must stay less than ribbonWidth/2 = %g", p.ToothBow, root, p.Width/2)
	}
	return nil
}

// validateMesh checks the mesh group and the bore range.
func (p Params) validateMesh() error {
	if p.Engagement <= 0 || p.Engagement > p.ToothHeight {
		return fmt.Errorf("screwgear: engagement %g must be greater than 0 and at most toothHeight = %g",
			p.Engagement, p.ToothHeight)
	}
	if p.CrossAngle <= 0 || p.CrossAngle >= math.Pi {
		return fmt.Errorf("screwgear: crossAngle %g rad must lie strictly between 0 and 180 degrees", p.CrossAngle)
	}
	if math.Abs(p.AssemblyPhase) >= p.ToothPitch {
		return fmt.Errorf("screwgear: assemblyPhase %g must lie strictly within +-toothPitch = %g",
			p.AssemblyPhase, p.ToothPitch)
	}
	if reach := p.CageRadius + p.CollarHalf + sleeveCutMargin; reach >= p.Length()/2 {
		return fmt.Errorf("screwgear: cageRadius %g puts a bore's cut %g mm out, "+
			"which must stay less than half the ribbon length %g", p.CageRadius, reach, p.Length()/2)
	}
	return nil
}

// validateSleeve is the sleeve's four checks, in order. The first three are closed
// forms; the fourth samples the bores' channels.
func (p Params) validateSleeve() error {
	ri := p.SleeveInner()
	// Each cut starts sleeveCutMargin before the channel's corner reaches the inner face,
	// at sqrt(ri^2 - c^2) - sleeveCutMargin along its axis. That has to be past the middle,
	// or the two cuts of one gear overlap.
	if reach := math.Hypot(p.BoreCorner(), sleeveCutMargin); !(reach < ri) {
		return fmt.Errorf("screwgear: cageRadius %g: the bore's corner %g mm, with the cut's margin, "+
			"must stay inside the sleeve's inner radius %g mm so the cut starts in the hollow",
			p.CageRadius, reach, ri)
	}
	if reach := p.meshFootprintRadius() + p.Clearance; !(reach <= ri) {
		return fmt.Errorf("screwgear: cageRadius %g: the mesh zone and its clearance reach %g mm, "+
			"which must not pass the sleeve's inner radius %g mm", p.CageRadius, reach, ri)
	}
	if need := p.AxisOffset()/2 + p.BoreCorner() + p.CollarWall; !(p.CageRise >= need) {
		return fmt.Errorf("screwgear: cageRise %g must be at least %g mm so the channels keep collarWall %g "+
			"from the end faces", p.CageRise, need, p.CollarWall)
	}
	pf, err := newPairFrame(p)
	if err != nil {
		return err
	}
	if sep, which := channelSeparation(newPlainSleeve(pf)); !(sep >= p.CollarWall) {
		return fmt.Errorf("screwgear: collarWall %g: %s come %g mm apart, which is less than collarWall",
			p.CollarWall, which, sep)
	}
	return nil
}
