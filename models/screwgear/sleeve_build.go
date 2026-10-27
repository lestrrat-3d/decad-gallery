package screwgear

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

// buildSleeve makes the tube, cuts the four twisted bores in the spec's order (gear A -R,
// gear A +R, gear B -R, gear B +R), then cuts each window that has room. Everything is in
// the pair frame.
func buildSleeve(ctx context.Context, doc *decad.Document, f sleeve) (*decad.Body, error) {
	cage, err := f.tube(ctx, doc)
	if err != nil {
		return nil, err
	}
	for _, b := range f.bores() {
		tool, err := f.boreTool(ctx, doc, b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.name, err)
		}
		if cage, err = decad.Cut(ctx, cage, tool); err != nil {
			return nil, fmt.Errorf("cut %s: %w", b.name, err)
		}
	}
	for _, w := range f.windows {
		if w.room() != "" {
			continue
		}
		tool, err := f.windowTool(ctx, doc, w)
		if err != nil {
			return nil, fmt.Errorf("window facing %v: %w", w.facing, err)
		}
		if cage, err = decad.Cut(ctx, cage, tool); err != nil {
			return nil, fmt.Errorf("cut window facing %v: %w", w.facing, err)
		}
	}
	return cage, nil
}

// tube is the ring between radii Ri and Ro, standing CageRise either side of the middle plane.
func (f sleeve) tube(ctx context.Context, doc *decad.Document) (*decad.Body, error) {
	w := sketch.NewWorld()
	s, err := w.CreateSketch(w.XY())
	if err != nil {
		return nil, fmt.Errorf("tube sketch: %w", err)
	}
	for _, r := range []float64{f.ri, f.ro} {
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), r), r))
	}
	if err := solveAndVerify(ctx, s, "tube"); err != nil {
		return nil, err
	}
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != 1 {
			continue
		}
		tube, err := decad.Extrude(ctx, doc, s, pr, decad.Distance{D: units.Millimeters(2 * f.zb), Dir: decad.Along})
		if err != nil {
			return nil, fmt.Errorf("extrude tube: %w", err)
		}
		down, err := r3.Translation(r3.NewVec(0, 0, -f.zb))
		if err != nil {
			return nil, err
		}
		return decad.Transform(ctx, tube, down)
	}
	return nil, fmt.Errorf("tube: no ring profile")
}

// boreTool is the bore's twisted channel: its opening rectangle swept along the gear's axis
// over the cut's span, turning at the ribbon's own rate.
func (f sleeve) boreTool(ctx context.Context, doc *decad.Document, b bore) (*decad.Body, error) {
	g := b.g
	s0, s1 := b.span(f)
	w := sketch.NewWorld()
	sk, err := w.CreateSketch(w.XY())
	if err != nil {
		return nil, fmt.Errorf("bore sketch: %w", err)
	}
	sin, cos := math.Sincos(g.angle(s0))
	corners := openingCorners(g, b.sign)
	points := make([]*sketch.Point, len(corners))
	for i, c := range corners {
		points[i] = sketchcheck.DimensionPoint(sk, c[0]*cos-c[1]*sin, c[0]*sin+c[1]*cos)
	}
	for i := range points {
		sk.CreateLine(points[i], points[(i+1)%len(points)])
	}
	if err := solveAndVerify(ctx, sk, "bore section"); err != nil {
		return nil, err
	}
	path, err := decad.NewPath(r3.Vec{}, decad.LineTo{End: r3.NewVec(0, 0, s1-s0)})
	if err != nil {
		return nil, fmt.Errorf("bore path: %w", err)
	}
	tool, err := decad.Sweep(ctx, doc, sk, sk.Profiles()[0], path,
		decad.WithSweepTwist(units.Radians((s1-s0)/g.p.Lambda())))
	if err != nil {
		return nil, fmt.Errorf("sweep bore: %w", err)
	}
	return placeOnFrame(ctx, tool, g.origin.Add(g.ez.Scale(s0)), g.ex, g.ey)
}

// windowTool is the window's hexagon pushed out through the wall in its facing direction.
func (f sleeve) windowTool(ctx context.Context, doc *decad.Document, win sleeveWindow) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.Vec{}, win.across, r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("window frame: %w", err)
	}
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("window plane: %w", err)
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("window sketch: %w", err)
	}
	points := make([]*sketch.Point, len(win.corners))
	for i, c := range win.corners {
		points[i] = sketchcheck.DimensionPoint(sk, c[0], c[1])
	}
	for i := range points {
		sk.CreateLine(points[i], points[(i+1)%len(points)])
	}
	if err := solveAndVerify(ctx, sk, "window"); err != nil {
		return nil, err
	}
	profiles := sk.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("window: want 1 valid profile, got %d", len(profiles))
	}
	return decad.Extrude(ctx, doc, sk, profiles[0], decad.Distance{D: units.Millimeters(f.ro + 1), Dir: decad.Along})
}
