// Package helicalgear builds gears with involute teeth twisted between two axial sections.
package helicalgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	gearplacement "github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch/param"
)

// Params adds a top-section twist to the spur gear dimensions.
// Angles are radians and lengths are millimeters.
type Params struct {
	spurgear.Params
	HelixAngle float64
}

// DefaultParams returns the spur defaults with the source's 14.5-degree twist.
func DefaultParams() Params {
	return Params{Params: spurgear.DefaultParams(), HelixAngle: 14.5 * math.Pi / 180}
}

// ValidateHelixAngle accepts a positive twist from 1 through 45 degrees.
// The angle is in radians.
func ValidateHelixAngle(angle float64) error {
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return fmt.Errorf("helix angle must be finite")
	}
	if angle == 0 {
		return fmt.Errorf("zero helix angle is a spur gear; use spurgear.Build")
	}
	if angle < math.Pi/180 || angle > math.Pi/4 {
		return fmt.Errorf("helix angle must be from 1 to 45 degrees")
	}
	return nil
}

// Build adds a helical gear to doc. ctx and doc must not be nil.
// The whole toothed outline is lofted to the same outline turned by HelixAngle one Thickness above it, and the
// 2N root seams are rounded with the spur root fillet radius scaled by cos(HelixAngle); see BuildFilletedLoft.
// BoreDiameter then cuts a centered through-bore and CapChamfer chamfers the outer cap loops. The gear is built
// about the local origin and +Z and moved onto Params.Placement, the reference point and plane, with one Transform;
// a nil Placement leaves it on the world XY plane.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, fmt.Errorf("invalid placement frame")
	}
	body, err := BuildFilletedLoft(ctx, doc, p, "thickness")
	if err != nil {
		return nil, err
	}
	body, err = Finish(ctx, doc, body, p)
	if err != nil {
		return nil, err
	}
	return gearplacement.Place(ctx, body, p.Placement)
}

// Finish cuts the through-bore and chamfers the outer cap loops of a gear built about the local origin, as
// Params.BoreDiameter and Params.CapChamfer ask. The bore tool reads the document's spur parameters, which
// BuildFilletedLoft has defined.
func Finish(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	var err error
	local := p.Params
	local.Placement = nil
	if local.BoreDiameter > 0 {
		if gear, err = spurgear.CutThroughBore(ctx, doc, gear, local); err != nil {
			return nil, err
		}
	}
	if local.CapChamfer > 0 {
		if gear, err = spurgear.ChamferOuterCaps(ctx, gear, local.CapChamfer); err != nil {
			return nil, err
		}
	}
	return gear, nil
}

// BuildTooth lofts tooth zero alone, without the root disc or the pattern. ctx and doc must not be nil.
func BuildTooth(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	return Variant(p, "thickness", LoftTooth).BuildTooth(ctx, doc, p.Params)
}

// Variant returns the spur pipeline changes of a helical tooth, which BuildTooth uses. The gear itself is
// built by BuildFilletedLoft and takes no part in the spur pipeline. planeOffset is the
// parameter expression for the distance from the lower cap to the twisted section,
// and tooth builds tooth zero. Herringbone gears pass half the thickness.
func Variant(p Params, planeOffset string, tooth func(context.Context, spurgear.ToothInput) (*decad.Body, error)) spurgear.Variant {
	return spurgear.Variant{
		Define:     func(t *param.Table) error { return defineHelical(t, p, planeOffset) },
		Tooth:      tooth,
		LocalBuild: true,
	}
}

// LoftTooth lofts the tooth section on the lower cap to the same section twisted by
// helix_angle on the plane helix_plane_offset above it.
func LoftTooth(ctx context.Context, in spurgear.ToothInput) (*decad.Body, error) {
	offset, err := in.Doc.Param("helix_plane_offset")
	if err != nil {
		return nil, fmt.Errorf("read helix plane offset: %w", err)
	}
	helix, err := in.Doc.Param("helix_angle")
	if err != nil {
		return nil, fmt.Errorf("read helix angle: %w", err)
	}
	upper, err := in.PlaneAt(offset)
	if err != nil {
		return nil, fmt.Errorf("create twisted section plane: %w", err)
	}
	lowerSketch, lowerProfile, err := spurgear.BuildToothSection(ctx, in.World, in.Lower, in.Params, 0)
	if err != nil {
		return nil, fmt.Errorf("build lower tooth section: %w", err)
	}
	upperSketch, upperProfile, err := spurgear.BuildToothSection(ctx, in.World, upper, in.Params, helix.Base())
	if err != nil {
		return nil, fmt.Errorf("build twisted tooth section: %w", err)
	}
	tooth, err := decad.Loft(ctx, in.Doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("loft helical tooth: %w", err)
	}
	return tooth, nil
}
