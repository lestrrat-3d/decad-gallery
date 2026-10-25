package cycloidal

import (
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// drawPatternedLoop draws one repeat of a closed loop of fit splines and copies it n-1 times about center with
// a circular pattern, as the Fusion add-in does. repeat holds the points of each spline of one repeat, in order:
// a spline's last point is the next spline's first, and the last spline ends where the next repeat starts.
//
// Only the repeat's points carry dimensions. Its end point is left free: the ring pattern places it one step
// around from the start, and neighbouring copies share that one point, so the loop closes on exact shared
// endpoints, as decad needs to record it. n must be at least 2.
func drawPatternedLoop(s *sketch.Sketch, center *sketch.Point, repeat [][]point, n int) error {
	var start, shared, end *sketch.Point
	splines := make([]sketch.Entity, 0, len(repeat))
	for i, run := range repeat {
		fit := make([]*sketch.Point, len(run))
		for j, q := range run {
			switch {
			case j == 0 && shared != nil:
				fit[j] = shared
			case i == len(repeat)-1 && j == len(run)-1:
				fit[j] = s.CreatePoint(q.x, q.y)
				end = fit[j]
			default:
				fit[j] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		if start == nil {
			start = fit[0]
		}
		shared = fit[len(fit)-1]
		spline, err := s.CreateFitSpline(fit...)
		if err != nil {
			return fmt.Errorf("spline %d of the repeat: %w", i+1, err)
		}
		splines = append(splines, spline)
	}
	if _, err := sketch.CreatePatternCircularRing(s, splines, center, n, start, end); err != nil {
		return fmt.Errorf("pattern the repeat %d times: %w", n, err)
	}
	return nil
}
