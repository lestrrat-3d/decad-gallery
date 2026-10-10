package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// BuildBoredBlank cuts a centered through-bore from the reference blank.
// ctx and doc must not be nil; BoreDiameter must be positive.
func BuildBoredBlank(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("bore diameter must be finite and positive")
	}
	blankParams := p
	blankParams.BoreDiameter = 0
	d, err := derive(blankParams)
	if err != nil {
		return nil, err
	}
	section, err := blankSection(blankParams, d)
	if err != nil {
		return nil, err
	}
	maxBore := 1.9 * math.Min(section[2].radius, section[4].radius)
	if p.BoreDiameter > maxBore {
		return nil, fmt.Errorf("bore diameter %.6g exceeds maximum %.6g mm", p.BoreDiameter, maxBore)
	}
	blank, err := BuildBlank(ctx, doc, blankParams)
	if err != nil {
		return nil, err
	}
	box, err := blank.Bounds()
	if err != nil {
		return nil, fmt.Errorf("measure blank bounds: %w", err)
	}
	const margin = 1.0
	world := sketch.NewWorld()
	frame, err := r3.NewFrame(r3.NewVec(box.Min.X-margin, 0, 0),
		r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("create bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create bore sketch: %w", err)
	}
	center := s.CreatePoint(0, 0)
	s.Fix(center)
	s.CreateCircle(center, p.BoreDiameter/2)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve bore sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("bore sketch has %d profiles; want one valid region", len(profiles))
	}
	tool, err := doc.Extrude(s, profiles[0], decad.Distance{
		D: units.Millimeters(box.Max.X - box.Min.X + 2*margin), Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	body, err := decad.Cut(ctx, blank, tool)
	if err != nil {
		return nil, fmt.Errorf("cut reference blank bore: %w", err)
	}
	return body, nil
}
