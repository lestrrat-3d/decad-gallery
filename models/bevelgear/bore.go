package bevelgear

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

// BuildBoredBlank cuts a centered through-bore from the reference blank.
// ctx and doc must not be nil; BoreDiameter must be positive.
func BuildBoredBlank(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := validateBore(p); err != nil {
		return nil, err
	}
	blankParams := p
	blankParams.BoreDiameter = 0
	blank, err := BuildBlank(ctx, doc, blankParams)
	if err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, blank, p.BoreDiameter)
}

func validateBore(p Params) error {
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return fmt.Errorf("bore diameter must be finite and positive")
	}
	maxBore, err := maximumBoreDiameter(p)
	if err != nil {
		return err
	}
	if p.BoreDiameter > maxBore {
		return fmt.Errorf("bore diameter %.6g exceeds maximum %.6g mm", p.BoreDiameter, maxBore)
	}
	return nil
}

// ResolveBoreDiameter resolves one member's enabled source bore. A zero
// diameter selects the smaller of one quarter of its pitch diameter and its
// blank's maximum bore diameter. The caller skips this when bores are disabled.
func ResolveBoreDiameter(p Params) (float64, error) {
	if !finite(p.BoreDiameter) || p.BoreDiameter < 0 {
		return 0, fmt.Errorf("bore diameter must be finite and nonnegative")
	}
	maxBore, err := maximumBoreDiameter(p)
	if err != nil {
		return 0, err
	}
	if p.BoreDiameter > maxBore {
		return 0, fmt.Errorf("bore diameter %.6g exceeds maximum %.6g mm", p.BoreDiameter, maxBore)
	}
	if p.BoreDiameter > 0 {
		return p.BoreDiameter, nil
	}
	return math.Min(p.Module*float64(p.Teeth)/4, maxBore), nil
}

func maximumBoreDiameter(p Params) (float64, error) {
	blankParams := p
	blankParams.BoreDiameter = 0
	blankParams.SpiralAngle = 0
	d, err := derive(blankParams)
	if err != nil {
		return 0, err
	}
	section, err := blankSection(blankParams, d)
	if err != nil {
		return 0, err
	}
	return 1.9 * math.Min(section[2].radius, section[4].radius), nil
}

func cutBore(ctx context.Context, doc *decad.Document, target *decad.Body, diameter float64) (*decad.Body, error) {
	box, err := target.Bounds()
	if err != nil {
		return nil, fmt.Errorf("measure bore target bounds: %w", err)
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
	circle := s.CreateCircle(s.Origin(), diameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, diameter))
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve bore sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify bore sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("bore sketch has %d profiles; want one valid region", len(profiles))
	}
	tool, err := decad.Extrude(ctx, doc, s, profiles[0], decad.Distance{
		D: units.Millimeters(box.Max.X - box.Min.X + 2*margin), Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	body, err := decad.Cut(ctx, target, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return body, nil
}
