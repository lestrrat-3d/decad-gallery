package planetarygearbox

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// buildSun builds the sun gear about the local origin, from z=0 to the gear thickness.
func (r *run) buildSun(ctx context.Context) (*decad.Body, error) {
	p := r.p
	sun, err := spurgear.Build(ctx, r.doc, spurgear.Params{
		Module: p.Module, Teeth: p.SunTeeth, PressureAngle: p.PressureAngle,
		Thickness: p.GearThickness, Samples: p.Samples, BoreDiameter: p.ShaftDiameter,
	})
	if err != nil {
		return nil, fmt.Errorf("sun gear: %w", err)
	}
	return sun, nil
}

// buildPlanet builds a planet gear about the local origin, from z=0 to the gear thickness.
func (r *run) buildPlanet(ctx context.Context) (*decad.Body, error) {
	p := r.p
	planet, err := spurgear.Build(ctx, r.doc, spurgear.Params{
		Module: p.Module, Teeth: p.PlanetTeeth, PressureAngle: p.PressureAngle,
		Thickness: p.GearThickness, Samples: p.Samples, BoreDiameter: r.d.planetBore,
	})
	if err != nil {
		return nil, fmt.Errorf("planet gear: %w", err)
	}
	return planet, nil
}

// ringTool is the spur pipeline with the ring's radii: the external gear whose teeth are the
// ring's tooth spaces. Its root circle is the ring's tip circle and its tip circle is the ring's
// root circle. It has no root fillet.
func ringTool() spurgear.Variant {
	return spurgear.Variant{Define: func(t *param.Table) error {
		for _, e := range []struct{ name, expr string }{
			{"root_radius", "pitch_radius - module"},
			{"tip_radius", "pitch_radius + 1.25 * module"},
			{"fillet_radius", "0"},
		} {
			if err := t.SetExpr(e.name, e.expr, units.Millimeter); err != nil {
				return fmt.Errorf("define parameter %s: %w", e.name, err)
			}
		}
		return nil
	}}
}

// ringMargin is how far the tooth-space tool extends past each end of the ring blank.
const ringMargin = 1

// buildRing builds the housing: an outer cylinder with the tooth spaces cut through it. It is
// built about the local origin from z=-StageGap to the ring length, with its tooth-space tool
// turned to the planets' phase.
func (r *run) buildRing(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	tool, err := ringTool().Build(ctx, r.doc, spurgear.Params{
		Module: p.Module, Teeth: p.RingTeeth, PressureAngle: p.PressureAngle,
		Thickness: d.ringLength + 2*ringMargin, Samples: p.Samples,
	})
	if err != nil {
		return nil, fmt.Errorf("ring tooth spaces: %w", err)
	}
	if angle := RingToothAngle(p); angle != 0 {
		turn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(angle))
		if err != nil {
			return nil, fmt.Errorf("ring tooth phase: %w", err)
		}
		if tool, err = decad.Transform(ctx, tool, turn); err != nil {
			return nil, fmt.Errorf("turn ring tooth spaces: %w", err)
		}
	}
	blank, err := r.extrudeDisc(ctx, disc{
		what: "ring blank", z: ringMargin, length: d.ringLength, diameter: 2 * d.ringOuter,
		expr: "2 * ring_outer_radius",
	})
	if err != nil {
		return nil, err
	}
	ring, err := decad.Cut(ctx, blank, tool)
	if err != nil {
		return nil, fmt.Errorf("cut ring tooth spaces: %w", err)
	}
	if !ring.IsSolid() {
		return nil, fmt.Errorf("ring is not a solid")
	}
	return ring, nil
}

// ringShift is the move from the ring's construction height to its place: the blank is built
// ringMargin above the origin and belongs StageGap below it.
func (r *run) ringShift() float64 { return -ringMargin - r.p.StageGap }
