package cycloidal

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
)

// splitAtInflections splits an ordered polyline into runs that each bend one way, so a fit spline through a run
// has one sign of curvature. Neighboring runs share the point at the cut.
func splitAtInflections(points []point) [][]point {
	var runs [][]point
	start := 0
	sign := 0.0
	for i := 1; i < len(points)-1; i++ {
		a, b, c := points[i-1], points[i], points[i+1]
		turn := (b.x-a.x)*(c.y-b.y) - (b.y-a.y)*(c.x-b.x)
		if turn == 0 {
			continue
		}
		if sign != 0 && (turn > 0) != (sign > 0) {
			runs = append(runs, points[start:i+1])
			start = i
		}
		sign = turn
	}
	return append(runs, points[start:])
}

// contourStrides are the strides at which a run of the contour is thinned, first choice first: a fit spline
// through every sampled point wiggles in curvature where the contour is nearly straight, which decad cannot
// certify as one sign, so the build tries the next stride when decad refuses.
var contourStrides = []int{3, 2, 4, 6, 1}

// thinned keeps every stride-th point of run and always its last.
func thinned(run []point, stride int) []point {
	var out []point
	for i := 0; i < len(run); i += stride {
		out = append(out, run[i])
	}
	if last := run[len(run)-1]; out[len(out)-1] != last {
		out = append(out, last)
	}
	return out
}

// buildHousing builds the housing as one prism and two cuts. The prism is the outer cylinder from the underside
// of the base to the top of the disc stack. The first cut removes everything inside the casing's swept contour
// from the top of the base up, which leaves the casing wall and the base floor; the second cuts the base's
// center hole. No two bodies share a face, which decad does not decide, and the result is the base ring and
// the casing in one body.
func (r *run) buildHousing(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	stack := float64(p.DiscCount)*p.DiscThickness + float64(p.DiscCount-1)*p.DiscGap
	bottom := -planeGap - p.BaseThickness

	// The outer cylinder.
	outerPlane, err := r.planeAt(bottom)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: housing outer plane: %w", err)
	}
	os, err := r.world.CreateSketch(outerPlane)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: housing outer sketch: %w", err)
	}
	if _, err := r.circleAt(os, 0, 0, d.housingOuter, "housing_outer_diameter"); err != nil {
		return nil, fmt.Errorf("cycloidal: housing outer circle: %w", err)
	}
	if err := solveAndVerify(ctx, os, "cycloidal: housing outer"); err != nil {
		return nil, err
	}
	outerProfile, err := regionWithHoles(os, 0, "cycloidal: housing outer")
	if err != nil {
		return nil, err
	}
	prism, err := decad.Extrude(ctx, r.doc, os, outerProfile,
		decad.Distance{D: units.Millimeters(stack - bottom), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: extrude housing outer: %w", err)
	}

	var cavity *decad.Body
	for _, stride := range contourStrides {
		cavity, err = r.buildCavity(ctx, stride)
		if !errors.Is(err, decad.ErrUnsupported) {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	walled, err := decad.Cut(ctx, prism, cavity)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: cut casing cavity: %w", err)
	}

	// The base's center hole, from below the housing up into the cavity.
	holePlane, err := r.planeAt(bottom - 1)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: housing hole plane: %w", err)
	}
	hs, err := r.world.CreateSketch(holePlane)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: housing hole sketch: %w", err)
	}
	if _, err := r.circleAt(hs, 0, 0, d.housingInner, "housing_inner_diameter"); err != nil {
		return nil, fmt.Errorf("cycloidal: housing hole circle: %w", err)
	}
	if err := solveAndVerify(ctx, hs, "cycloidal: housing hole"); err != nil {
		return nil, err
	}
	holeProfile, err := regionWithHoles(hs, 0, "cycloidal: housing hole")
	if err != nil {
		return nil, err
	}
	hole, err := decad.Extrude(ctx, r.doc, hs, holeProfile,
		decad.Distance{D: units.Millimeters(-bottom + 1), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: extrude housing hole: %w", err)
	}
	housing, err := decad.Cut(ctx, walled, hole)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: cut housing hole: %w", err)
	}
	return housing, nil
}

// buildCavity extrudes the space inside the casing's contour, from the top of the base to past the stack top,
// with the contour's points thinned by stride.
func (r *run) buildCavity(ctx context.Context, stride int) (*decad.Body, error) {
	p, d := r.p, r.d
	stack := float64(p.DiscCount)*p.DiscThickness + float64(p.DiscCount-1)*p.DiscGap
	cavityPlane, err := r.planeAt(-planeGap)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: casing plane: %w", err)
	}
	cs, err := r.world.CreateSketch(cavityPlane)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: casing sketch: %w", err)
	}
	// One pin pitch of the contour, patterned about the drive axis.
	pitch := make([][]point, 0)
	for _, run := range splitAtInflections(housingContour(p, d)) {
		pitch = append(pitch, thinned(run, stride))
	}
	if err := drawPatternedLoop(cs, cs.Origin(), pitch, p.PinCount); err != nil {
		return nil, fmt.Errorf("cycloidal: casing pitches: %w", err)
	}
	if err := solveAndVerify(ctx, cs, "cycloidal: casing"); err != nil {
		return nil, err
	}
	cavityProfile, err := regionWithHoles(cs, 0, "cycloidal: casing")
	if err != nil {
		return nil, err
	}
	cavity, err := decad.Extrude(ctx, r.doc, cs, cavityProfile,
		decad.Distance{D: units.Millimeters(stack + planeGap + 1), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: extrude casing cavity: %w", err)
	}
	return cavity, nil
}
