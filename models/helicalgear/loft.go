package helicalgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// DefineParameters writes the spur parameters of p and the helical ones into t: helix_angle, the distance
// helix_plane_offset from the lower cap to the twisted section (planeOffset is its expression over the spur
// names, "thickness" for a helical gear and "thickness / 2" for a herringbone half), and helical_fillet_radius,
// the spur fillet radius scaled by cos(helix_angle).
func DefineParameters(t *param.Table, p Params, planeOffset string) error {
	if err := spurgear.DefineParameters(t, p.Params); err != nil {
		return err
	}
	return defineHelical(t, p, planeOffset)
}

// defineHelical writes the helical parameter entries into t, after the spur ones.
func defineHelical(t *param.Table, p Params, planeOffset string) error {
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
}

// rejectEmbedded rejects a tooth whose base circle lies inside its root circle. The whole-outline loft below
// fillets the ordinary profile, whose roots are the ends of radial lines.
func rejectEmbedded(p Params) error {
	n := float64(p.Teeth)
	base := p.Module * n / 2 * math.Cos(p.PressureAngle)
	root := (p.Module*n - 2.5*p.Module) / 2
	if base < root {
		return fmt.Errorf("the base circle %v lies inside the root circle %v (embedded flanks are not supported)", base, root)
	}
	return nil
}

// BuildFilletedLoft builds the toothed outline of the whole gear on the local frame (origin, +X, +Y, +Z), lofts
// it to the same outline turned by HelixAngle on the plane helix_plane_offset above, and rounds every root
// seam with helical_fillet_radius. The result runs from z=0 to z=helix_plane_offset and has neither a bore nor a
// chamfer. planeOffset is the expression of helix_plane_offset; it must define a positive distance.
// Params.Placement is ignored. ctx and doc must not be nil.
//
// The outline has one tooth profile per tooth, so there is no root disc, no tooth pattern and no union. The
// lower outline is drawn at twist 0 and the upper one with every flank turned by HelixAngle in sketch coordinates,
// not by turning its plane. Each root contact of the lower outline is joined by the loft to the same contact
// turned by HelixAngle; those 2N edges are the ones filleted.
func BuildFilletedLoft(ctx context.Context, doc *decad.Document, p Params, planeOffset string) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	local := p
	local.Placement = nil
	if err := rejectEmbedded(local); err != nil {
		return nil, err
	}
	world := sketch.NewWorld()
	for name, t := range map[string]*param.Table{"document": doc.Params(), "sketch": world.Params()} {
		if err := DefineParameters(t, local, planeOffset); err != nil {
			return nil, fmt.Errorf("define %s parameters: %w", name, err)
		}
	}
	offset, err := doc.Param("helix_plane_offset")
	if err != nil {
		return nil, fmt.Errorf("read helix plane offset: %w", err)
	}
	helix, err := doc.Param("helix_angle")
	if err != nil {
		return nil, fmt.Errorf("read helix angle: %w", err)
	}
	height := offset.Base()
	frame, err := r3.NewFrame(r3.NewVec(0, 0, height), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("upper section frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("upper section plane: %w", err)
	}
	lowerSketch, lowerProfile, lowerRoots, err := spurgear.BuildTwistedProfile(ctx, world, world.XY(), local.Params, 0)
	if err != nil {
		return nil, fmt.Errorf("lower outline: %w", err)
	}
	upperSketch, upperProfile, upperRoots, err := spurgear.BuildTwistedProfile(ctx, world, upper, local.Params, helix.Base())
	if err != nil {
		return nil, fmt.Errorf("twisted outline: %w", err)
	}
	if len(lowerRoots) != len(upperRoots) {
		return nil, fmt.Errorf("outlines have %d and %d root contacts", len(lowerRoots), len(upperRoots))
	}
	loft, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("loft helical outline: %w", err)
	}
	radius, err := doc.Param("helical_fillet_radius")
	if err != nil {
		return nil, fmt.Errorf("read helical fillet radius: %w", err)
	}
	if radius.Base() <= 0 {
		return loft, nil
	}
	var seams *decad.EdgeQuery
	for i := range lowerRoots {
		start := r3.NewVec(lowerRoots[i][0], lowerRoots[i][1], 0)
		end := r3.NewVec(upperRoots[i][0], upperRoots[i][1], height)
		if seams == nil {
			seams = decad.Edges(decad.EndpointAt(start), decad.EndpointAt(end))
			continue
		}
		seams.Or(decad.EndpointAt(start), decad.EndpointAt(end))
	}
	rounded, err := decad.Fillet(ctx, loft, seams.Exactly(len(lowerRoots)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet helical root seams: %w", err)
	}
	return rounded, nil
}
