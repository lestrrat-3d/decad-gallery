package main

import (
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

type platePoint struct{ x, y float64 }

func platePolar(radius, angle float64) platePoint {
	x, y := polarPoint(radius, angle)
	return platePoint{x, y}
}

func platePolygon(s *sketch.Sketch, xy []platePoint) {
	points := make([]*sketch.Point, len(xy))
	for i, p := range xy {
		points[i] = sketchcheck.DimensionPoint(s, p.x, p.y)
	}
	for i, p := range points {
		s.CreateLine(p, points[(i+1)%len(points)])
	}
}

// The four cardinal rays meet the gear rim. The intercardinal rays are shorter.
// A triangular opening follows one side of each ray.
func drawCompass(s *sketch.Sketch, r float64) int {
	const rays = 8
	// Offset the pattern so symmetric polygon vertices fall on different scanlines
	// when Decad triangulates the pierced cap.
	const phaseInSteps = .32
	star := make([]platePoint, 0, 2*rays)
	for i := range rays {
		a := (phaseInSteps + float64(i)) * 2 * math.Pi / rays
		tip := .81
		if i%2 == 1 {
			tip = .62
		}
		star = append(star, platePolar(tip*r, a), platePolar(.35*r, a+math.Pi/rays))
		sin, cos := math.Sincos(a)
		triangle := make([]platePoint, 0, 3)
		for _, p := range []platePoint{{.235, .015}, {.301, .087}, {tip - .082, .012}} {
			triangle = append(triangle, platePoint{r * (p.x*cos - p.y*sin), r * (p.x*sin + p.y*cos)})
		}
		platePolygon(s, triangle)
	}
	platePolygon(s, star)
	return rays
}

// Two circular arcs curve each vane in the same direction. Their ends lie on
// the hub and rim circles, which form the other two edges of each opening.
func drawVanes(s *sketch.Sketch, r float64) int {
	const count = 6
	const step = 2 * math.Pi / count
	border := s.CreateCircle(s.Origin(), .792*r)
	s.AddConstraint(sketch.NewRadius(border, .792*r))
	for i := range count {
		a := float64(i) * step
		inner := [2]platePoint{platePolar(.257*r, a), platePolar(.257*r, a+.53*step)}
		outer := [2]platePoint{platePolar(.695*r, a+.55*step), platePolar(.695*r, a+1.33*step)}
		arc := func(ends [2]platePoint) [2]*sketch.Point {
			p := sketchcheck.DimensionPoint(s, ends[0].x, ends[0].y)
			q := sketchcheck.DimensionArcEnd(s, ends[1].x, ends[1].y)
			s.CreateArc(s.Origin(), p, q)
			return [2]*sketch.Point{p, q}
		}
		inside, outside := arc(inner), arc(outer)
		for j := range 2 {
			dx, dy := (outer[j].x-inner[j].x)/2, (outer[j].y-inner[j].y)/2
			cx := inner[j].x + dx - 1.15*dy
			cy := inner[j].y + dy + 1.15*dx
			ornamentArc(s, inside[j], outside[j], cx, cy, dx, dy)
		}
	}
	return count
}

// Eighteen hexagons occupy two full rings of a hexagonal lattice. The center
// cell stays solid behind the hub.
func drawHoneycomb(s *sketch.Sketch, r float64) int {
	// Rotate the whole lattice so its vertices do not share horizontal scanlines.
	sin, cos := math.Sincos(.17)
	hexagon := func(x, y, radius, phase float64) {
		corners := make([]platePoint, 6)
		for i := range corners {
			p := platePolar(radius, phase+float64(i)*math.Pi/3)
			dx, dy := x+p.x, y+p.y
			corners[i] = platePoint{dx*cos - dy*sin, dx*sin + dy*cos}
		}
		platePolygon(s, corners)
	}
	hexagon(0, 0, .817*r, 0)
	pitch := .285 * r
	for row := -2; row <= 2; row++ {
		for col := max(-2, -row-2); col <= min(2, 2-row); col++ {
			if row == 0 && col == 0 {
				continue
			}
			x := pitch * (float64(col) + float64(row)/2)
			y := pitch * float64(row) * math.Sqrt(3) / 2
			hexagon(x, y, .136*r, math.Pi/6)
		}
	}
	return 18
}

// Eight round openings decorate the faceplate around its central bore.
func drawPerforated(s *sketch.Sketch, r float64) int {
	const count = 8
	outer := s.CreateCircle(s.Origin(), .792*r)
	s.AddConstraint(sketch.NewRadius(outer, .792*r))
	for i := range count {
		x, y := polarPoint(.49*r, (float64(i)+.2)*2*math.Pi/count)
		center := sketchcheck.DimensionPoint(s, x, y)
		hole := s.CreateCircle(center, .13*r)
		s.AddConstraint(sketch.NewRadius(hole, .13*r))
	}
	return count
}

// Six diamond-shaped windows radiate from the central hub.
func drawDiamonds(s *sketch.Sketch, r float64) int {
	const count = 6
	outer := s.CreateCircle(s.Origin(), .792*r)
	s.AddConstraint(sketch.NewRadius(outer, .792*r))
	for i := range count {
		a := (float64(i) + .17) * 2 * math.Pi / count
		points := []platePoint{platePolar(.29*r, a), platePolar(.49*r, a-.17),
			platePolar(.69*r, a), platePolar(.49*r, a+.17)}
		platePolygon(s, points)
	}
	return count
}
