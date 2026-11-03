package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
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
