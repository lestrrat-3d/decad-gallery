package sevensegment

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type buttonDetent struct {
	Cam, Key *decad.Body
	Sketches []*sketch.Sketch
}

func buttonDetentVerified(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve: %w", err)
	}
	r := s.Verify(ctx)
	if !r.Analysed() || r.Status != sketch.FullyConstrained || r.DOF != 0 ||
		len(r.Redundant) != 0 || len(r.Conflicts) != 0 || !r.ProfilesValid || len(r.Profiles) != 1 {
		return nil, fmt.Errorf("verify: status=%v dof=%d redundant=%d conflicts=%d profiles=%d",
			r.Status, r.DOF, len(r.Redundant), len(r.Conflicts), len(r.Profiles))
	}
	return r.Profiles[0], nil
}

func buttonDetentCircle(ctx context.Context, doc *decad.Document, radius, halfLength float64) (*decad.Body, *sketch.Sketch, error) {
	w := sketch.NewWorld()
	s, err := w.CreateSketch(w.XY())
	if err != nil {
		return nil, nil, err
	}
	c := s.CreateCircle(s.Origin(), radius)
	s.AddConstraint(sketch.NewRadius(c, radius))
	p, err := buttonDetentVerified(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	b, err := decad.Extrude(ctx, doc, s, p, decad.Symmetric{D: units.Millimeters(halfLength)})
	return b, s, err
}

func buttonDetentOpening(ctx context.Context, doc *decad.Document) (*decad.Body, *sketch.Sketch, error) {
	w := sketch.NewWorld()
	s, err := w.CreateSketch(w.XY())
	if err != nil {
		return nil, nil, err
	}
	xaxis := s.CreateLine(s.Origin(), s.CreatePoint(1, 0))
	xaxis.SetConstruction(true)
	yaxis := s.CreateLine(s.Origin(), s.CreatePoint(0, 1))
	yaxis.SetConstruction(true)
	s.AddConstraint(sketch.NewHorizontal(xaxis), sketch.NewDistance(xaxis.Start, xaxis.End, 1),
		sketch.NewVertical(yaxis), sketch.NewDistance(yaxis.Start, yaxis.End, 1))
	x := math.Sqrt(2.55*2.55 - 1.06*1.06)
	ur := s.CreatePoint(x, 1.06)
	ul := s.CreatePoint(-x, 1.06)
	ll := s.CreatePoint(-x, -1.06)
	lr := s.CreatePoint(x, -1.06)
	top := s.CreateArc(s.Origin(), ur, ul)
	s.CreateArc(s.Origin(), ll, lr)
	s.AddConstraint(sketch.NewRadius(top, 2.55), sketch.NewVerticalDistance(s.Origin(), ur, 1.06),
		sketch.NewVerticalDistance(s.Origin(), ul, 1.06), sketch.NewSymmetric(ul, ll, xaxis),
		sketch.NewVerticalDistance(s.Origin(), lr, -1.06))
	stations := [][2]float64{{8.3, 1.06}, {8.8, 1.01}, {9.6, 1.01}, {10, 1.06}, {10.1, 1.06}}
	upperRight := make([]*sketch.Point, len(stations))
	lowerRight := make([]*sketch.Point, len(stations))
	upperLeft := make([]*sketch.Point, len(stations))
	lowerLeft := make([]*sketch.Point, len(stations))
	for i, p := range stations {
		upperRight[i] = s.CreatePoint(p[0], p[1])
		lowerRight[i] = s.CreatePoint(p[0], -p[1])
		upperLeft[i] = s.CreatePoint(-p[0], p[1])
		lowerLeft[i] = s.CreatePoint(-p[0], -p[1])
		s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), upperRight[i], p[0]),
			sketch.NewVerticalDistance(s.Origin(), upperRight[i], p[1]),
			sketch.NewSymmetric(upperRight[i], lowerRight[i], xaxis),
			sketch.NewSymmetric(upperRight[i], upperLeft[i], yaxis),
			sketch.NewSymmetric(upperLeft[i], lowerLeft[i], xaxis))
	}
	for _, side := range []struct {
		start, end   *sketch.Point
		upper, lower []*sketch.Point
	}{
		{ur, lr, upperRight, lowerRight}, {ul, ll, upperLeft, lowerLeft},
	} {
		prev := side.start
		for _, p := range side.upper {
			s.CreateLine(prev, p)
			prev = p
		}
		prevLower := side.end
		for _, p := range side.lower {
			s.CreateLine(prevLower, p)
			prevLower = p
		}
		s.CreateLine(prev, prevLower)
	}
	p, err := buttonDetentVerified(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("opening sketch: %w", err)
	}
	b, err := decad.Extrude(ctx, doc, s, p, decad.Symmetric{D: units.Millimeters(3)})
	return b, s, err
}

func buildButtonDetent(ctx context.Context, doc *decad.Document, phase float64) (*buttonDetent, error) {
	if ctx == nil || doc == nil || math.IsNaN(phase) || math.IsInf(phase, 0) || phase < 0 || phase > 360 {
		return nil, fmt.Errorf("build button detent: invalid context, document or phase")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("build button detent: %w", err)
	}
	disc, ds, err := buttonDetentDisc(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("build detent disc: %w", err)
	}
	hub, hs, err := buttonDetentCircle(ctx, doc, 13, 2.5)
	if err != nil {
		return nil, fmt.Errorf("build detent hub: %w", err)
	}
	cam, err := decad.Union(ctx, disc, hub)
	if err != nil {
		return nil, fmt.Errorf("unite detent disc and hub: %w", err)
	}
	opening, os, err := buttonDetentOpening(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("build detent opening: %w", err)
	}
	a := math.Atan2(24, 10) + 108*math.Pi/180
	b := math.Atan2(20*math.Sin(a)-24, 20*math.Cos(a)-10)
	spin, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(b))
	if err != nil {
		return nil, fmt.Errorf("opening rotation: %w", err)
	}
	opening, err = decad.Transform(ctx, opening, spin)
	if err != nil {
		return nil, fmt.Errorf("rotate opening: %w", err)
	}
	cam, err = decad.Cut(ctx, cam, opening)
	if err != nil {
		return nil, fmt.Errorf("cut detent opening: %w", err)
	}
	frame, err := r3.NewFrame(r3.NewVec(10, -107, 52), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("cam frame: %w", err)
	}
	placement, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("cam placement: %w", err)
	}
	cam, err = decad.Transform(ctx, cam, placement)
	if err != nil {
		return nil, fmt.Errorf("place cam: %w", err)
	}
	key, ks, err := buttonDetentCircle(ctx, doc, 1, 10)
	if err != nil {
		return nil, fmt.Errorf("build detent key: %w", err)
	}
	keyFrame, err := r3.NewFrame(r3.NewVec(10, -107, 52), r3.NewVec(0, 1, 0),
		r3.NewVec(-math.Sin(b), 0, math.Cos(b)))
	if err != nil {
		return nil, fmt.Errorf("key frame: %w", err)
	}
	keyPlacement, err := r3.FromFrame(keyFrame)
	if err != nil {
		return nil, fmt.Errorf("key placement: %w", err)
	}
	key, err = decad.Transform(ctx, key, keyPlacement)
	if err != nil {
		return nil, fmt.Errorf("place key: %w", err)
	}
	pose, err := r3.RotationAround(r3.NewVec(10, -69.75, 52), r3.NewVec(0, -1, 0), units.Degrees(phase))
	if err != nil {
		return nil, fmt.Errorf("detent phase: %w", err)
	}
	cam, err = decad.Transform(ctx, cam, pose)
	if err != nil {
		return nil, fmt.Errorf("phase cam: %w", err)
	}
	key, err = decad.Transform(ctx, key, pose)
	if err != nil {
		return nil, fmt.Errorf("phase key: %w", err)
	}
	if !cam.IsSolid() || len(cam.Lumps()) != 1 || !key.IsSolid() {
		return nil, fmt.Errorf("detent bodies are not single solids")
	}
	for _, part := range []struct {
		name string
		body *decad.Body
	}{{"Detent Cams", cam}, {"Detent Keys", key}} {
		container, err := doc.Root().NewUniqueChild(part.name)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", part.name, err)
		}
		if err := container.AddBody(part.body); err != nil {
			return nil, fmt.Errorf("group %s: %w", part.name, err)
		}
	}
	return &buttonDetent{Cam: cam, Key: key, Sketches: []*sketch.Sketch{ds, hs, os, ks}}, nil
}
