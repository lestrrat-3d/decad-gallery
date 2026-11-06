// Package herringbonegear builds double-helical gears with mirrored tooth halves.
package herringbonegear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	gearplacement "github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
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
// The lower half is the helical whole-outline loft to half the thickness with its root seams rounded, see
// helicalgear.BuildFilletedLoft. The upper half is its mirror image about the plane at half the thickness, and
// one Union joins the halves. BoreDiameter then cuts a centered through-bore and CapChamfer chamfers the outer
// cap loops. The gear is built about the local origin and +Z and moved onto Params.Placement with one
// Transform; a nil Placement leaves it on the world XY plane.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, fmt.Errorf("invalid placement frame")
	}
	half, err := helicalgear.BuildFilletedLoft(ctx, doc, p, "thickness / 2")
	if err != nil {
		return nil, err
	}
	offset, err := doc.Param("helix_plane_offset")
	if err != nil {
		return nil, fmt.Errorf("read mid-body offset: %w", err)
	}
	mid, err := r3.NewFrame(r3.NewVec(0, 0, offset.Base()), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create mid-body plane: %w", err)
	}
	mirrored, err := decad.MirroredCopy(ctx, half, decad.MirrorFrame{Frame: mid})
	if err != nil {
		return nil, fmt.Errorf("mirror half gear: %w", err)
	}
	gear, err := decad.Union(ctx, half, mirrored)
	if err != nil {
		return nil, fmt.Errorf("join gear halves: %w", err)
	}
	if gear, err = helicalgear.Finish(ctx, doc, gear, p); err != nil {
		return nil, err
	}
	return gearplacement.Place(ctx, gear, p.Placement)
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
