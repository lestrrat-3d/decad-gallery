// Package herringbonegear builds double-helical gears with mirrored tooth halves.
package herringbonegear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Params uses the same dimensions and helix angle as a helical gear.
type Params = helicalgear.Params

// DefaultParams returns the inherited spur dimensions and 14.5-degree twist.
func DefaultParams() Params { return helicalgear.DefaultParams() }

// Build adds a herringbone gear to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	tooth, err := BuildTooth(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	blank, pattern, err := helicalgear.PreparePattern(ctx, doc, p, tooth)
	if err != nil {
		return nil, err
	}
	gear, err := decad.Union(ctx, blank, pattern...)
	if err != nil {
		return nil, fmt.Errorf("join tooth pattern to root disc: %w", err)
	}
	return helicalgear.FinishAssembly(ctx, doc, p, gear)
}

// BuildTooth lofts one half, mirrors it, and joins both halves into a chevron.
// ctx and doc must not be nil.
func BuildTooth(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := helicalgear.ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	world := sketch.NewWorld()
	midpoint := p.Thickness / 2
	frame, err := r3.NewFrame(r3.NewVec(0, 0, midpoint), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create mid-body plane: %w", err)
	}
	midPlane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create mid-body sketch plane: %w", err)
	}
	lowerSketch, lowerProfile, err := spurgear.BuildToothSection(ctx, world, world.XY(), p.Params, 0)
	if err != nil {
		return nil, fmt.Errorf("build lower tooth section: %w", err)
	}
	midSketch, midProfile, err := spurgear.BuildToothSection(ctx, world, midPlane, p.Params, p.HelixAngle)
	if err != nil {
		return nil, fmt.Errorf("build twisted mid-body section: %w", err)
	}
	half, err := doc.Loft(ctx, lowerSketch, lowerProfile, midSketch, midProfile)
	if err != nil {
		return nil, fmt.Errorf("loft half tooth: %w", err)
	}
	tooth, err := half.MirroredCopy(ctx, decad.MirrorFrame{Frame: frame})
	if err != nil {
		return nil, fmt.Errorf("mirror half tooth: %w", err)
	}
	tooth, err = decad.Union(ctx, half, tooth)
	if err != nil {
		return nil, fmt.Errorf("join tooth halves: %w", err)
	}
	return tooth, nil
}
