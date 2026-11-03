package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

func keyedOutline(s *sketch.Sketch, radius, halfwidth, extent float64) {
	if s == nil {
		return
	}

	x := math.Sqrt(radius*radius - halfwidth*halfwidth)
	upper := s.CreatePoint(x, halfwidth)
	lower := s.CreatePoint(x, -halfwidth)
	outerLower := s.CreatePoint(extent, -halfwidth)
	outerUpper := s.CreatePoint(extent, halfwidth)
	arc := s.CreateArc(s.Origin(), upper, lower)
	bottom := s.CreateLine(lower, outerLower)
	outer := s.CreateLine(outerLower, outerUpper)
	top := s.CreateLine(outerUpper, upper)
	s.AddConstraint(
		sketch.NewRadius(arc, radius),
		sketch.NewHorizontal(bottom),
		sketch.NewVertical(outer),
		sketch.NewHorizontal(top),
		sketch.NewVerticalDistance(s.Origin(), upper, halfwidth),
		sketch.NewVerticalDistance(s.Origin(), lower, -halfwidth),
		sketch.NewHorizontalDistance(s.Origin(), outerUpper, extent),
	)
}

// turnedKeyedOutline is keyedOutline with its key axis turned angle radians counterclockwise from the
// sketch's +X. A construction line from the origin, horizontal, references the angle; a second one from the
// origin carries the key axis out to the key's outer face.
func turnedKeyedOutline(s *sketch.Sketch, radius, halfwidth, extent, angle float64) {
	if s == nil {
		return
	}
	c, sn := math.Cos(angle), math.Sin(angle)
	turn := func(x, y float64) *sketch.Point { return s.CreatePoint(c*x-sn*y, sn*x+c*y) }
	ref := s.CreatePoint(1, 0)
	refLine := s.CreateLine(s.Origin(), ref)
	refLine.SetConstruction(true)
	tip := turn(extent, 0)
	axis := s.CreateLine(s.Origin(), tip)
	axis.SetConstruction(true)
	x := math.Sqrt(radius*radius - halfwidth*halfwidth)
	upper := turn(x, halfwidth)
	lower := turn(x, -halfwidth)
	outerLower := turn(extent, -halfwidth)
	outerUpper := turn(extent, halfwidth)
	arc := s.CreateArc(s.Origin(), upper, lower)
	bottom := s.CreateLine(lower, outerLower)
	outer := s.CreateLine(outerLower, outerUpper)
	top := s.CreateLine(outerUpper, upper)
	turnAngle := sketch.NewAngle(refLine, axis, 0)
	s.AddConstraint(
		sketch.NewHorizontal(refLine),
		sketch.NewHorizontalDistance(s.Origin(), ref, 1),
		turnAngle,
		sketch.NewDistance(s.Origin(), tip, extent),
		sketch.NewRadius(arc, radius),
		sketch.NewParallel(bottom, axis),
		sketch.NewParallel(top, axis),
		sketch.NewPerpendicular(outer, axis),
		sketch.NewDistancePointLine(upper, axis, halfwidth),
		sketch.NewDistancePointLine(lower, axis, halfwidth),
		sketch.NewPointOnLine(tip, outer),
	)
	turnAngle.SetValue(units.Radians(angle))
}

// liftKeyTool is the keyway tool of a keyedOutline, sketched on f, a plane of l's local coordinates, and placed
// by l.
func (b *builder) liftKeyTool(l lift, f r3.Frame, radius, halfwidth, extent, length float64) *decad.Body {
	s := b.sectionOn(f)
	keyedOutline(s, radius, halfwidth, extent)
	return b.liftExtrude(s, 0, length, l)
}

func (b *builder) keyedAnnulus(l lift, f r3.Frame, radius, halfwidth, extent, outer, length float64) *decad.Body {
	s := b.sectionOn(f)
	circle(s, 0, 0, outer)
	keyedOutline(s, radius, halfwidth, extent)
	return b.liftExtrude(s, 1, length, l)
}
