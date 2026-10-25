package cycloidal

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

// outputPinCenter returns the center of output pin j on the output pin circle about the drive axis.
func (r *run) outputPinCenter(j int) point {
	angle := 2 * math.Pi * float64(j) / float64(r.p.OutputPinCount)
	return point{r.d.outputRadius * math.Cos(angle), r.d.outputRadius * math.Sin(angle)}
}

// buildOutput builds the output plate above the disc stack, with a socket for each pin, and output pin 1.
// The pin runs from the top of the plate down to the bottom of the first disc, through every disc's output
// holes.
func (r *run) buildOutput(ctx context.Context) (*decad.Body, *decad.Body, error) {
	p := r.p
	plateHeight := float64(p.DiscCount)*p.DiscThickness + float64(p.DiscCount-1)*p.DiscGap + planeGap
	plane, err := r.planeAt(plateHeight)
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output plate plane: %w", err)
	}
	// The plate: the plate circle with a socket circle per pin.
	ps, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output plate sketch: %w", err)
	}
	if _, err := r.circleAt(ps, 0, 0, r.d.outputPlateDiameter, "output_plate_diameter"); err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output plate circle: %w", err)
	}
	if err := r.patternPinCircle(ps); err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output plate sockets: %w", err)
	}
	if err := solveAndVerify(ctx, ps, "cycloidal: output plate"); err != nil {
		return nil, nil, err
	}
	plateProfile, err := regionWithHoles(ps, p.OutputPinCount, "cycloidal: output plate")
	if err != nil {
		return nil, nil, err
	}
	thickness, err := r.doc.Param("output_plate_thickness")
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: read output plate thickness: %w", err)
	}
	plate, err := decad.Extrude(ctx, r.doc, ps, plateProfile, decad.Distance{D: thickness, Dir: decad.Along})
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: extrude output plate: %w", err)
	}
	// The pin: one circle, extruded from the plate's lower face up through the plate and down to the first
	// disc's lower face. patternPins repeats the finished pin.
	pins, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output pin sketch: %w", err)
	}
	c := r.outputPinCenter(0)
	if _, err := r.circleAt(pins, c.x, c.y, r.d.outputPinDiameter, "resolved_output_pin_diameter"); err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output pin circle: %w", err)
	}
	if err := solveAndVerify(ctx, pins, "cycloidal: output pin"); err != nil {
		return nil, nil, err
	}
	height, err := r.doc.Param("output_plate_height")
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: read output plate height: %w", err)
	}
	var seed *sketch.Profile
	for _, profile := range pins.Profiles() {
		if !profile.Valid || len(profile.Holes) != 0 || seed != nil {
			return nil, nil, fmt.Errorf("cycloidal: output pin: unexpected region")
		}
		seed = profile
	}
	if seed == nil {
		return nil, nil, fmt.Errorf("cycloidal: output pin: no region")
	}
	pin, err := decad.Extrude(ctx, r.doc, pins, seed, decad.TwoSided{
		One: decad.DistanceSide{D: thickness},
		Two: decad.DistanceSide{D: height},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: extrude output pin: %w", err)
	}
	return plate, pin, nil
}

// patternPinCircle makes the output pin circles of one sketch: pin 0's circle on +X, repeated by a circular
// pattern about the sketch origin, the drive axis. The seed keeps the dimensions, so the copies follow it.
func (r *run) patternPinCircle(s *sketch.Sketch) error {
	c := r.outputPinCenter(0)
	seed, err := r.circleAt(s, c.x, c.y, r.d.outputPinDiameter, "resolved_output_pin_diameter")
	if err != nil {
		return err
	}
	if _, err := sketch.CreatePatternCircular(s, []sketch.Entity{seed}, s.Origin(), r.p.OutputPinCount); err != nil {
		return fmt.Errorf("pattern: %w", err)
	}
	return nil
}

// patternPins returns the M output pins, pin k at angle 2πk/M about the drive axis: pin, then its copies from a
// circular pattern about the placement's axis.
func (r *run) patternPins(ctx context.Context, pin *decad.Body) ([]*decad.Body, error) {
	copies, err := decad.PatternCopies(ctx, pin, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.p.OutputPinCount,
	})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: pattern output pins: %w", err)
	}
	return append([]*decad.Body{pin}, copies...), nil
}
