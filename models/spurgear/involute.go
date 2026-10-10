package spurgear

import "math"

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
	rotation := flankRotation(p, d)
	left := make([]point, p.Samples)
	right := make([]point, p.Samples)
	for i := range left {
		radius := d.base + (d.tip-d.base)*float64(i)/float64(p.Samples-1)
		sample := involutePoint(d.base, radius)
		sample.y = -sample.y
		sample = rotate(sample, rotation)
		left[i] = rotate(sample, angle)
		sample.y = -sample.y
		right[i] = rotate(sample, angle)
	}
	return left, right
}

func flankRotation(p Params, d dimensions) float64 {
	atPitch := involutePoint(d.base, d.pitch)
	return math.Pi/(2*float64(p.Teeth)) - math.Atan2(-atPitch.y, atPitch.x)
}
