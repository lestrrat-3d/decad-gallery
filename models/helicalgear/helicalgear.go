// Package helicalgear builds gears with involute teeth twisted between two axial sections.
package helicalgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
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
// The spur pipeline builds it about the local origin and +Z, then moves it onto
// Params.Placement, the reference point and plane, because the tooth Unions fail on
// bodies built at an offset; a nil Placement leaves it on the world XY plane. Only the tooth, which is
// lofted between the lower section and the twisted section one Thickness above,
// and the root fillet, whose radius is scaled by cos(HelixAngle), differ from spur.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	return Variant(p, "thickness", LoftTooth).Build(ctx, doc, p.Params)
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

// Variant returns the spur pipeline changes of a helical gear. planeOffset is the
// parameter expression for the distance from the lower cap to the twisted section,
// and tooth builds tooth zero. Herringbone gears pass half the thickness.
func Variant(p Params, planeOffset string, tooth func(context.Context, spurgear.ToothInput) (*decad.Body, error)) spurgear.Variant {
	return spurgear.Variant{
		Define: func(t *param.Table) error {
			if err := t.SetValue("helix_angle", units.Radians(p.HelixAngle)); err != nil {
				return fmt.Errorf("parameter helix_angle: %w", err)
			}
			if err := t.SetExpr("helix_plane_offset", planeOffset, units.Millimeter); err != nil {
				return fmt.Errorf("parameter helix_plane_offset: %w", err)
			}
			if err := t.SetExpr("helical_fillet_radius", "fillet_radius * cos(helix_angle)", units.Millimeter); err != nil {
				return fmt.Errorf("parameter helical_fillet_radius: %w", err)
			}
			return nil
		},
		Tooth:      tooth,
		LocalBuild: true,
		RootFillet: filletRootEdges,
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
