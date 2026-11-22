package sevensegment

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// Internal dimensions that the physics tests read, so that they follow the model and not copies of its numbers.
const (
	NotchFloor    = notchFloor
	SlotEnd       = slotEnd
	TonguePinR    = flagPinRadius
	TongueHullR   = tongueHullR
	TongueWidth   = tongueWidth
	TongueTopGap  = tongueTopGap
	PinAxisZ      = pinAxisZ
	SliderHeadTop = sliderHeadTop
	RotorEdgeR    = rotorEdgeRound
	BladeAcross   = bladeAcross
	SlotRadius    = slotRadius
	JournalHoleR  = journalHoleRadius
	PinStationAGD = pinStationAGD
	PinStationBC  = pinStationBC
	PinStationEF  = pinStationEF

	ShaftCamRadius = shaftCamRadius
	ShaftCamFlat   = shaftCamFlat
	CamBoreRadius  = camBoreRadius
	CamBoreFlat    = camBoreFlat
)

// RaiseOf returns the droop that segment s's pin height has built in.
func RaiseOf(s Segment) float64 { return raise(s) }

// MeasureNotchFloor returns the distance of segment seg's built notch floor from the rotor axis along the yellow face's
// normal, found by bisection on the built black body along the b axis (the yellow slab holds the a axis) in the middle of the first notch.
func MeasureNotchFloor(t *testing.T, seg Segment) float64 {
	t.Helper()
	f := sharedFlag(t, seg)
	n := rotorNotches(seg)[0]
	mid := (n[0] + n[1]) / 2
	lo, hi := 2.0, 3.6 // empty at lo (inside the notch), material at hi (the shell under the face)
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
