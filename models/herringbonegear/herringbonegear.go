// Package herringbonegear builds double-helical gears with mirrored tooth halves.
package herringbonegear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
)

// Params uses the same dimensions and helix angle as a helical gear.
type Params = helicalgear.Params

// DefaultParams returns the inherited spur dimensions and 14.5-degree twist.
func DefaultParams() Params { return helicalgear.DefaultParams() }

// variant is the helical pipeline with the twisted section at half the thickness
// and a mirrored tooth.
func variant(p Params) spurgear.Variant {
	return helicalgear.Variant(p, "thickness / 2", chevron)
}

// Build adds a herringbone gear to doc. ctx and doc must not be nil.
// The spur pipeline builds it about the local origin and +Z, then moves it onto
// Params.Placement, the reference point and plane, because the tooth Unions fail on
// bodies built at an offset; a nil Placement leaves it on the world XY plane.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := helicalgear.ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	return variant(p).Build(ctx, doc, p.Params)
}

// BuildTooth lofts one half, mirrors it, and joins both halves into a chevron.
// ctx and doc must not be nil. The tooth is built in the local frame; Params.Placement is ignored.
func BuildTooth(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := helicalgear.ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	return variant(p).BuildTooth(ctx, doc, p.Params)
}

// chevron lofts the lower half tooth, mirrors it across the plane helix_plane_offset
// above the lower cap, and joins the halves.
func chevron(ctx context.Context, in spurgear.ToothInput) (*decad.Body, error) {
	half, err := helicalgear.LoftTooth(ctx, in)
	if err != nil {
		return nil, err
	}
	offset, err := in.Doc.Param("helix_plane_offset")
	if err != nil {
		return nil, fmt.Errorf("read mid-body offset: %w", err)
	}
	mid, err := in.Shifted(offset)
	if err != nil {
		return nil, fmt.Errorf("create mid-body plane: %w", err)
	}
	mirrored, err := decad.MirroredCopy(ctx, half, decad.MirrorFrame{Frame: mid})
	if err != nil {
		return nil, fmt.Errorf("mirror half tooth: %w", err)
	}
	tooth, err := decad.Union(ctx, half, mirrored)
	if err != nil {
		return nil, fmt.Errorf("join tooth halves: %w", err)
	}
	return tooth, nil
}
