package spurgear

import (
	"fmt"
	"math"
)

type point struct {
	x float64
	y float64
}

type dimensions struct {
	pitch float64
	base  float64
	root  float64
	tip   float64
}

func derive(p Params) dimensions {
	pitch := p.Module * float64(p.Teeth) / 2
	return dimensions{
		pitch: pitch,
		base:  pitch * math.Cos(p.PressureAngle),
		root:  (p.Module*float64(p.Teeth) - 2.5*p.Module) / 2,
		tip:   (p.Module*float64(p.Teeth) + 2*p.Module) / 2,
	}
}

func involutePoint(base, radius float64) point {
	alpha := math.Acos(base / radius)
	t := math.Tan(alpha)
	return point{
		x: base * (math.Cos(t) + t*math.Sin(t)),
		y: base * (math.Sin(t) - t*math.Cos(t)),
	}
}

func rotate(p point, angle float64) point {
	sine, cosine := math.Sin(angle), math.Cos(angle)
	return point{
		x: p.x*cosine - p.y*sine,
		y: p.x*sine + p.y*cosine,
	}
}

func toothFlanks(p Params, d dimensions, angle float64) ([]point, []point) {
	return flankPoints(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
}

// InvoluteFlanks samples one tooth with a real-valued effective tooth count.
// Each returned point is an (x, y) coordinate in millimeters.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	if math.IsNaN(pitch) || math.IsInf(pitch, 0) ||
		math.IsNaN(base) || math.IsInf(base, 0) ||
		math.IsNaN(tip) || math.IsInf(tip, 0) ||
		math.IsNaN(effectiveTeeth) || math.IsInf(effectiveTeeth, 0) ||
		math.IsNaN(angle) || math.IsInf(angle, 0) ||
		base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute radii, tooth count, samples, or angle")
	}
	left, right := flankPoints(pitch, base, tip, effectiveTeeth, samples, angle)
	leftXY := make([][2]float64, len(left))
	rightXY := make([][2]float64, len(right))
	for i := range left {
		leftXY[i] = [2]float64{left[i].x, left[i].y}
		rightXY[i] = [2]float64{right[i].x, right[i].y}
	}
	return leftXY, rightXY, nil
}

func flankPoints(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([]point, []point) {
	rotation := flankRotation(pitch, base, effectiveTeeth)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range left {
		radius := base + (tip-base)*float64(i)/float64(samples-1)
		sample := involutePoint(base, radius)
		sample.y = -sample.y
		sample = rotate(sample, rotation)
		left[i] = rotate(sample, angle)
		sample.y = -sample.y
		right[i] = rotate(sample, angle)
	}
	return left, right
}

func flankRotation(pitch, base, effectiveTeeth float64) float64 {
	atPitch := involutePoint(base, pitch)
	return math.Pi/(2*effectiveTeeth) - math.Atan2(-atPitch.y, atPitch.x)
}
