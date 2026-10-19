// Package modelsketch holds Sketch helpers shared by the gallery models.
package modelsketch

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// Polar places points relative to a fixed center with a radius and an angle dimension each.
// The angle is measured from a construction line that runs from the center along the sketch X axis.
type Polar struct {
	s      *sketch.Sketch
	center *sketch.Point
	ref    *sketch.Line
}

// NewPolar anchors center in s and adds the construction reference line along +X.
// s and center must not be nil; center must be a point of s.
func NewPolar(s *sketch.Sketch, center *sketch.Point) *Polar {
	s.Fix(center)
	cx, cy := center.X(), center.Y()
	end := s.CreatePoint(cx+1, cy)
	s.Fix(end)
	end.SetConstruction(true)
	ref := s.CreateLine(center, end)
	ref.SetConstruction(true)
	return &Polar{s: s, center: center, ref: ref}
}

// Center returns the anchored center point.
func (g *Polar) Center() *sketch.Point { return g.center }

// Point creates a point at (x, y) held by an angle dimension from the reference line.
// A radial dimension is added when radial is true; leave it off when an arc about the center
// already ties the point's radius to another point, because the extra dimension would be redundant.
func (g *Polar) Point(x, y float64, radial bool) (*sketch.Point, error) {
	cx, cy := g.center.X(), g.center.Y()
	dx, dy := x-cx, y-cy
	radius := math.Hypot(dx, dy)
	if radius == 0 {
		return nil, fmt.Errorf("point (%g, %g) coincides with the center", x, y)
	}
	pt := g.s.CreatePoint(x, y)
	spoke := g.s.CreateLine(g.center, pt)
	spoke.SetConstruction(true)
	if radial {
		g.s.AddConstraint(sketch.NewDistance(g.center, pt, radius))
	}
	g.s.AddConstraint(sketch.NewAngle(g.ref, spoke, math.Atan2(dy, dx)*180/math.Pi))
	return pt, nil
}
