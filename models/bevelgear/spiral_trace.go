package bevelgear

import (
	"fmt"
	"math"
)

// spiralTrace is the cutter-circle arc in the root-element tangent frame.
type spiralTrace struct {
	toe, heel [2]float64
	total     float64
	handSign  float64
}

// buildSpiralTrace follows the source's near-intersection and crown-roll law.
// The sign is reversed for the pinion so an equal pair has mirrored traces.
func buildSpiralTrace(frame spiralSliceFrame, p Params, pinion bool) (spiralTrace, error) {
	if frame.span <= 0 || frame.mean <= 0 || frame.pitchAngle <= 0 ||
		p.SpiralAngle <= 0 || p.SpiralAngle >= math.Pi/3 {
		return spiralTrace{}, fmt.Errorf("spiral trace frame or angle is invalid")
	}
	handSign := 1.0
	switch p.SpiralHand {
	case "right":
	case "left":
		handSign = -1
	default:
		return spiralTrace{}, fmt.Errorf("spiral hand must be right or left")
	}
	if pinion {
		handSign = -handSign
	}
	cutter := p.CutterRadius
	if cutter == 0 {
		cutter = frame.mean
	}
	if cutter <= 0 || math.IsNaN(cutter) || math.IsInf(cutter, 0) {
		return spiralTrace{}, fmt.Errorf("spiral cutter radius must be positive and finite")
	}
	cx := frame.mean - cutter*math.Sin(p.SpiralAngle)
	cy := handSign * cutter * math.Cos(p.SpiralAngle)
	toe, err := cutterIntersectionNear(frame.toe-0.06*frame.span, cx, cy, cutter, frame.mean)
	if err != nil {
		return spiralTrace{}, fmt.Errorf("spiral toe intersection: %w", err)
	}
	heel, err := cutterIntersectionNear(frame.heel+0.06*frame.span, cx, cy, cutter, frame.mean)
	if err != nil {
		return spiralTrace{}, fmt.Errorf("spiral heel intersection: %w", err)
	}
	phiCrown := math.Atan2(heel[1], heel[0]) - math.Atan2(toe[1], toe[0])
	total := math.Abs(phiCrown) / math.Sin(frame.pitchAngle)
	if math.IsNaN(total) || math.IsInf(total, 0) {
		return spiralTrace{}, fmt.Errorf("spiral crown roll is not finite")
	}
	return spiralTrace{toe: toe, heel: heel, total: total, handSign: handSign}, nil
}

func cutterIntersectionNear(radius, cx, cy, cutter, mean float64) ([2]float64, error) {
	centerDistance := math.Hypot(cx, cy)
	if radius <= 0 || centerDistance == 0 {
		return [2]float64{}, fmt.Errorf("circle intersection has no positive radius or center distance")
	}
	along := (radius*radius - cutter*cutter + centerDistance*centerDistance) / (2 * centerDistance)
	heightSquared := radius*radius - along*along
	if heightSquared < -1e-12*radius*radius {
		return [2]float64{}, fmt.Errorf("cutter circle misses cone-distance circle")
	}
	height := math.Sqrt(math.Max(0, heightSquared))
	ux, uy := cx/centerDistance, cy/centerDistance
	first := [2]float64{along*ux - height*uy, along*uy + height*ux}
	second := [2]float64{along*ux + height*uy, along*uy - height*ux}
	if math.Hypot(first[0]-mean, first[1]) <= math.Hypot(second[0]-mean, second[1]) {
		return first, nil
	}
	return second, nil
}
