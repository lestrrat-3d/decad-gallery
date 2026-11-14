package planetarygearbox

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Params are the gearbox inputs. Lengths are in millimeters and angles in
// radians. Zero fields are not defaults; use DefaultParams for those.
type Params struct {
	// Module is the tooth module of all three gears.
	Module float64
	// PressureAngle is the pressure angle of all three gears.
	PressureAngle float64
	// SunTeeth, PlanetTeeth and RingTeeth are the tooth counts. RingTeeth must be
	// SunTeeth + 2*PlanetTeeth.
	SunTeeth, PlanetTeeth, RingTeeth int
	// PlanetCount is the number of planets in each stage.
	PlanetCount int
	// Stages is the number of identical stages stacked in the one ring.
	Stages int
	// Samples is the number of fit points per tooth flank.
	Samples int
	// GearThickness is the axial thickness of a stage's sun and planets.
	GearThickness float64
	// CarrierThickness is the axial thickness of a carrier plate.
	CarrierThickness float64
	// StageGap is the axial gap between a stage's gears, its carrier, the next
	// stage and the ring ends.
	StageGap float64
	// CoverThickness is the axial thickness of each end cover's frame, which the bearing boss stands on.
	CoverThickness float64
	// BoltHoleDiameter is the diameter of the four clearance holes that join the covers and the ring.
	BoltHoleDiameter float64
	// FlatDepth is the depth of the flat that gives each shaft a D section where a sun sits on it, and gives each
	// thick part (a carrier pin, a link shaft, the output shaft) its D section in the carrier. The sun and the carrier
	// have a matching D bore, so the shaft carries torque into them. The shaft is a full round cylinder where a bearing
	// sits. It must stay under the shaft collar margin, 1.2. Zero leaves round shafts and round bores.
	FlatDepth float64
	// SeatChamfer is the setback of the chamfer on the entrance of each bearing seat, the planets' pockets and the
	// covers' boss pockets, so a bearing can be pressed into a tight printed seat. Zero leaves the entrance sharp.
	SeatChamfer float64
	// EdgeChamfer is the setback of the 45° chamfer on the outer and window edges of the large flat faces of the ring,
	// the covers and the carriers. It must be at least MinEdgeChamfer.
	EdgeChamfer float64
	// InputShaftLength is how far the input shaft reaches out of the front cover's bearing boss.
	InputShaftLength float64
	// BearingClearance is the radial clearance between each purchased bearing and its pocket.
	BearingClearance float64
	// PlanetBearing names the catalog bearing (models/ballbearing) each planet turns on, e.g. "605ZZ" or "608ZZ".
	// Each carrier pin is as wide as its bore.
	PlanetBearing string
	// ShaftBearing names the catalog bearing each cover holds for the input or output shaft. Every shaft is as wide as
	// its bore where it passes a bearing or a sun.
	ShaftBearing string
	// CarrierWall is the width of carrier material around the shaft and around each planet pin.
	CarrierWall float64
	// RingWall is the radial wall of the ring behind the tooth roots. It carries the bolt holes.
	RingWall float64
	// Backlash is how much each tooth of the sun and the planets is thinned at the pitch
	// circle. The ring's tooth spaces are widened by the same amount, so a sun-planet pair and
	// a planet-ring pair each have twice Backlash of play.
	Backlash float64
	// OutputShaftLength is how far the output shaft reaches out of the rear cover's bearing boss.
	OutputShaftLength float64
	// EarCount is the number of mounting ears on the ring. Zero leaves the ring a plain cylinder.
	EarCount int
	// EarBoreDiameter is the diameter of the through hole in each ear, parallel to the gearbox axis. The ear is
	// sized from it, so a larger bore makes a larger ear. The model adds no fastener, head pocket or nut cavity:
	// the user chooses what passes through the ear.
	EarBoreDiameter float64
	// Placement is the reference point and plane. Nil selects the world XY frame
	// at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the 13/29/71 three-planet gearbox with one stage.
func DefaultParams() Params {
	return Params{
		Module:            1,
		PressureAngle:     20 * math.Pi / 180,
		SunTeeth:          13,
		PlanetTeeth:       29,
		RingTeeth:         71,
		PlanetCount:       3,
		Stages:            1,
		Samples:           15,
		GearThickness:     8,
		CarrierThickness:  3,
		StageGap:          1,
		CoverThickness:    3,
		BoltHoleDiameter:  minBoltHoleDiameter,
		FlatDepth:         0.5,
		SeatChamfer:       0.5,
		EdgeChamfer:       MinEdgeChamfer,
		InputShaftLength:  12,
		BearingClearance:  seatClearance,
		PlanetBearing:     "605ZZ",
		ShaftBearing:      "605ZZ",
		CarrierWall:       3,
		RingWall:          7.1,
		Backlash:          0.15,
		OutputShaftLength: 12,
		EarCount:          3,
		EarBoreDiameter:   4,
	}
}

// Ratio returns the reduction from the first sun to the last carrier with the
// ring held fixed.
func (p Params) Ratio() float64 {
	stage := 1 + float64(p.RingTeeth)/float64(p.SunTeeth)
	return math.Pow(stage, float64(p.Stages))
}

// planetBearing is the catalog bearing each planet turns on. It is the zero Spec when p names no catalog bearing.
func planetBearing(p Params) ballbearing.Spec {
	b, _ := ballbearing.Lookup(p.PlanetBearing)
	return b
}

// shaftBearing is the catalog bearing each cover holds for the input or output shaft. It is the zero Spec when p
// names no catalog bearing.
func shaftBearing(p Params) ballbearing.Spec {
	b, _ := ballbearing.Lookup(p.ShaftBearing)
	return b
}

// bearingNameError returns the error for a bearing name outside the catalog, or nil when the name is in it.
func bearingNameError(role, name string) error {
	if _, ok := ballbearing.Lookup(name); ok {
		return nil
	}
	names := make([]string, 0, len(ballbearing.Catalog()))
	for _, s := range ballbearing.Catalog() {
		names = append(names, s.Name)
	}
	return fmt.Errorf("%s bearing %q is not in the bearing catalog; choose one of %s", role, name, strings.Join(names, ", "))
}

const (
	gearRootModuleClearanceFactor = 0.25
	looseGearClearance            = 0.5
	gearRootMeasurementMargin     = 0.02
)

// gearRootClearance returns the minimum clearance required between a gear root and a mating tip.
func gearRootClearance(module float64) float64 {
	return math.Max(gearRootModuleClearanceFactor*module, looseGearClearance)
}

func gearRootDepth(module float64) float64 { return module + gearRootClearance(module) }

// lipWidth is the radial width of the lip that holds a bearing in its pocket: the lip bore is this much
// smaller than the bearing's outside radius. lipThickness is the axial thickness of that lip.
const (
	lipWidth     = 2
	lipThickness = 1
	// seatClearance is the radial clearance between a purchased bearing and the bore that holds it fixed.
	seatClearance = ballbearing.SeatClearance
	// fitClearance is the smallest surface gap used by adjustable gearbox fits.
	fitClearance = 0.25
	// planetBearingLift raises each planet bearing above its pocket floor.
	planetBearingLift = fitClearance
	// pinShoulderClearance keeps each carrier pin's thick part above the planet bearing's inner race.
	pinShoulderClearance = fitClearance
	// pinShoulderReach is the shortest length of a pin's thick part below the carrier plate, so it prints sound.
	pinShoulderReach = fitClearance
	// collarBearingGap is the Tight gap between the input shaft's collar and the front bearing's inner face.
	collarBearingGap = fitClearance
	// collarClearance is the gap between a shaft collar and the sun or carrier it faces. It keeps the collar's
	// face off the plane where the flat ends, and gives the printed parts a little play.
	collarClearance = 0.5
	// pressFitClearance is the radial gap between a thick D part (a carrier pin, a link shaft or the output shaft) and
	// its socket or hole in the carrier plate, and between a shaft and the D bore of the sun it turns. It is a press
	// fit, tighter than the 0.2 mm Tight minimum of Part clearances, because the part stays in the plate or the sun
	// turns with the shaft.
	pressFitClearance = ballbearing.PressFitClearance
	// stepOverlap is how far a thin rod reaches into its thick part, so the union crosses faces.
	stepOverlap = 0.05
	// minSocketWall is the plate material kept around a pin socket at mid-plate.
	minSocketWall = 1.0
	// minSocketFaceWall is the plate material kept between a pin socket and the chamfered outline at a face.
	minSocketFaceWall = 0.5
	// boltRimMargin is the material kept between a bolt hole and the edge of the ring or cover rim.
	boltRimMargin = 1.5
	// armMargin is how much wider than a pin each carrier arm and cover spoke is on either side, so the pin
	// is not tangent to the arm edges.
	armMargin = 1
	// collarMargin is how far a shaft collar reaches past the shaft on each side. A collar is the printed
	// stand-in for a washer: it spans the gap between a bearing and the part that would rub on it, and it
	// must stay narrower than its bearing's InnerRaceDiameter (see ballbearing.Spec).
	collarMargin = 1.2
	// coverSpokes is the number of spokes joining a cover's rim to its bearing hub.
	coverSpokes = 3
)

// dims are the derived dimensions in millimeters.
type dims struct {
	orbit, planetRoot, planetTip float64
	ringTip, ringRoot, ringOuter float64
	pin, pinCollar               float64
	planetPocket, planetLip      float64
	boss, arm, collar            float64
	shaft, shaftBore             float64
	coverPocket, coverLip        float64
	coverBoss, coverHub          float64
	boltCircle, rimInner         float64
	stagePitch, ringLength       float64
}

// derive computes the derived dimensions of p without validating p.
func derive(p Params) dims {
	m := p.Module
	d := dims{
		orbit:      m * float64(p.SunTeeth+p.PlanetTeeth) / 2,
		planetRoot: m*float64(p.PlanetTeeth)/2 - gearRootDepth(m) - gearRootMeasurementMargin,
		planetTip:  (m*float64(p.PlanetTeeth)+2*m)/2 - gearRootMeasurementMargin,
		ringTip:    m*float64(p.RingTeeth)/2 - m,
		ringRoot:   m*float64(p.RingTeeth)/2 + gearRootDepth(m),
		stagePitch: p.GearThickness + p.CarrierThickness + 2*p.StageGap,
	}
	pb, sb := planetBearing(p), shaftBearing(p)
	d.pin = pb.ShaftDiameter()
	d.pinCollar = d.pin + 2*collarMargin
	d.planetPocket = pb.OuterDiameter + 2*p.BearingClearance
	d.planetLip = pb.OuterDiameter - 2*lipWidth
	d.shaft = sb.ShaftDiameter()
	d.shaftBore = d.shaft + 2*pressFitClearance
	d.collar = d.shaft + 2*collarMargin
	d.coverPocket = sb.OuterDiameter + 2*p.BearingClearance
	d.coverLip = sb.OuterDiameter - 2*lipWidth
	d.coverBoss = d.coverPocket + 2*p.CarrierWall
	d.coverHub = d.coverBoss + 2*p.CarrierWall
	d.boss = d.pin + 2*p.CarrierWall
	d.arm = d.pin + 2*armMargin
	d.ringOuter = d.ringRoot + p.RingWall
	d.boltCircle = (d.ringRoot + d.ringOuter) / 2
	d.rimInner = rimInnerRadius(d.boltCircle)
	d.ringLength = float64(p.Stages)*d.stagePitch + p.StageGap
	return d
}

// rimInnerRadius is the inner radius of a cover's rim: the bolt circle radius less the head pocket radius and
// the material kept around the pocket.
func rimInnerRadius(boltCircle float64) float64 {
	return boltCircle - headPocketDiameter()/2 - boltRimMargin
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Validate applies the input checks of the construction contract. It returns the first of Problems(p).
func (p Params) Validate() error {
	if probs := Problems(p); len(probs) > 0 {
		return errors.New(probs[0])
	}
	return nil
}
