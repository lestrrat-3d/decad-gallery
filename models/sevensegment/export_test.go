package sevensegment

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// Internal dimensions that the physics tests read, so that they follow the model and not copies of its numbers.
const (
	NotchFloor      = notchFloor
	SlotEnd         = slotEnd
	TongueWidth     = tongueWidth
	PinAxisZ        = pinAxisZ
	SliderHeadTop   = sliderHeadTop
	TongueRootR     = tongueRootR
	TongueRootDip   = tongueRootDip
	RotorEdgeR      = rotorEdgeRound
	BladeAcross     = bladeAcross
	SlotRadius      = slotRadius
	JournalPinR     = JournalRadius - pivotJournalPlay
	JournalFriction = JournalRadius * pivotWedge
	PinStationAGD   = pinStationAGD
	PinStationBC    = pinStationBC
	PinStationEF    = pinStationEF

	PivotEndPlay   = pivotEndPlay
	PivotPinTip    = pivotPinTip
	PivotBlockFrom = pivotBlockFrom
	PivotBlockLong = pivotBlockLong

	ColumnTop        = columnTop
	GuideFrom        = guideFrom
	GuideTop         = guideTop
	LedgeFrom        = ledgeFrom
	RibHeight        = ribHeight
	FrontInnerY      = frontInnerY
	RearInnerY       = rearInnerY
	FrontWallOuterY  = frontWallOuterY
	BearingWidth     = bearingWidth
	ShaftCamEnd      = shaftCamEnd
	ShaftCamFrom     = shaftCamFrom
	ShaftRearFrom    = shaftRearFrom
	ShaftJournal     = shaftJournal
	ShaftStubFlatTo  = shaftStubFlatTo
	ShaftStubRadius  = shaftStubRadius
	CamFaceOffset    = camFaceOffset
	CradleFrom       = cradleFrom
	TubeFrom         = tubeFrom
	TopFrom          = topFrom
	MarkRadius       = markRadius
	BearingRaceR     = bearingRaceR
	PivotPocketHalf  = pivotPocketHalf
	HousingBezelY    = housingBezelY
	ButtonUndersideY = buttonUndersideY
	HubBaseY         = hubBaseY
	WindowHalfWidth  = windowHalfWidth
	WindowChamfer    = windowChamfer
	WindowEnd        = windowEnd
	HousingCrownY    = housingCrownY
	RotorPlateY      = rotorPlateY
	ShaftCamRadius   = shaftCamRadius
	ShaftCamFlat     = shaftCamFlat
	CamBoreRadius    = camBoreRadius
	CamBoreFlat      = camBoreFlat
)

// The drive cone's dimensions that are computed from its half angle, and the constants that they come from.
var (
	ConeSlope    = coneSlope
	ConeMouthR   = coneMouthR
	ConeJoinR    = coneJoinR
	ConeTipR     = coneTipR
	ConeLength   = coneLength
	ConeOffsetW  = coneOffsetW
	ConeOffsetZ  = coneOffsetZ
	ConeBaseR    = coneBaseR
	ConeDiscR    = coneDiscR
	ConeInsertR  = coneInsertR
	ConeFit      = coneFit
	ConePlay     = conePlay
	ConeSpan     = coneSpan
	ConeJoin     = coneJoin
	ConeHalfDeg  = coneHalfAngle
	ConeChamfer  = coneChamfer
	ConeEngage   = coneEngage
	ConeOffsetDg = coneOffsetDeg
)

// ToAxis returns +1 when the rotor axis lies toward +w from the drive axis in a tongue sketch of a slider along u, else -1.
func ToAxis(uIsX bool) float64 { return toAxis(uIsX) }

// RaiseOf returns the droop that segment s's pin height has built in.
func RaiseOf(s Segment) float64 { return raise(s) }

// MeasureNotchFloor returns the distance of segment seg's built notch floor from the rotor axis along the yellow face's
// normal, found by bisection on the built black body along the b axis (the yellow slab holds the a axis) in the middle of the first notch.
func MeasureNotchFloor(t *testing.T, seg Segment) float64 {
	t.Helper()
	f := sharedFlag(t, seg)
	n := rotorNotches(seg)[0]
	mid := (n[0] + n[1]) / 2
	lo, hi := 3.0, 5.4 // empty at lo (inside the notch), material at hi (the shell under the face)
	if inBody(t, f.Black, rotorPoint(seg, 0, lo, mid)) || !inBody(t, f.Black, rotorPoint(seg, 0, hi, mid)) {
		t.Fatalf("notch floor is not between %.1f and %.1f", lo, hi)
	}
	for hi-lo > 0.004 {
		m := (lo + hi) / 2
		if inBody(t, f.Black, rotorPoint(seg, 0, m, mid)) {
			hi = m
		} else {
			lo = m
		}
	}
	return (lo + hi) / 2
}

// SectionAt cuts body with the plane y = y and returns its loops as (x, z - ShaftZ).
func SectionAt(t *testing.T, body *decad.Body, y float64) [][]r3.Vec { return sectionAt(t, body, y) }

// LoopsGap returns the least distance between the loops a and b.
func LoopsGap(a, b [][]r3.Vec) float64 { return loopsGap(a, b) }
