package screwgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// toothFitPoints is how many points each section's toothed side is fitted through.
const toothFitPoints = 11

// sectionsPerTooth is how many loft sections span one tooth pitch.
const sectionsPerTooth = 10

// edgeAt is the toothed edge's position across the width at thickness coordinate v and
// station s, before the twist. The tooth phase moves the whole edge along the axis.
func (g gear) edgeAt(v, s float64) float64 {
	p := g.p
	cell := (s + math.Tan(p.ToothSlant)*v - g.phase) / p.ToothPitch
	return p.Width/2 - p.ToothHeight/2 + p.ToothHeight/2*math.Cos(2*math.Pi*cell) - p.ToothBow*v*v
}

// section draws the gear's cross-section at station s on the plane z = s of the gear's
// own frame (u along X, v along Y), turned by the twist angle there.
func (g gear) section(ctx context.Context, w *sketch.World, s float64) (decad.LoftSection, error) {
	p := g.p
	plane := w.XY()
	if s != 0 {
		var err error
		if plane, err = w.CreateOffsetPlane(w.XY(), s); err != nil {
			return decad.LoftSection{}, fmt.Errorf("section plane: %w", err)
		}
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		return decad.LoftSection{}, fmt.Errorf("section sketch: %w", err)
	}
	sin, cos := math.Sincos(g.angle(s))
	at := func(u, v float64) *sketch.Point {
		return sketchcheck.DimensionPoint(sk, u*cos-v*sin, u*sin+v*cos)
	}
	half := p.Thickness / 2
	back0 := at(-p.Width/2, -half)
	fit := make([]*sketch.Point, toothFitPoints)
	for j := range fit {
		v := -half + p.Thickness*float64(j)/float64(toothFitPoints-1)
		fit[j] = at(g.edgeAt(v, s), v)
	}
	back1 := at(-p.Width/2, half)
	sk.CreateLine(back0, fit[0])
	if _, err := sk.CreateFitSpline(fit...); err != nil {
		return decad.LoftSection{}, fmt.Errorf("toothed edge: %w", err)
	}
	sk.CreateLine(fit[toothFitPoints-1], back1)
	sk.CreateLine(back1, back0)
	if err := solveAndVerify(ctx, sk, "ribbon section"); err != nil {
		return decad.LoftSection{}, err
	}
	for _, pr := range sk.Profiles() {
		if pr.Valid && len(pr.Outer) == 4 && len(pr.Holes) == 0 {
			return decad.LoftSection{Sketch: sk, Profile: pr}, nil
		}
	}
	return decad.LoftSection{}, fmt.Errorf("ribbon section at station %g has no four-edge profile", s)
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// ribbonBody lofts the gear's whole ribbon, then moves it from the gear's own frame onto the
// gear's place in the pair frame.
func (g gear) ribbonBody(ctx context.Context, doc *decad.Document) (*decad.Body, error) {
	body, _, _, err := g.ribbonLoft(ctx, doc)
	if err != nil {
		return nil, err
	}
	return g.place(ctx, body)
}

// ribbonLoft lofts the gear's whole ribbon in the gear's own frame through one section per
// tenth of a pitch. It returns the body, the sections and the station step between them.
// The ribbon runs the length of ToothCount pitches centred on the crossing station, then
// advanced by the gear's tooth phase along its own axis.
func (g gear) ribbonLoft(ctx context.Context, doc *decad.Document) (*decad.Body, []decad.LoftSection, float64, error) {
	p := g.p
	w := sketch.NewWorld()
	count := p.ToothCount*sectionsPerTooth + 1
	step := p.Length() / float64(count-1)
	first := -p.Length()/2 + g.phase
	sections := make([]decad.LoftSection, count)
	for i := range sections {
		sec, err := g.section(ctx, w, first+float64(i)*step)
		if err != nil {
			return nil, nil, 0, err
		}
		sections[i] = sec
	}
	body, err := decad.LoftSections(ctx, doc, sections...)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("loft ribbon: %w", err)
	}
	return body, sections, step, nil
}

// place moves a body built in the gear's own frame onto the gear's axis in the pair frame.
func (g gear) place(ctx context.Context, body *decad.Body) (*decad.Body, error) {
	return placeOnFrame(ctx, body, g.origin, g.ex, g.ey)
}

// placeOnFrame moves body from the local frame onto the frame at origin whose X and Y are
// ex and ey.
func placeOnFrame(ctx context.Context, body *decad.Body, origin, ex, ey r3.Vec) (*decad.Body, error) {
	frame, err := r3.NewFrame(origin, ex, ey)
	if err != nil {
		return nil, fmt.Errorf("frame: %w", err)
	}
	move, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("frame transform: %w", err)
	}
	placed, err := decad.Transform(ctx, body, move)
	if err != nil {
		return nil, fmt.Errorf("place body: %w", err)
	}
	return placed, nil
}
