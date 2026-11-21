package sevensegment

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

func buttonDetentDisc(ctx context.Context, doc *decad.Document) (*decad.Body, *sketch.Sketch, error) {
	w := sketch.NewWorld()
	s, err := w.CreateSketch(w.XY())
	if err != nil {
		return nil, nil, fmt.Errorf("create detent sketch: %w", err)
	}
	ref := s.CreateLine(s.Origin(), s.CreatePoint(1, 0))
	ref.SetConstruction(true)
	s.AddConstraint(sketch.NewHorizontal(ref), sketch.NewDistance(ref.Start, ref.End, 1))
	vertices := make([]*sketch.Point, 360)
	for i := range vertices {
		theta := float64(i) * math.Pi / 180
		r, d := buttonDetentPitch(float64(i))
		c, v := math.Cos(theta), math.Sin(theta)
		length := math.Hypot(r, d)
		x := r*c - 3*(r*c+d*v)/length
		y := -r*v - 3*(-r*v+d*c)/length
		vertices[i] = s.CreatePoint(x, y)
		ray := s.CreateLine(s.Origin(), vertices[i])
		ray.SetConstruction(true)
		s.AddConstraint(sketch.NewAngle(ref, ray, math.Atan2(y, x)*180/math.Pi),
			sketch.NewDistance(s.Origin(), vertices[i], math.Hypot(x, y)))
	}
	for i := range vertices {
		s.CreateLine(vertices[i], vertices[(i+1)%len(vertices)])
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve detent disc: %w", err)
	}
	report := s.Verify(ctx)
	if !report.Analysed() || report.Status != sketch.FullyConstrained || report.DOF != 0 ||
		len(report.Redundant) != 0 || len(report.Conflicts) != 0 || !report.ProfilesValid || len(report.Profiles) != 1 {
		return nil, nil, fmt.Errorf("verify detent disc: status=%v dof=%d redundant=%d conflicts=%d profiles=%d",
			report.Status, report.DOF, len(report.Redundant), len(report.Conflicts), len(report.Profiles))
	}
	body, err := decad.Extrude(ctx, doc, s, report.Profiles[0], decad.Symmetric{D: units.Millimeters(1)})
	if err != nil {
		return nil, nil, fmt.Errorf("extrude detent disc: %w", err)
	}
	return body, s, nil
}
