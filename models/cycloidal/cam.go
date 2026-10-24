package cycloidal

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// regionWithHoles returns the only valid region of s with the given number of holes.
func regionWithHoles(s *sketch.Sketch, holes int, stage string) (*sketch.Profile, error) {
	var found *sketch.Profile
	for _, candidate := range s.Profiles() {
		if candidate.Valid && len(candidate.Holes) == holes {
			if found != nil {
				return nil, fmt.Errorf("%s: more than one region has %d holes", stage, holes)
			}
			found = candidate
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%s: no valid region with %d holes among %d regions", stage, holes, len(s.Profiles()))
	}
	return found, nil
}

// buildCam builds the eccentric cam. Each disc has a section whose outer circle is centered on that disc's
// center, so a two-disc cam is two cylinders 2E apart. The first section reaches up to the second disc's lower
// face and the second starts at the top of the first disc, so the sections overlap through the gap and join
// into one body; the input bore on the drive axis is cut through the whole cam last.
func (r *run) buildCam(ctx context.Context) (*decad.Body, error) {
	p := r.p
	var sections []*decad.Body
	for index := range p.DiscCount {
		bottom, top := 0.0, p.DiscThickness+p.DiscGap
		if p.DiscCount == 1 {
			top = p.DiscThickness
		}
		if index == 1 {
			bottom, top = p.DiscThickness, 2*p.DiscThickness+p.DiscGap
		}
		if p.DiscCount == 2 && p.DiscGap == 0 {
			// The sections stop short of the plane between the discs; the bridge joins them.
			if index == 0 {
				top -= camSeparation / 2
			} else {
				bottom += camSeparation / 2
			}
		}
		plane, err := r.planeAt(bottom)
		if err != nil {
			return nil, fmt.Errorf("cycloidal: cam section %d plane: %w", index+1, err)
		}
		s, err := r.world.CreateSketch(plane)
		if err != nil {
			return nil, fmt.Errorf("cycloidal: cam section %d sketch: %w", index+1, err)
		}
		cx, _ := r.discGeometry(index)
		if _, err := r.circleAt(s, cx, 0, p.CenterBearingDiameter, "center_bearing_diameter"); err != nil {
			return nil, fmt.Errorf("cycloidal: cam section %d outer circle: %w", index+1, err)
		}
		stage := fmt.Sprintf("cycloidal: cam section %d", index+1)
		if err := solveAndVerify(ctx, s, stage); err != nil {
			return nil, err
		}
		profile, err := regionWithHoles(s, 0, stage)
		if err != nil {
			return nil, err
		}
		section, err := decad.Extrude(ctx, r.doc, s, profile,
			decad.Distance{D: units.Millimeters(top - bottom), Dir: decad.Along})
		if err != nil {
			return nil, fmt.Errorf("cycloidal: extrude cam section %d: %w", index+1, err)
		}
		sections = append(sections, section)
	}
	cam := sections[0]
	if len(sections) > 1 {
		if p.DiscGap == 0 {
			bridge, err := r.buildCamBridge(ctx)
			if err != nil {
				return nil, err
			}
			sections = append(sections, bridge)
		}
		var err error
		cam, err = decad.Union(ctx, sections[0], sections[1:]...)
		if err != nil {
			return nil, fmt.Errorf("cycloidal: join cam sections: %w", err)
		}
	}
	if p.InputShaftDiameter == 0 {
		return cam, nil
	}
	// The bore tool starts one plane gap below the cam and ends one above, so no tool face lies in a cam face.
	plane, err := r.planeAt(-planeGap)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: input bore plane: %w", err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: input bore sketch: %w", err)
	}
	if _, err := r.circleAt(s, 0, 0, p.InputShaftDiameter, "input_shaft_diameter"); err != nil {
		return nil, fmt.Errorf("cycloidal: input bore circle: %w", err)
	}
	if err := solveAndVerify(ctx, s, "cycloidal: input bore"); err != nil {
		return nil, err
	}
	profile, err := regionWithHoles(s, 0, "cycloidal: input bore")
	if err != nil {
		return nil, err
	}
	height := float64(p.DiscCount)*p.DiscThickness + float64(p.DiscCount-1)*p.DiscGap + 2*planeGap
	tool, err := decad.Extrude(ctx, r.doc, s, profile, decad.Distance{D: units.Millimeters(height), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: extrude input bore: %w", err)
	}
	bored, err := decad.Cut(ctx, cam, tool)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: cut input bore: %w", err)
	}
	return bored, nil
}

// With no gap between the discs, the two cam sections stop camSeparation apart, centered on the plane between the
// discs, and a bridge reaches camOverlap into each. Decad decides a union only when the bodies interpenetrate
// deeper than its 0.1 mm chord tolerance, so the overlap is above that, and the separation keeps the sections'
// end faces from touching within it.
const (
	camSeparation = 0.25
	camOverlap    = 0.2
)

// buildCamBridge builds the cylinder that joins the two cam sections when the discs touch. With no gap the
// sections only share the plane between the discs, and decad does not decide a union of bodies that share a
// face. The bridge is centered on the drive axis, straddles that plane, and reaches into both sections, so each
// union has a volume in common. Its radius is halfway between the input bore and the largest circle about the
// axis that fits inside both sections, so a bore cut afterwards leaves material around it.
func (r *run) buildCamBridge(ctx context.Context) (*decad.Body, error) {
	p := r.p
	half := camSeparation/2 + camOverlap
	plane, err := r.planeAt(p.DiscThickness - half)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: cam bridge plane: %w", err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("cycloidal: cam bridge sketch: %w", err)
	}
	inner := p.CenterBearingDiameter/2 - p.Eccentricity
	diameter := inner + p.InputShaftDiameter/2
	circle := s.CreateCircle(s.Origin(), diameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, diameter))
	if err := solveAndVerify(ctx, s, "cycloidal: cam bridge"); err != nil {
		return nil, err
	}
	profile, err := regionWithHoles(s, 0, "cycloidal: cam bridge")
	if err != nil {
		return nil, err
	}
	bridge, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(2 * half), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cycloidal: extrude cam bridge: %w", err)
	}
	return bridge, nil
}
