package sevensegment

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type buttonReleaseCrank struct{ Crank, Input, Output, Bias, Pivot *decad.Body }

func releaseSketch(frame r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	p, err := w.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, err
	}
	return w.CreateSketch(p)
}

func releaseProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("got %d profiles, want one", len(profiles))
	}
	return profiles[0], nil
}

func releasePolygon(ctx context.Context, doc *decad.Document) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.NewVec(-46, 0, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, err
	}
	s, err := releaseSketch(frame)
	if err != nil {
		return nil, err
	}
	coords := [][2]float64{{-23, 7}, {-23, 22.7}, {-29, 22.7}, {-29, 13}, {-38.8, 13}, {-38.8, 7}}
	points := make([]*sketch.Point, len(coords))
	for i, p := range coords {
		points[i] = s.CreatePoint(p[0], p[1])
	}
	for i := range points {
		j := (i + 1) % len(points)
		l := s.CreateLine(points[i], points[j])
		if coords[i][0] == coords[j][0] {
			s.AddConstraint(sketch.NewVertical(l))
		} else {
			s.AddConstraint(sketch.NewHorizontal(l))
		}
	}
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), points[0], -23),
		sketch.NewVerticalDistance(s.Origin(), points[0], 7),
		sketch.NewVerticalDistance(s.Origin(), points[1], 22.7),
		sketch.NewHorizontalDistance(s.Origin(), points[2], -29),
		sketch.NewVerticalDistance(s.Origin(), points[3], 13),
		sketch.NewHorizontalDistance(s.Origin(), points[4], -38.8),
	)
	profile, err := releaseProfile(ctx, s)
	if err != nil {
		return nil, fmt.Errorf("polygon: %w", err)
	}
	return decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(3), Dir: decad.Along})
}

func releaseDrill(ctx context.Context, doc *decad.Document, x, y, z, r, depth float64) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.NewVec(x, 0, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, err
	}
	s, err := releaseSketch(frame)
	if err != nil {
		return nil, err
	}
	c := sketchcheck.DimensionPoint(s, y, z)
	circle := s.CreateCircle(c, r)
	s.AddConstraint(sketch.NewRadius(circle, r))
	profile, err := releaseProfile(ctx, s)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(depth), Dir: decad.Along})
}

// releasePin draws radius on U and axial position on V.
func releasePin(ctx context.Context, doc *decad.Document, y, z float64, points [][2]float64) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, y, z), r3.NewVec(0, 0, 1), r3.NewVec(1, 0, 0))
	if err != nil {
		return nil, err
	}
	s, err := releaseSketch(frame)
	if err != nil {
		return nil, err
	}
	ps := make([]*sketch.Point, len(points))
	for i, p := range points {
		ps[i] = s.CreatePoint(p[0], p[1])
	}
	for i := range ps {
		j := (i + 1) % len(ps)
		l := s.CreateLine(ps[i], ps[j])
		if points[i][0] == points[j][0] {
			s.AddConstraint(sketch.NewVertical(l))
		} else {
			s.AddConstraint(sketch.NewHorizontal(l))
		}
	}
	// Equal-coordinate constraints share repeated radii and axial stations.
	radii := map[float64]*sketch.Point{}
	stations := map[float64]*sketch.Point{}
	for i, p := range points {
		j := (i + 1) % len(points)
		if p[0] == points[j][0] {
			if first := radii[p[0]]; first != nil {
				s.AddConstraint(sketch.NewHorizontalDistance(first, ps[i], 0))
			} else {
				s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[i], p[0]))
				radii[p[0]] = ps[i]
			}
		} else {
			if first := stations[p[1]]; first != nil {
				s.AddConstraint(sketch.NewVerticalDistance(first, ps[i], 0))
			} else {
				s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), ps[i], p[1]))
				stations[p[1]] = ps[i]
			}
		}
	}
	profile, err := releaseProfile(ctx, s)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(ctx, doc, s, profile, decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}, decad.FullRevolution{})
}

func buildButtonReleaseCrank(ctx context.Context, doc *decad.Document, lift float64) (*buttonReleaseCrank, error) {
	if ctx == nil || doc == nil || math.IsNaN(lift) || math.IsInf(lift, 0) || lift < 0 || lift > 2 {
		return nil, fmt.Errorf("button release crank: invalid context, document or lift %g", lift)
	}
	crank, err := releasePolygon(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("button release crank: blank: %w", err)
	}
	for _, hole := range [][3]float64{{-26, 10, 1.88}, {-26, 20, 1.36}, {-36, 10, 1.36}, {-26, 15, 1.36}} {
		tool, e := releaseDrill(ctx, doc, -46.1, hole[0], hole[1], hole[2], 3.2)
		if e != nil {
			return nil, fmt.Errorf("button release crank: bore: %w", e)
		}
		crank, e = decad.Cut(ctx, crank, tool)
		if e != nil {
			return nil, fmt.Errorf("button release crank: cut bore: %w", e)
		}
	}
	for _, hole := range [][2]float64{{-26, 20}, {-36, 10}, {-26, 15}} {
		for _, x := range []float64{-46.1, -43.84} {
			tool, e := releaseDrill(ctx, doc, x, hole[0], hole[1], 2.18, 0.94)
			if e != nil {
				return nil, fmt.Errorf("button release crank: counterbore: %w", e)
			}
			crank, e = decad.Cut(ctx, crank, tool)
			if e != nil {
				return nil, fmt.Errorf("button release crank: cut counterbore: %w", e)
			}
		}
	}
	input := [][2]float64{{0, -45.95}, {1.8, -45.95}, {1.8, -45.54}, {.98, -45.54}, {.98, -43.46}, {1.8, -43.46}, {1.8, -43.05}, {1, -43.05}, {1, -39.5}, {0, -39.5}}
	negative := func(end float64) [][2]float64 {
		return [][2]float64{{0, end}, {1, end}, {1, -45.95}, {1.8, -45.95}, {1.8, -45.54}, {.98, -45.54}, {.98, -43.46}, {1.8, -43.46}, {1.8, -43.05}, {0, -43.05}}
	}
	pivot := [][2]float64{{0, -48.95}, {2, -48.95}, {2, -48.63}, {1.5, -48.63}, {1.5, -42.62}, {2.3, -42.62}, {2.3, -42.27}, {0, -42.27}}
	bodies := []*decad.Body{crank}
	for _, spec := range []struct {
		y, z   float64
		points [][2]float64
	}{{-26, 20, input}, {-36, 10, negative(-49.5)}, {-26, 15, negative(-57.5)}, {-26, 10, pivot}} {
		b, e := releasePin(ctx, doc, spec.y, spec.z, spec.points)
		if e != nil {
			return nil, fmt.Errorf("button release crank: pin: %w", e)
		}
		bodies = append(bodies, b)
	}
	rot, err := r3.RotationAround(r3.NewVec(0, -26, 10), r3.NewVec(1, 0, 0), units.Radians(math.Asin((1-lift)/10)))
	if err != nil {
		return nil, fmt.Errorf("button release crank: rotation: %w", err)
	}
	frame, err := r3.NewFrame(r3.NewVec(10, -140, 52), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("button release crank: world frame: %w", err)
	}
	place, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("button release crank: world placement: %w", err)
	}
	for i, b := range bodies {
		if i < 4 {
			b, err = decad.Transform(ctx, b, rot)
			if err != nil {
				return nil, fmt.Errorf("button release crank: rotate: %w", err)
			}
		}
		b, err = decad.Transform(ctx, b, place)
		if err != nil {
			return nil, fmt.Errorf("button release crank: place: %w", err)
		}
		bodies[i] = b
	}
	group, err := doc.Root().NewUniqueChild("Release Mechanism")
	if err != nil {
		return nil, fmt.Errorf("button release crank: mechanism container: %w", err)
	}
	crankContainer, err := group.NewChild("Release Crank")
	if err != nil {
		return nil, fmt.Errorf("button release crank: crank container: %w", err)
	}
	if err = crankContainer.AddBody(bodies[0]); err != nil {
		return nil, fmt.Errorf("button release crank: add crank: %w", err)
	}
	pins, err := group.NewChild("Release Pins")
	if err != nil {
		return nil, fmt.Errorf("button release crank: pins container: %w", err)
	}
	for _, body := range bodies[1:] {
		if err = pins.AddBody(body); err != nil {
			return nil, fmt.Errorf("button release crank: add pin: %w", err)
		}
	}
	return &buttonReleaseCrank{bodies[0], bodies[1], bodies[2], bodies[3], bodies[4]}, nil
}
