package cycloidal

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// run holds what every construction step of one Build shares.
type run struct {
	doc   *decad.Document
	world *sketch.World
	p     Params
	d     dimensions
	frame r3.Frame
}

func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// begin validates p and defines its parameters in the document and in a sketch world.
func begin(ctx context.Context, doc *decad.Document, p Params) (*run, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cycloidal: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("cycloidal: nil document")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, fmt.Errorf("cycloidal: invalid placement frame")
	}
	if err := Validate(p); err != nil {
		return nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: placement: %w", err)
	}
	world := sketch.NewWorld()
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cycloidal: document parameters: %w", err)
	}
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, fmt.Errorf("cycloidal: world parameters: %w", err)
	}
	return &run{doc: doc, world: world, p: p, d: resolve(p), frame: frame}, nil
}

// planeAt returns the sketch plane lifted by height along the placement normal.
func (r *run) planeAt(height float64) (*sketch.Plane, error) {
	if height == 0 && r.p.Placement == nil {
		return r.world.XY(), nil
	}
	frame, err := r3.NewFrame(r.frame.Origin().Add(r.frame.N().Scale(height)), r.frame.U(), r.frame.V())
	if err != nil {
		return nil, err
	}
	return r.world.CreatePlaneFromFrame(frame)
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

// circleAt adds a circle centered at (x, y), dimensioned from the sketch origin, and binds its diameter to
// the named parameter expression.
func (r *run) circleAt(s *sketch.Sketch, x, y, diameter float64, expr string) (*sketch.Circle, error) {
	center := sketchcheck.DimensionPoint(s, x, y)
	circle := s.CreateCircle(center, diameter/2)
	dim := sketch.NewDiameter(circle, diameter)
	s.AddConstraint(dim)
	if err := s.Bind(dim, r.world.Params(), expr); err != nil {
		return nil, fmt.Errorf("bind %q: %w", expr, err)
	}
	return circle, nil
}

// discGeometry returns the disc's center offset along the placement U axis and its clocking, for disc index d.
func (r *run) discGeometry(index int) (cx, phi float64) {
	if index == 0 {
		return r.p.Eccentricity, 0
	}
	return -r.p.Eccentricity, math.Pi
}

// discHeight is the height of disc index's lower face above the placement plane.
func (r *run) discHeight(index int) float64 {
	return float64(index) * (r.p.DiscThickness + r.p.DiscGap)
}
