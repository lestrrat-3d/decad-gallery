package cycloidal

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// buildDisc builds rotor disc index: one extruded profile whose outer loop is the L lobes, with the enlarged
// center bore and the M output holes as inner loops. Disc 1 is disc 0 turned half a turn about the drive axis.
func (r *run) buildDisc(ctx context.Context, index int) (*decad.Body, error) {
	var err error
	for _, limit := range lobeTurnLimits {
		var body *decad.Body
		body, err = r.buildDiscSampled(ctx, index, limit)
		if !errors.Is(err, decad.ErrUnsupported) {
			return body, err
		}
	}
	return nil, err
}

// buildDiscSampled builds disc index with its lobes sampled at the given turn limit.
func (r *run) buildDiscSampled(ctx context.Context, index int, turnLimit float64) (*decad.Body, error) {
	p, d := r.p, r.d
	plane, err := r.planeAt(r.discHeight(index))
	if err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d plane: %w", index+1, err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d sketch: %w", index+1, err)
	}
	cx, phi := r.discGeometry(index)
	segments := lobeSegments(p, d, turnLimit)
	span := 2 * math.Pi / float64(d.lobes)
	var first, shared *sketch.Point
	for k := range d.lobes {
		for i, segment := range segments {
			points := make([]*sketch.Point, len(segment))
			for j, t := range segment {
				switch {
				case j == 0 && shared != nil:
					points[j] = shared
				case i == len(segments)-1 && j == len(segment)-1 && k == d.lobes-1:
					points[j] = first
				default:
					q := discPoint(p, d, t+float64(k)*span, cx, 0, phi)
					points[j] = sketchcheck.DimensionPoint(s, q.x, q.y)
				}
			}
			if first == nil {
				first = points[0]
			}
			shared = points[len(points)-1]
			if _, err := s.CreateFitSpline(points...); err != nil {
				return nil, fmt.Errorf("cycloidal: disc %d lobe %d part %d spline: %w", index+1, k+1, i+1, err)
			}
		}
	}
	if _, err := r.circleAt(s, cx, 0, p.CenterBearingDiameter+p.BearingClearance, "disc_bore_diameter"); err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d center bore: %w", index+1, err)
	}
	for j := range p.OutputPinCount {
		angle := phi + 2*math.Pi*float64(j)/float64(p.OutputPinCount)
		x := cx + d.outputRadius*math.Cos(angle)
		y := d.outputRadius * math.Sin(angle)
		if _, err := r.circleAt(s, x, y, d.outputHoleDiameter, "output_hole_diameter"); err != nil {
			return nil, fmt.Errorf("cycloidal: disc %d output hole %d: %w", index+1, j+1, err)
		}
	}
	if err := solveAndVerify(ctx, s, fmt.Sprintf("cycloidal: disc %d", index+1)); err != nil {
		return nil, err
	}
	// Each hole is also a region of its own; the disc is the one region with every hole.
	var profile *sketch.Profile
	for _, candidate := range s.Profiles() {
		if candidate.Valid && len(candidate.Holes) == 1+p.OutputPinCount {
			if profile != nil {
				return nil, fmt.Errorf("cycloidal: disc %d: more than one region has %d holes", index+1, 1+p.OutputPinCount)
			}
			profile = candidate
		}
	}
	if profile == nil {
		return nil, fmt.Errorf("cycloidal: disc %d: no valid region with %d holes among %d regions",
			index+1, 1+p.OutputPinCount, len(s.Profiles()))
	}
	thickness, err := r.doc.Param("disc_thickness")
	if err != nil {
		return nil, fmt.Errorf("cycloidal: read disc thickness: %w", err)
	}
	body, err := decad.Extrude(ctx, r.doc, s, profile, decad.Distance{D: thickness, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: extrude disc %d: %w", index+1, err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("cycloidal: disc %d is not a solid", index+1)
	}
	return body, nil
}
