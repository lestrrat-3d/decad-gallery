// Package spurgear builds straight-toothed involute gears with decad.
package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Params holds the dimensions of one spur gear. Angles are radians; lengths are millimeters.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
}

// DefaultParams returns the values from instructions.md.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

func (p Params) validate() error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("module and thickness must be positive; teeth and samples must be at least 3")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle must lie between 0 and pi/2 radians")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer cannot be negative")
	}
	if p.BoreDiameter > 0 {
		return fmt.Errorf("through-bore is not implemented")
	}
	if p.CapChamfer > 0 {
		return fmt.Errorf("cap chamfer is not implemented")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("gear radii are invalid")
	}
	if d.base < d.root {
		return fmt.Errorf("embedded flanks require root-circle trimming")
	}
	radius := filletRadius(p, d)
	if radius <= 0 || math.IsNaN(radius) || math.IsInf(radius, 0) {
		return fmt.Errorf("root fillet radius is not positive")
	}
	leg := math.Sqrt(d.root*d.root + 2*d.root*radius)
	if leg >= d.base {
		return fmt.Errorf("root fillet reaches the involute flank")
	}
	gap := 2*math.Pi/float64(p.Teeth) - 2*flankRotation(p, d) - 2*math.Atan2(radius, leg)
	if gap <= 0 {
		return fmt.Errorf("adjacent root fillets leave no root arc")
	}
	return nil
}

func filletRadius(p Params, d dimensions) float64 {
	angle := math.Pi/float64(p.Teeth) - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)
	return 0.9 * d.root * angle / 2
}

// Build adds one gear to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	d := derive(p)
	world := sketch.NewWorld()
	frame, err := r3.NewFrame(r3.NewVec(0, 0, p.Thickness), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create upper gear frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create upper gear plane: %w", err)
	}
	lowerSketch, lowerProfile, err := buildGearProfile(ctx, world, world.XY(), p, d)
	if err != nil {
		return nil, fmt.Errorf("lower gear profile: %w", err)
	}
	upperSketch, upperProfile, err := buildGearProfile(ctx, world, upper, p, d)
	if err != nil {
		return nil, fmt.Errorf("upper gear profile: %w", err)
	}
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("loft gear profile: %w", err)
	}
	return body, nil
}

// rootFillet joins a radial tooth flank to the root circle with a circular arc.
// sign is -1 on a right flank and +1 on a left flank, toward the root gap.
func rootFillet(unit point, root, radius, sign float64) (point, point, point) {
	leg := math.Sqrt(root*root + 2*root*radius)
	normal := point{-unit.y, unit.x}
	center := point{leg*unit.x + sign*radius*normal.x, leg*unit.y + sign*radius*normal.y}
	onRoot := point{root * center.x / (root + radius), root * center.y / (root + radius)}
	onLine := point{leg * unit.x, leg * unit.y}
	return center, onRoot, onLine
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	center := s.CreatePoint(0, 0)
	s.Fix(center)
	fixed := func(xy point) *sketch.Point {
		pt := s.CreatePoint(xy.x, xy.y)
		s.Fix(pt)
		return pt
	}
	radius := filletRadius(p, d)
	rotation := flankRotation(p, d)
	var firstRoot, previousRoot *sketch.Point
	for tooth := range p.Teeth {
		angle := float64(tooth) * 2 * math.Pi / float64(p.Teeth)
		left, right := toothFlanks(p, d, angle)
		leftPoints := make([]*sketch.Point, len(left))
		rightPoints := make([]*sketch.Point, len(right))
		for i := range left {
			leftPoints[i] = fixed(left[i])
			rightPoints[i] = fixed(right[i])
		}
		rightUnit := rotate(point{1, 0}, angle-rotation)
		leftUnit := rotate(point{1, 0}, angle+rotation)
		rightCenter, rightRootXY, rightLineXY := rootFillet(rightUnit, d.root, radius, -1)
		leftCenter, leftRootXY, leftLineXY := rootFillet(leftUnit, d.root, radius, +1)
		rightRoot, leftRoot := fixed(rightRootXY), fixed(leftRootXY)
		rightLine, leftLine := fixed(rightLineXY), fixed(leftLineXY)
		if previousRoot != nil {
			s.CreateArc(center, previousRoot, rightRoot)
		} else {
			firstRoot = rightRoot
		}
		// Sketch arcs sweep counter-clockwise. Both minor fillet arcs are
		// recorded opposite to the profile's boundary traversal.
		s.CreateArc(fixed(rightCenter), rightLine, rightRoot)
		s.CreateLine(rightLine, rightPoints[0])
		if _, err := s.CreateFitSpline(rightPoints...); err != nil {
			return nil, nil, fmt.Errorf("tooth %d right flank: %w", tooth, err)
		}
		s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
		if _, err := s.CreateFitSpline(leftPoints...); err != nil {
			return nil, nil, fmt.Errorf("tooth %d left flank: %w", tooth, err)
		}
		s.CreateLine(leftPoints[0], leftLine)
		s.CreateArc(fixed(leftCenter), leftRoot, leftLine)
		previousRoot = leftRoot
	}
	s.CreateArc(center, previousRoot, firstRoot)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve gear profile: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("gear sketch has %d profiles; want one valid region", len(profiles))
	}
	return s, profiles[0], nil
}
