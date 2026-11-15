package onewayclutch

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). Each fit gap is its class minimum from docs/model-pipeline.md
// "Part clearances" plus FitMargin.
const (
	FitMargin  = 0.05
	TightFit   = 0.2 + FitMargin  // 0.25: the hub to each plate boss
	RegularFit = 0.35 + FitMargin // 0.40: the head to its tooth gap
	LooseFit   = 0.5 + FitMargin  // 0.55: the arm to the tooth tips

	Pawls = 3  // pawls on the hub, 120 degrees apart
	Teeth = 24 // internal teeth of the ring, 15 degrees apart

	ToothTip    = 14.5 // ring tooth tip radius R_tip
	ToothRoot   = 17.0 // ring tooth root radius R_root
	RootRound   = 0.8  // ring tooth root round, equal to HeadGap + TipRound
	RingBand    = 13.5 // blank bore radius in the tooth band, all cut away by the tooth tool
	RingCounter = 18.5 // ring counterbore radius
	RingOuter   = 21.0 // ring outside radius (OD 42)
	RingChamfer = 0.5  // lead-in of the ring's outer loop and counterbores at both ends

	// The ring's ears: three full-height lugs on its outside, each with an M2 clearance bore along the axis.
	Ears        = 3    // ears on the ring
	EarCircle   = 23.5 // radius of the ear bores' centres
	EarRadius   = 3.6  // ear half width and end round radius
	EarRound    = 1.5  // round where each ear side meets the ring's outside
	EarReach    = EarCircle + EarRadius
	ToothEndGap = 0.85 // axial gap from each plate's inner face (and the hub's) to the tooth band's end

	CoreRadius   = 10.7 // hub core R_c
	SlotWidth    = 2.05 // slot under each arm s
	ArmThickness = 1.2  // arm t: three 0.4 mm lines, the bending-part minimum
	ArmInner     = CoreRadius + SlotWidth
	ArmOuter     = ArmInner + ArmThickness

	// WebAngle is the angle of pawl 0's web back face and SlotAngle that of its slot end, in the counter-clockwise layout.
	WebAngle  = -20.0
	SlotAngle = -6.0
	// LockAngle is the angle of the ring lock face that pawl 0 sits against.
	LockAngle = 90.0
	WebRound  = 1.0 // web root round
	TipRound  = 0.4 // head tip round
	HeadGap   = RegularFit

	HubHeight = 10.0 // hub and pawls
	Seam      = TightFit

	PlateRadius        = RingCounter - RegularFit // 18.1
	Recess             = RegularFit               // annulus below the boss face
	BackPlateThickness = 3.5
	CoverThickness     = 4.0
	PlateChamfer       = 0.5
	HubChamfer         = 0.5

	BoltCircle = 8.4
	BoltLength = 16.0
	PressWall  = 1.6 // least wall round a press-fit bore (AGENTS.md "Verification")

	// Axial stack, z along the shaft from the back plate's outer face.
	HubFrom    = BackPlateThickness + Seam  // 3.75
	CoverFrom  = HubFrom + HubHeight + Seam // 14.0
	CoverTo    = CoverFrom + CoverThickness // 18.0
	RingFrom   = 0.25
	RingTo     = CoverTo - 0.75 // 17.25, symmetric about the hub's mid-plane
	RingHeight = RingTo - RingFrom
	BandFrom   = BackPlateThickness - Recess + ToothEndGap // 3.95
	BandTo     = CoverFrom + Recess - ToothEndGap          // 13.55
)

// EarAngles are the ear axes' angles, in degrees. They are symmetric about x = 0, so the ears do not depend on the
// freewheel direction; each lies outside a tooth root in both directions.
var EarAngles = [Ears]float64{90, 210, 330}

// BoltAngles are the bolt axes' angles, in degrees. They are symmetric about x = 0, so the plates, bolts and nuts do not
// depend on the freewheel direction.
var BoltAngles = [3]float64{30, 150, 270}

// Direction is the way the ring turns freely about a held shaft, seen from the front (the cover).
type Direction int

const (
	// Clockwise is the zero-excluded first direction, so a forgotten field fails Validate.
	Clockwise Direction = iota + 1
	CounterClockwise
)

func (d Direction) String() string {
	switch d {
	case Clockwise:
		return "clockwise"
	case CounterClockwise:
		return "counter-clockwise"
	}
	return fmt.Sprintf("Direction(%d)", int(d))
}

// Sign returns -1 for Clockwise and +1 for CounterClockwise. Every hub and ring sketch is drawn in the counter-clockwise
// layout with each point (x, y) mapped to (Sign*x, y); any other Direction gives 0.
func (d Direction) Sign() float64 {
	switch d {
	case Clockwise:
		return -1
	case CounterClockwise:
		return 1
	}
	return 0
}

// Shaft is a preset D-flat shaft: its diameter and the depth of its flat below the round.
type Shaft struct {
	Name                string
	Diameter, FlatDepth float64
}

func shafts() []Shaft {
	diameter := func(bearing string) float64 {
		spec, _ := ballbearing.Lookup(bearing)
		return spec.ShaftDiameter()
	}
	return []Shaft{
		{Name: "605ZZ", Diameter: diameter("605ZZ"), FlatDepth: 0.5},
		{Name: "608ZZ", Diameter: diameter("608ZZ"), FlatDepth: 0.75},
		{Name: "608ZZ-gearbox", Diameter: diameter("608ZZ"), FlatDepth: 0.5},
	}
}

// Shafts returns a copy of the shaft presets, in table order.
func Shafts() []Shaft { return slices.Clone(shafts()) }

// LookupShaft returns the preset whose Name equals name exactly.
func LookupShaft(name string) (Shaft, bool) {
	for _, s := range shafts() {
		if s.Name == name {
			return s, true
		}
	}
	return Shaft{}, false
}

// Params specifies a clutch. A nil Placement selects the local frame.
type Params struct {
	// Shaft names a Shafts entry.
	Shaft string
	// Freewheel is the direction the ring turns freely, seen from the cover.
	Freewheel Direction
	// BoreClearance is the radial clearance of every D bore on the shaft, on the round and on the flat.
	BoreClearance float64
	Placement     *r3.Frame
}

// DefaultParams returns the 605ZZ shaft, clockwise freewheel, with a press-fit bore.
func DefaultParams() Params {
	return Params{Shaft: "605ZZ", Freewheel: Clockwise, BoreClearance: ballbearing.PressFitClearance}
}

// Dimensions are the values Derive computes from Params, in mm, in the local frame. The pawl values are in the
// counter-clockwise layout of pawl 0, before the direction's mirror.
type Dimensions struct {
	Shaft      Shaft
	Sign       float64
	BoreRadius float64 // r_b: Diameter/2 + BoreClearance
	BoreFlat   float64 // f_b: distance from the axis to the bore's flat, on the +Y side
	BoreLoop   float64 // length of the bore's outline: its arc the long way round plus its flat
	Sites      [3]m2fastener.Site

	Apex          r3.Vec  // A: where the head's lock line meets its flank line
	TipCenter     r3.Vec  // centre of the head's tip round
	HeadReach     float64 // outermost head radius
	HeadTravel    float64 // HeadReach - ToothTip
	FlankEnd      r3.Vec  // H_f: flank meets the arm outer arc
	LockEnd       r3.Vec  // H_l: lock line meets the arm inner arc
	RampAngle     float64 // direction of the ring ramp, and of the flank, in degrees
	FlankDistance float64 // distance of the flank line from the axis
	RampDistance  float64 // distance of the ring ramp line from the axis
}

// Validate reports the first problem Derive finds.
func (p Params) Validate() error {
	_, err := p.Derive()
	return err
}

// polar returns the point at radius r and angle deg degrees.
func polar(r, deg float64) r3.Vec {
	s, c := math.Sincos(deg * math.Pi / 180)
	return r3.Vec{X: r * c, Y: r * s}
}

// Derive validates p and computes its dimensions, checking in this order: Shaft, Freewheel, BoreClearance, the wall
// between the bore and the nut pockets, Placement, each bolt site.
func (p Params) Derive() (Dimensions, error) {
	shaft, ok := LookupShaft(p.Shaft)
	if !ok {
		names := make([]string, 0, 3)
		for _, s := range shafts() {
			names = append(names, s.Name)
		}
		return Dimensions{}, fmt.Errorf("onewayclutch: unknown shaft preset %q; want one of %s", p.Shaft,
			strings.Join(names, ", "))
	}
	if p.Freewheel != Clockwise && p.Freewheel != CounterClockwise {
		return Dimensions{}, errors.New("onewayclutch: Freewheel must be Clockwise or CounterClockwise")
	}
	c := p.BoreClearance
	if math.IsNaN(c) || math.IsInf(c, 0) || c < ballbearing.PressFitClearance || c > 0.5 {
		return Dimensions{}, fmt.Errorf("onewayclutch: BoreClearance must be finite, from %g to 0.5 mm",
			ballbearing.PressFitClearance)
	}
	dm := Dimensions{
		Shaft:      shaft,
		Sign:       p.Freewheel.Sign(),
		BoreRadius: shaft.Diameter/2 + c,
		BoreFlat:   shaft.Diameter/2 - shaft.FlatDepth + c,
	}
	if BoltCircle-m2fastener.NutCavityRadius-dm.BoreRadius < PressWall {
		return Dimensions{}, fmt.Errorf(
			"onewayclutch: BoreClearance %g leaves less than %.1f mm of wall between the bore and the nut pockets",
			c, PressWall)
	}
	half := math.Sqrt(dm.BoreRadius*dm.BoreRadius - dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + 2*half
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errors.New("onewayclutch: Placement must be a valid frame")
	}
	for i, a := range BoltAngles {
		dir := polar(1, a)
		pos := polar(BoltCircle, a)
		dm.Sites[i] = m2fastener.Site{
			Head:   r3.Vec{X: pos.X, Y: pos.Y, Z: CoverTo},
			Dir:    r3.Vec{Z: -1},
			Corner: r3.Vec{X: dir.X, Y: dir.Y},
			Length: BoltLength,
			Cavity: CoverTo - m2fastener.NutCavityHeight,
		}
		if err := dm.Sites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("onewayclutch: bolt %d: %w", i+1, err)
		}
	}
	dm.derivePawl()
	return dm, nil
}

// derivePawl computes pawl 0's head from the seated tooth gap bounded by the lock face at LockAngle and the previous
// tooth's ramp, from P (ToothTip at 15 degrees before the lock face) to Q (ToothRoot on the lock face).
func (dm *Dimensions) derivePawl() {
	pitch := 360.0 / Teeth
	p := polar(ToothTip, LockAngle-pitch)
	q := polar(ToothRoot, LockAngle)
	ramp := q.Sub(p)
	ramp = ramp.Scale(1 / ramp.Len())
	dm.RampAngle = math.Atan2(ramp.Y, ramp.X) * 180 / math.Pi
	// n is the ramp's unit normal pointing away from the axis.
	n := r3.Vec{X: ramp.Y, Y: -ramp.X}
	if n.Dot(p) < 0 {
		n = n.Scale(-1)
	}
	dm.RampDistance = n.Dot(p)
	dm.FlankDistance = dm.RampDistance - HeadGap
	// The lock line is x = HeadGap (for LockAngle 90 degrees).
	x := HeadGap
	dm.Apex = r3.Vec{X: x, Y: (dm.FlankDistance - n.X*x) / n.Y}
	dm.TipCenter = r3.Vec{X: x + TipRound, Y: (dm.FlankDistance - TipRound - n.X*(x+TipRound)) / n.Y}
	dm.HeadReach = dm.TipCenter.Len() + TipRound
	dm.HeadTravel = dm.HeadReach - ToothTip
	dm.LockEnd = r3.Vec{X: x, Y: math.Sqrt(ArmInner*ArmInner - x*x)}
	// H_f: the flank meets the circle ArmOuter; take the root nearer the apex.
	foot := n.Scale(dm.FlankDistance)
	along := math.Sqrt(ArmOuter*ArmOuter - dm.FlankDistance*dm.FlankDistance)
	a, b := foot.Add(ramp.Scale(along)), foot.Sub(ramp.Scale(along))
	dm.FlankEnd = a
	if b.Sub(dm.Apex).Len() < a.Sub(dm.Apex).Len() {
		dm.FlankEnd = b
	}
}
