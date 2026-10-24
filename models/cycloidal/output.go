package cycloidal

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/lestrrat-3d/decad"
)

// outputPinCenter returns the center of output pin j on the output pin circle about the drive axis.
func (r *run) outputPinCenter(j int) point {
	angle := 2 * math.Pi * float64(j) / float64(r.p.OutputPinCount)
	return point{r.d.outputRadius * math.Cos(angle), r.d.outputRadius * math.Sin(angle)}
}

// buildOutput builds the output plate above the disc stack, with a socket for each pin, and the M output pins.
// Each pin runs from the top of the plate down to the bottom of the first disc, through every disc's output
// holes.
func (r *run) buildOutput(ctx context.Context) (*decad.Body, []*decad.Body, error) {
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
	for j := range p.OutputPinCount {
		c := r.outputPinCenter(j)
		if _, err := r.circleAt(ps, c.x, c.y, r.d.outputPinDiameter, "resolved_output_pin_diameter"); err != nil {
			return nil, nil, fmt.Errorf("cycloidal: output plate socket %d: %w", j+1, err)
		}
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
	// The pins: one circle per pin, each extruded from the plate's lower face up through the plate and down to
	// the first disc's lower face.
	pins, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: output pin sketch: %w", err)
	}
	for j := range p.OutputPinCount {
		c := r.outputPinCenter(j)
		if _, err := r.circleAt(pins, c.x, c.y, r.d.outputPinDiameter, "resolved_output_pin_diameter"); err != nil {
			return nil, nil, fmt.Errorf("cycloidal: output pin %d circle: %w", j+1, err)
		}
	}
	if err := solveAndVerify(ctx, pins, "cycloidal: output pins"); err != nil {
		return nil, nil, err
	}
	height, err := r.doc.Param("output_plate_height")
	if err != nil {
		return nil, nil, fmt.Errorf("cycloidal: read output plate height: %w", err)
	}
	var bodies []*decad.Body
	for _, profile := range pins.Profiles() {
		if !profile.Valid || len(profile.Holes) != 0 {
			return nil, nil, fmt.Errorf("cycloidal: output pins: unexpected region")
		}
		body, err := decad.Extrude(ctx, r.doc, pins, profile, decad.TwoSided{
			One: decad.DistanceSide{D: thickness},
			Two: decad.DistanceSide{D: height},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("cycloidal: extrude output pin: %w", err)
		}
		bodies = append(bodies, body)
	}
	if len(bodies) != p.OutputPinCount {
		return nil, nil, fmt.Errorf("cycloidal: got %d output pins, want %d", len(bodies), p.OutputPinCount)
	}
	// Order the pins by their angle about the drive axis, so pin k is the one at angle 2πk/M.
	angles := make(map[*decad.Body]float64, len(bodies))
	for _, body := range bodies {
		c, err := body.Centroid()
		if err != nil {
			return nil, nil, fmt.Errorf("cycloidal: measure output pin: %w", err)
		}
		offset := c.Value.Sub(r.frame.Origin())
		angle := math.Atan2(offset.Dot(r.frame.V()), offset.Dot(r.frame.U()))
		if angle < -1e-9 {
			angle += 2 * math.Pi
		}
		angles[body] = angle
	}
	sort.Slice(bodies, func(i, j int) bool { return angles[bodies[i]] < angles[bodies[j]] })
	return plate, bodies, nil
}
