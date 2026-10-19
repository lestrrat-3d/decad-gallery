package modelsketch

import (
	"math"

	"github.com/lestrrat-3d/sketch"
)

// zeroTolerance is the offset below which a point is held by a geometric alignment instead of a dimension.
const zeroTolerance = 1e-12

// PointFrom creates a point at (x, y) held relative to from by a horizontal and a vertical offset.
// A zero offset becomes a vertical or horizontal alignment of the two points instead of a zero dimension.
// s and from must not be nil; from must be a point of s that is already held in place.
func PointFrom(s *sketch.Sketch, from *sketch.Point, x, y float64) *sketch.Point {
	pt := s.CreatePoint(x, y)
	dx, dy := x-from.X(), y-from.Y()
	if math.Abs(dx) < zeroTolerance {
		s.AddConstraint(sketch.NewVerticalPoints(from, pt))
	} else {
		s.AddConstraint(sketch.NewHorizontalDistance(from, pt, dx))
	}
	if math.Abs(dy) < zeroTolerance {
		s.AddConstraint(sketch.NewHorizontalPoints(from, pt))
	} else {
		s.AddConstraint(sketch.NewVerticalDistance(from, pt, dy))
	}
	return pt
}
