package sketchcheck

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// DimensionPoint locates a computed point relative to the sketch origin.
func DimensionPoint(s *sketch.Sketch, x, y float64) *sketch.Point {
	point := s.CreatePoint(x, y)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), point, x),
		sketch.NewVerticalDistance(s.Origin(), point, y),
	)
	return point
}

// DimensionArcEnd leaves the radial coordinate to the arc's radius relation.
func DimensionArcEnd(s *sketch.Sketch, x, y float64) *sketch.Point {
	point := s.CreatePoint(x, y)
	if math.Abs(x) >= math.Abs(y) {
		s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), point, y))
	} else {
		s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), point, x))
	}
	return point
}

// VerifyFullyConstrained requires a sketch with no free or duplicate constraints.
func VerifyFullyConstrained(ctx context.Context, s *sketch.Sketch) error {
	report := s.Verify(ctx)
	if report.Status != sketch.FullyConstrained || report.DOF != 0 ||
		len(report.Redundant) != 0 || len(report.Conflicts) != 0 {
		return fmt.Errorf("status %s, DOF %d, %d redundant, %d conflicts",
			report.Status, report.DOF, len(report.Redundant), len(report.Conflicts))
	}
	return report.Check()
}
