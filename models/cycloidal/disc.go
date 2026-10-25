package cycloidal

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
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
	bore, err := r.circleAt(s, cx, 0, p.CenterBearingDiameter+p.BearingClearance, "disc_bore_diameter")
	if err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d center bore: %w", index+1, err)
	}
	// One lobe, valley to valley, patterned about the disc center. The profile runs clockwise as its curve
	// parameter grows, and the pattern turns counterclockwise, so the lobe is drawn from its last point back to
	// its first.
	segments := lobeSegments(p, d, turnLimit)
	lobe := make([][]point, len(segments))
	for i, segment := range segments {
		run := make([]point, len(segment))
		for j, t := range segment {
			run[len(segment)-1-j] = discPoint(p, d, t, cx, 0, phi)
		}
		lobe[len(segments)-1-i] = run
	}
	if err := drawPatternedLoop(s, bore.Center, lobe, d.lobes); err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d lobes: %w", index+1, err)
	}
	// One output hole at the disc's clocking angle, repeated about the disc center.
	hole, err := r.circleAt(s, cx+d.outputRadius*math.Cos(phi), d.outputRadius*math.Sin(phi),
		d.outputHoleDiameter, "output_hole_diameter")
	if err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d output hole: %w", index+1, err)
	}
	if _, err := sketch.CreatePatternCircular(s, []sketch.Entity{hole}, bore.Center, p.OutputPinCount); err != nil {
		return nil, fmt.Errorf("cycloidal: disc %d output holes: %w", index+1, err)
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
